package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/sabhz/trani/internal/audio"
	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/llm"
	"github.com/sabhz/trani/internal/transcribe"
	"github.com/sabhz/trani/pkg/notify"
)

// Session represents an active recording session.
type Session struct {
	title          string
	path           string
	promptTemplate string
	preserveAudio  bool
	startedAt      time.Time
	notifyID       string

	recorder    *audio.Recorder
	transcriber transcribe.Transcriber
	llm         llm.Generator
	notifier    *notify.Notifier
	cfg         *config.Config
}

// New creates a new session with the given parameters.
func New(promptTemplate string, preserveAudio bool, cfg *config.Config) (*Session, error) {
	timestamp := time.Now().Format("20060102-1504")
	sessionPath := filepath.Join(cfg.Paths.SessionsDir, timestamp)

	transcriber, err := transcribe.New(cfg.Transcription)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transcriber: %w", err)
	}

	recorder := audio.New(cfg.Audio, cfg.Paths.TempDir)

	llmClient, err := llm.New(cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	if err := ensureDefaultPrompts(cfg.Paths.PromptsDir); err != nil {
		return nil, fmt.Errorf("failed to initialize prompts: %w", err)
	}

	notifier := notify.New()

	return &Session{
		title:          timestamp,
		path:           sessionPath,
		promptTemplate: promptTemplate,
		preserveAudio:  preserveAudio,
		startedAt:      time.Now(),
		recorder:       recorder,
		transcriber:    transcriber,
		llm:            llmClient,
		notifier:       notifier,
		cfg:            cfg,
	}, nil
}

// createDirectory creates the session directory if it doesn't exist.
func (s *Session) createDirectory() error {
	if err := os.MkdirAll(s.path, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}
	return nil
}

