package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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

	recorder    *audio.Recorder
	transcriber transcribe.Transcriber
	llm         llm.Generator
	notifier    *notify.Notifier
	cfg         *config.Config
}

// State represents the serializable state of an active session.
type State struct {
	Active         bool      `json:"active"`
	Title          string    `json:"title"`
	StartedAt      time.Time `json:"started_at"`
	Path           string    `json:"path"`
	PromptTemplate string    `json:"prompt_template"`
	PreserveAudio  bool      `json:"preserve_audio"`
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

// Start begins a new recording session.
func (s *Session) Start(ctx context.Context) error {
	statePath := filepath.Join(s.cfg.Paths.TempDir, "current_session.json")
	if _, err := os.Stat(statePath); err == nil {
		existingState, err := os.ReadFile(statePath)
		if err == nil {
			var state State
			if json.Unmarshal(existingState, &state) == nil && state.Active {
				return fmt.Errorf("session already active: %s", state.Title)
			}
		}
	}

	if err := s.createDirectory(); err != nil {
		return err
	}

	if err := s.recorder.Start(ctx); err != nil {
		return fmt.Errorf("failed to start recording: %w", err)
	}

	if err := s.SaveState(); err != nil {
		s.recorder.Stop()
		return err
	}

	message := fmt.Sprintf("Grabación iniciada - %s", s.title)
	s.notifier.Info("🎙️ Trani", message)

	notesPath := filepath.Join(s.path, "notas.md")
	cmd := exec.CommandContext(ctx, "nvim", notesPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		s.recorder.Stop()
		return fmt.Errorf("editor exited with error: %w", err)
	}

	if err := s.extractAndRenameIfNeeded(); err != nil {
		s.recorder.Stop()
		return fmt.Errorf("failed to rename session: %w", err)
	}

	if err := s.SaveState(); err != nil {
		s.recorder.Stop()
		return fmt.Errorf("failed to save session state: %w", err)
	}

	return s.Stop(ctx)
}

// SaveState writes the current session state to temp/current_session.json.
func (s *Session) SaveState() error {
	state := State{
		Active:         true,
		Title:          s.title,
		StartedAt:      s.startedAt,
		Path:           s.path,
		PromptTemplate: s.promptTemplate,
		PreserveAudio:  s.preserveAudio,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	statePath := filepath.Join(s.cfg.Paths.TempDir, "current_session.json")
	if err := os.MkdirAll(s.cfg.Paths.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// ClearState removes the current_session.json file.
func (s *Session) ClearState() error {
	statePath := filepath.Join(s.cfg.Paths.TempDir, "current_session.json")
	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear state file: %w", err)
	}
	return nil
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

// LoadActive restores a session from current_session.json.
func LoadActive(cfg *config.Config) (*Session, error) {
	statePath := filepath.Join(cfg.Paths.TempDir, "current_session.json")

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no active session found")
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if !state.Active {
		return nil, fmt.Errorf("no active session found")
	}

	session, err := New(state.PromptTemplate, state.PreserveAudio, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct session: %w", err)
	}

	session.title = state.Title
	session.startedAt = state.StartedAt
	session.path = state.Path

	return session, nil
}

func (s *Session) loadPromptTemplate(hasNotes bool) string {
	suffix := ".txt"
	if !hasNotes {
		suffix = "_no_notes.txt"
	}

	filename := s.promptTemplate + suffix
	promptPath := filepath.Join(s.cfg.Paths.PromptsDir, filename)

	content, err := os.ReadFile(promptPath)
	if err != nil {
		defaultFilename := "default" + suffix
		defaultPath := filepath.Join(s.cfg.Paths.PromptsDir, defaultFilename)
		content, err = os.ReadFile(defaultPath)
		if err != nil {
			return "Error: No se pudo cargar la plantilla de prompt. Verifica la configuración."
		}
	}

	return string(content)
}

// fillPromptTemplate replaces {{TRANSCRIPTION}} and {{NOTES}} placeholders.
func fillPromptTemplate(template, transcription, notes string) string {
	result := strings.ReplaceAll(template, "{{TRANSCRIPTION}}", transcription)
	result = strings.ReplaceAll(result, "{{NOTES}}", notes)
	return result
}

// Stop stops the recording session and processes the audio.
func (s *Session) Stop(ctx context.Context) error {
	if err := s.recorder.Stop(); err != nil {
		return fmt.Errorf("failed to stop recording: %w", err)
	}

	s.notifier.Info("⏸️ Trani", "Grabación detenida. Procesando...")

	recordingPath := s.recorder.RecordingPath()
	audioPath := filepath.Join(s.path, "audio.wav")

	if err := os.Rename(recordingPath, audioPath); err != nil {
		return fmt.Errorf("failed to move audio file: %w", err)
	}

	if err := postProcessAudio(audioPath); err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	transcription, err := s.transcriber.Transcribe(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	transcriptionPath := filepath.Join(s.path, "transcripcion.txt")
	if err := os.WriteFile(transcriptionPath, []byte(transcription), 0644); err != nil {
		return fmt.Errorf("failed to save transcription: %w", err)
	}

	notesPath := filepath.Join(s.path, "notas.md")
	notesContent, _ := os.ReadFile(notesPath)
	notes := strings.TrimSpace(string(notesContent))
	hasNotes := len(notes) > 0

	template := s.loadPromptTemplate(hasNotes)
	prompt := fillPromptTemplate(template, transcription, notes)

	resumen, err := s.llm.Generate(ctx, prompt)
	resumenPath := filepath.Join(s.path, "resumen.md")

	if err != nil {
		errorMsg := fmt.Sprintf("Error al generar resumen: %v", err)
		os.WriteFile(resumenPath, []byte(errorMsg), 0644)
	} else {
		os.WriteFile(resumenPath, []byte(resumen), 0644)
	}

	if err := s.cleanupAudio(); err != nil {
		return err
	}

	if err := s.ClearState(); err != nil {
		return err
	}

	s.notifier.Info("✅ Trani", fmt.Sprintf("Sesión completada - %s", s.title))
	return nil
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

// cleanupAudio removes the audio file unless preserveAudio flag is set.
func (s *Session) cleanupAudio() error {
	if s.preserveAudio {
		return nil
	}

	audioPath := filepath.Join(s.path, "audio.wav")
	if err := os.Remove(audioPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove audio file: %w", err)
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

