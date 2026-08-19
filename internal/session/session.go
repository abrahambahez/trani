package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
	title          string // also the .sources/<title> basename
	notePath       string // sessions/<title>.md; the user's notes and the final summary are the same file
	promptTemplate string
	startedAt      time.Time
	notifyID       string

	recorder    *audio.Recorder
	transcriber transcribe.Transcriber
	llm         llm.Generator
	notifier    *notify.Notifier
	cfg         *config.Config
}

// Title returns the session's timestamp-based title (also the .sources/<title> basename).
func (s *Session) Title() string { return s.title }

// New creates a new session with the given parameters.
func New(promptTemplate string, cfg *config.Config) (*Session, error) {
	timestamp := time.Now().Format("2006-01-02 1504")
	notePath := filepath.Join(cfg.Paths.SessionsDir, timestamp+".md")

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
		notePath:       notePath,
		promptTemplate: promptTemplate,
		startedAt:      time.Now(),
		recorder:       recorder,
		transcriber:    transcriber,
		llm:            llmClient,
		notifier:       notifier,
		cfg:            cfg,
	}, nil
}

// Start begins a new recording session. It opens the note in Obsidian
// non-blocking and waits for an explicit stop signal, since Obsidian is a
// separate GUI app. It hands off transcription/summary post-processing to a
// detached worker so the recording lock clears the moment recording stops,
// letting a new session start immediately.
func (s *Session) Start(ctx context.Context) error {
	if err := os.MkdirAll(s.cfg.Paths.SessionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}

	if err := os.MkdirAll(s.cfg.Paths.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Acquired atomically and as early as possible: a plain
	// check-then-write has a race where two near-simultaneous invocations
	// can both start recording, with the loser silently orphaned (unable
	// to be signaled through the CLI, recording forever).
	lock := &RecordingLock{
		PID:            os.Getpid(),
		Title:          s.title,
		Path:           s.notePath,
		StartedAt:      s.startedAt,
		PromptTemplate: s.promptTemplate,
	}
	if err := lock.Acquire(s.cfg); err != nil {
		return err
	}

	if err := s.recorder.Start(ctx); err != nil {
		ClearLock(s.cfg)
		return fmt.Errorf("failed to start recording: %w", err)
	}

	chunker, err := newChunker(s.cfg, s.title, s.recorder, s.transcriber)
	if err != nil {
		s.recorder.Stop()
		ClearLock(s.cfg)
		return err
	}

	message := fmt.Sprintf("Grabación iniciada - %s", s.title)
	notifyID, err := s.notifier.Start("🎙️ Trani", message)
	if err != nil {
		notifyID = ""
	}
	s.notifyID = notifyID

	lock.NotifyID = notifyID
	if err := lock.update(s.cfg); err != nil {
		s.recorder.Stop()
		ClearLock(s.cfg)
		return err
	}

	chunkerStop := make(chan struct{})
	chunkerDone := make(chan struct{})
	go func() {
		defer close(chunkerDone)
		chunker.run(ctx, chunkerStop)
	}()
	stopChunker := func() {
		close(chunkerStop)
		<-chunkerDone
	}

	if _, err := os.Stat(s.notePath); os.IsNotExist(err) {
		os.WriteFile(s.notePath, nil, 0644)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := openNote(ctx, s.notePath, s.cfg); err != nil {
		stopChunker()
		s.recorder.Stop()
		ClearLock(s.cfg)
		return fmt.Errorf("failed to open note: %w", err)
	}

	<-sigCh
	stopChunker()

	return s.finishRecording(ctx, chunker)
}

// finishRecording stops the recorder and clears the recording lock right
// away, before doing anything that could take a while (transcribing the
// last chunk, summarizing), so a new session can start immediately instead
// of waiting on a network call. It then does one last pass to pick up
// whatever chunk was still open and hands off summarization to a detached
// post-processing worker.
func (s *Session) finishRecording(ctx context.Context, chunker *chunker) error {
	if err := s.recorder.Stop(); err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	if err := ClearLock(s.cfg); err != nil {
		return err
	}

	if err := chunker.pollOnce(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "trani: chunk processing error: %v\n", err)
	}

	if s.notifyID != "" {
		s.notifier.Update(s.notifyID, "⏸️ Trani", "Grabación detenida. Procesando...")
	} else {
		s.notifier.Info("⏸️ Trani", "Grabación detenida. Procesando...")
	}

	return SpawnPostprocess(s.notePath, s.title, s.promptTemplate, s.notifyID)
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

// runSox invokes sox with the given arguments.
func runSox(args ...string) error {
	output, err := exec.CommandContext(context.Background(), "sox", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sox failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// postProcessAudio downsample to 16kHz mono and normalize audio.
func postProcessAudio(audioPath string) error {
	tempPath := audioPath + ".tmp.wav"

	if err := runSox(audioPath, "-r", "16000", "-c", "1", tempPath, "norm", "highpass", "80", "lowpass", "8000"); err != nil {
		return err
	}

	if err := os.Rename(tempPath, audioPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to replace audio file: %w", err)
	}

	return nil
}