// Start begins a new recording session. It blocks on the editor (nvim needs
// a terminal), but hands off transcription/summary post-processing to a
// detached worker so the recording lock clears the moment recording stops,
// letting a new session start immediately.
func (s *Session) Start(ctx context.Context) error {
	if lock, err := ReadLock(s.cfg); err != nil {
		return err
	} else if lock != nil {
		return fmt.Errorf("session already active: %s", lock.Title)
	}

	if err := s.createDirectory(); err != nil {
		return err
	}

	if err := os.MkdirAll(s.cfg.Paths.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := s.recorder.Start(ctx); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	message := fmt.Sprintf("Grabación iniciada - %s", s.title)
	notifyID, err := s.notifier.Start("🎙️ Trani", message)
	if err != nil {
		notifyID = ""
	}
	s.notifyID = notifyID

	lock := &RecordingLock{
		PID:            os.Getpid(),
		Title:          s.title,
		Path:           s.path,
		StartedAt:      s.startedAt,
		PromptTemplate: s.promptTemplate,
		PreserveAudio:  s.preserveAudio,
		NotifyID:       notifyID,
	}
	if err := lock.Save(s.cfg); err != nil {
		s.recorder.Stop()
		return err
	}

	notesPath := filepath.Join(s.path, "notas.md")
	editorCmd := exec.CommandContext(ctx, "nvim", notesPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Start(); err != nil {
		s.recorder.Stop()
		ClearLock(s.cfg)
		return fmt.Errorf("failed to start editor: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	editorDone := make(chan error, 1)
	go func() { editorDone <- editorCmd.Wait() }()

	select {
	case <-sigCh:
		editorCmd.Process.Signal(syscall.SIGTERM)
		<-editorDone
	case err := <-editorDone:
		if err != nil {
			s.recorder.Stop()
			ClearLock(s.cfg)
			return fmt.Errorf("editor exited with error: %w", err)
		}
	}

	if err := s.extractAndRenameIfNeeded(); err != nil {
		s.recorder.Stop()
		ClearLock(s.cfg)
		return fmt.Errorf("failed to rename session: %w", err)
	}

	return s.finishRecording()
}

// finishRecording stops the recorder, clears the recording lock so a new
// session can start, and hands off transcription/summary to a detached
// post-processing worker.
func (s *Session) finishRecording() error {
	if err := s.recorder.Stop(); err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	recordingPath := s.recorder.RecordingPath()
	audioPath := filepath.Join(s.path, "audio.wav")
	if err := os.Rename(recordingPath, audioPath); err != nil {
		return fmt.Errorf("failed to move audio file: %w", err)
	}

	if err := ClearLock(s.cfg); err != nil {
		return err
	}

	if s.notifyID != "" {
		s.notifier.Update(s.notifyID, "⏸️ Trani", "Grabación detenida. Procesando...")
	} else {
		s.notifier.Info("⏸️ Trani", "Grabación detenida. Procesando...")
	}

	return SpawnPostprocess(s.path, s.promptTemplate, s.preserveAudio, s.notifyID)
}

const defaultPromptWithNotes = `Tienes una transcripción de una sesión y las notas tomadas por el usuario.

TRANSCRIPCIÓN:
{{TRANSCRIPTION}}

NOTAS DEL USUARIO:
{{NOTES}}

Genera un documento markdown estructurado con:

1. RESUMEN EJECUTIVO (2-3 párrafos)
   - Contexto general de la sesión
   - Puntos clave discutidos
   - Conclusiones principales

2. DETALLES POR TEMA
   Usa los temas de las notas del usuario como estructura.
   Para cada tema identifica en la transcripción:
   - Detalles específicos mencionados
   - Datos, fechas, números relevantes
   - Procesos o procedimientos descritos
   - Decisiones tomadas
   - Contexto adicional importante

3. ACCIONES Y PENDIENTES
   - Action items identificados
   - Responsables (si se mencionan)
   - Fechas límite (si se mencionan)

4. DATOS IMPORTANTES
   - Fechas clave mencionadas
   - Números, métricas, estadísticas
   - Nombres de personas referenciadas
   - Documentos, sistemas o herramientas mencionadas

Mantén el formato limpio y profesional. Usa encabezados claros.`

const defaultPromptNoNotes = `Tienes la transcripción de una sesión. Analízala y genera un documento estructurado.

TRANSCRIPCIÓN:
{{TRANSCRIPTION}}

Genera un documento markdown con:

1. RESUMEN EJECUTIVO (2-3 párrafos)
   - Tema principal de la sesión
   - Puntos clave discutidos
   - Conclusiones principales

2. TEMAS PRINCIPALES
   Identifica los temas principales discutidos y para cada uno incluye:
   - Contexto y detalles
   - Puntos específicos mencionados
   - Decisiones o conclusiones

3. ACCIONES Y PENDIENTES
   - Action items identificados
   - Responsables (si se mencionan)
   - Fechas límite (si se mencionan)

4. DATOS IMPORTANTES
   - Fechas mencionadas
   - Números, métricas
   - Nombres de personas
   - Referencias a documentos/sistemas

Mantén el formato limpio y profesional.`

func ensureDefaultPrompts(promptsDir string) error {
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create prompts directory: %w", err)
	}

	defaultPath := filepath.Join(promptsDir, "default.txt")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		if err := os.WriteFile(defaultPath, []byte(defaultPromptWithNotes), 0644); err != nil {
			return fmt.Errorf("failed to write default.txt: %w", err)
		}
	}

	defaultNoNotesPath := filepath.Join(promptsDir, "default_no_notes.txt")
	if _, err := os.Stat(defaultNoNotesPath); os.IsNotExist(err) {
		if err := os.WriteFile(defaultNoNotesPath, []byte(defaultPromptNoNotes), 0644); err != nil {
			return fmt.Errorf("failed to write default_no_notes.txt: %w", err)
		}
	}

	return nil
}

// fillPromptTemplate replaces {{TRANSCRIPTION}} and {{NOTES}} placeholders.
func fillPromptTemplate(template, transcription, notes string) string {
	result := strings.ReplaceAll(template, "{{TRANSCRIPTION}}", transcription)
	result = strings.ReplaceAll(result, "{{NOTES}}", notes)
	return result
}

// postProcessAudio downsample to 16kHz mono and normalize audio.
func postProcessAudio(audioPath string) error {
	tempPath := audioPath + ".tmp.wav"

	cmd := exec.CommandContext(
		context.Background(),
		"sox",
		audioPath,
		"-r", "16000",
		"-c", "1",
		tempPath,
		"norm",
		"highpass", "80",
		"lowpass", "8000",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sox processing failed: %w", err)
	}

	if err := os.Rename(tempPath, audioPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to replace audio file: %w", err)
	}

	return nil
}

func slugify(text string) string {
	slug := strings.ToLower(text)
	slug = strings.ReplaceAll(slug, " ", "-")

	specialCharsRegex := regexp.MustCompile(`[^a-z0-9-]+`)
	slug = specialCharsRegex.ReplaceAllString(slug, "")

	multiHyphenRegex := regexp.MustCompile(`-+`)
	slug = multiHyphenRegex.ReplaceAllString(slug, "-")

	slug = strings.Trim(slug, "-")

	if len(slug) > 50 {
		runes := []rune(slug)
		if len(runes) > 50 {
			slug = string(runes[:50])
		}
	}

	slug = strings.Trim(slug, "-")

	return slug
}

func (s *Session) extractAndRenameIfNeeded() error {
	notesPath := filepath.Join(s.path, "notas.md")

	content, err := os.ReadFile(notesPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return nil
	}

	firstLine := lines[0]
	if !strings.HasPrefix(firstLine, "# ") {
		return nil
	}

	heading := strings.TrimPrefix(firstLine, "# ")
	heading = strings.TrimSpace(heading)

	if heading == "" {
		return nil
	}

	slug := slugify(heading)
	if slug == "" {
		return nil
	}

	timestamp := filepath.Base(s.path)
	newDirName := fmt.Sprintf("%s-%s", timestamp, slug)
	newPath := filepath.Join(filepath.Dir(s.path), newDirName)

	if err := os.Rename(s.path, newPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to rename session directory: %v\n", err)
		return nil
	}

	s.path = newPath
	s.title = newDirName
	return nil
}
