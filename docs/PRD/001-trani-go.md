# PRD: Trani - Go Reimplementation

## 1. Vision

Reimplement Trani CLI tool in Go, maintaining **exact feature parity** with current bash implementation while adding transcription backend flexibility (local whisper.cpp or OpenAI API).

**Philosophy:** Minimalist, lean code. Less is more.

## 2. Scope

### In Scope
- ✅ All current bash functionality preserved
- ✅ Local whisper.cpp transcription (default)
- ✅ OpenAI Whisper API transcription (opt-in)
- ✅ Claude API for LLM (only provider)
- ✅ YAML configuration at `~/.config/trani/config.yaml`
- ✅ Same CLI commands: `start`, `stop`, `toggle`
- ✅ Same workflow: neovim blocking, auto-processing on close
- ✅ Same directory structure and file naming

### Out of Scope
- Multiple LLM providers (Claude only)
- Local LLM support
- Web UI or daemon mode
- Audio backend alternatives (PipeWire/PulseAudio only)

## 3. Architecture

```
trani/
├── main.go
├── go.mod
├── cmd/
│   ├── root.go
│   ├── start.go
│   ├── stop.go
│   └── toggle.go
├── internal/
│   ├── audio/
│   │   └── recorder.go
│   ├── config/
│   │   └── config.go
│   ├── llm/
│   │   └── claude.go
│   ├── session/
│   │   └── session.go
│   └── transcribe/
│       ├── transcriber.go
│       ├── whisper.go
│       └── openai.go
└── pkg/
    └── notify/
        └── notify.go
```

## 4. Configuration

### 4.1. File Location
```
~/.config/trani/config.yaml
```

### 4.2. Structure
```yaml
transcription:
  backend: local  # "local" or "openai"
  
  local:
    model_path: ~/whisper.cpp/models/ggml-large-v3-turbo.bin
    binary_path: ~/whisper.cpp/build/bin/whisper-cli
    threads: 12
    language: es
  
  openai:
    model: whisper-1
    language: es

llm:
  claude:
    model: claude-sonnet-4-20250514
    max_tokens: 4000

audio:
  sample_rate: 16000
  channels: 1

paths:
  sessions_dir: ~/.config/trani/sessions  # Default if not specified
  temp_dir: ~/.config/trani/temp
  prompts_dir: ~/.config/trani/prompts
```

### 4.3. Environment Variables
```bash
OPENAI_API_KEY      # For OpenAI Whisper API
ANTHROPIC_API_KEY   # For Claude API (required)
```

### 4.4. Loading Priority
1. Environment variables (highest)
2. Config file
3. Hardcoded defaults (lowest)

### 4.5. Defaults
If `~/.config/trani/config.yaml` doesn't exist, or if `paths.sessions_dir` is not specified, use these defaults:

```go
// Default paths
sessions_dir: ~/.config/trani/sessions
temp_dir:     ~/.config/trani/temp
prompts_dir:  ~/.config/trani/prompts
```

**Priority:**
1. Explicit value in config file (highest)
2. Default path in `~/.config/trani/` (if not specified)
3. Environment variable expansion (~ to $HOME)

**Logic:**
```go
func (c *Config) ApplyDefaults() {
    configDir := filepath.Join(os.Getenv("HOME"), ".config", "trani")
    
    if c.Paths.SessionsDir == "" {
        c.Paths.SessionsDir = filepath.Join(configDir, "sessions")
    }
    if c.Paths.TempDir == "" {
        c.Paths.TempDir = filepath.Join(configDir, "temp")
    }
    if c.Paths.PromptsDir == "" {
        c.Paths.PromptsDir = filepath.Join(configDir, "prompts")
    }
}
```

## 5. Core Components

### 5.1. Config (`internal/config/config.go`)

```go
type Config struct {
    Transcription TranscriptionConfig
    LLM           LLMConfig
    Audio         AudioConfig
    Paths         PathsConfig
}

type TranscriptionConfig struct {
    Backend string
    Local   LocalWhisperConfig
    OpenAI  OpenAIConfig
}

type LocalWhisperConfig struct {
    ModelPath  string
    BinaryPath string
    Threads    int
    Language   string
}

type OpenAIConfig struct {
    Model    string
    Language string
}

type LLMConfig struct {
    Claude ClaudeConfig
}

type ClaudeConfig struct {
    Model     string
    MaxTokens int
}

type AudioConfig struct {
    SampleRate int
    Channels   int
}

type PathsConfig struct {
    SessionsDir string
    TempDir     string
    PromptsDir  string
}
```

**Functions:**
- `Load() (*Config, error)` - Load from file or defaults
- `(c *Config) ExpandPaths()` - Expand ~ to $HOME
- `(c *Config) ApplyDefaults()` - Set default paths if not specified

### 5.2. Transcriber (`internal/transcribe/`)

**Interface:**
```go
type Transcriber interface {
    Transcribe(ctx context.Context, audioPath string) (string, error)
}
```

**Factory:**
```go
func New(cfg config.TranscriptionConfig) (Transcriber, error)
```

**Implementations:**

#### `whisper.go`
```go
type WhisperLocal struct {
    modelPath  string
    binaryPath string
    threads    int
    language   string
}

func NewWhisperLocal(cfg config.LocalWhisperConfig) *WhisperLocal

func (w *WhisperLocal) Transcribe(ctx context.Context, audioPath string) (string, error)
```

Executes:
```bash
~/whisper.cpp/build/bin/whisper-cli \
  -m {model_path} \
  -f {audioPath} \
  -l {language} \
  -t {threads} \
  -otxt \
  -of {outputPath}
```

Reads `{outputPath}.txt` and returns content.

#### `openai.go`
```go
type OpenAI struct {
    apiKey   string
    model    string
    language string
    client   *http.Client
}

func NewOpenAI(cfg config.OpenAIConfig, apiKey string) *OpenAI

func (o *OpenAI) Transcribe(ctx context.Context, audioPath string) (string, error)
```

HTTP multipart upload to `https://api.openai.com/v1/audio/transcriptions`.

### 5.3. LLM (`internal/llm/claude.go`)

```go
type Claude struct {
    apiKey    string
    model     string
    maxTokens int
    client    *http.Client
}

func New(cfg config.ClaudeConfig, apiKey string) *Claude

func (c *Claude) Generate(ctx context.Context, prompt string) (string, error)
```

POST to `https://api.anthropic.com/v1/messages` with:
```json
{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 4000,
  "messages": [{"role": "user", "content": "..."}]
}
```

Return `response.content[0].text` or error message.

### 5.4. Audio Recorder (`internal/audio/recorder.go`)

```go
type Recorder struct {
    tempDir    string
    sampleRate int
    channels   int
    
    sinkModuleID    string
    loopMicModuleID string
    loopSysModuleID string
    recordingPID    int
}

func New(cfg config.AudioConfig, tempDir string) *Recorder

func (r *Recorder) Start(ctx context.Context) error

func (r *Recorder) Stop() error

func (r *Recorder) RecordingPath() string
```

**Start sequence:**
1. Load `module-null-sink` → save module ID
2. Load `module-loopback` for mic → save module ID
3. Load `module-loopback` for system → save module ID
4. Start `pw-record` → save PID
5. Return nil or error

**Stop sequence:**
1. Kill `pw-record` process
2. Unload all modules
3. Return nil or error

### 5.5. Session (`internal/session/session.go`)

```go
type Session struct {
    title          string
    path           string
    promptTemplate string
    preserveAudio  bool
    startedAt      time.Time
    
    recorder    *audio.Recorder
    transcriber transcribe.Transcriber
    llm         *llm.Claude
    notifier    *notify.Notifier
    cfg         *config.Config
}

type State struct {
    Active         bool
    Title          string
    StartedAt      time.Time
    Path           string
    PromptTemplate string
    PreserveAudio  bool
}

func New(title, promptTemplate string, preserveAudio bool, cfg *config.Config) (*Session, error)

func (s *Session) Start(ctx context.Context) error

func (s *Session) Stop(ctx context.Context) error

func LoadActive(cfg *config.Config) (*Session, error)

func (s *Session) SaveState() error

func (s *Session) ClearState() error
```

**Start workflow:**
1. Check if session already active → error if yes
2. Generate title if empty (format: `sesion_HH-MM`)
3. Create session directory: `sessions/YYYY-MM-DD-{title}/`
4. Initialize recorder, transcriber, llm
5. Start audio recording
6. Save state to `temp/current_session.json`
7. Notify "🎙️ Trani: Grabación iniciada - {title}"
8. Open `notas.md` in neovim (blocking)
9. When neovim closes → call `Stop()`

**Stop workflow:**
1. Load active session state → error if none
2. Stop audio recording
3. Notify "⏸️ Trani: Grabación detenida. Procesando..."
4. Move `temp/recording.wav` → `{sessionPath}/audio.wav`
5. Transcribe audio → `transcripcion.txt`
6. Check if `notas.md` has content
7. Load prompt template or use hardcoded fallback
8. Fill template with transcription and notes
9. Call Claude API → `resumen.md`
10. If API error → save error message in `resumen.md`
11. Delete `audio.wav` (unless preserve flag set)
12. Clear session state
13. Notify "✅ Trani: Sesión completada - {title}"

### 5.6. Notifier (`pkg/notify/notify.go`)

```go
type Notifier struct{}

func New() *Notifier

func (n *Notifier) Info(title, message string) error

func (n *Notifier) Error(title, message string) error
```

Executes `notify-send` with appropriate urgency.

## 6. CLI Commands

### 6.1. Root (`cmd/root.go`)

```go
var rootCmd = &cobra.Command{
    Use:   "trani",
    Short: "Audio recording with AI transcription and notes",
}
```

### 6.2. Start (`cmd/start.go`)

```
trani start [title] [--prompt TEMPLATE] [--preserve-audio]
```

**Flags:**
- `--prompt` (default: "default")
- `--preserve-audio` (default: false)

**Logic:**
1. Load config
2. Create new session
3. Call `session.Start(ctx)`

### 6.3. Stop (`cmd/stop.go`)

```
trani stop
```

**Logic:**
1. Load config
2. Load active session
3. Call `session.Stop(ctx)`

### 6.4. Toggle (`cmd/toggle.go`)

```
trani toggle [title] [--prompt TEMPLATE] [--preserve-audio]
```

**Logic:**
1. Load config
2. Try to load active session
3. If active → call `Stop()`
4. If not active → create new and call `Start()`

## 7. Prompt System

### 7.1. Template Files

**Location:** `~/trani/prompts/`

**Naming:**
- `{name}.txt` - With notes
- `{name}_no_notes.txt` - Without notes

**Variables:**
- `{{TRANSCRIPTION}}` - Full transcription
- `{{NOTES}}` - User notes

### 7.2. Loading Logic

```go
func (s *Session) loadPromptTemplate(hasNotes bool) (string, error)
```

1. Determine filename: `{template}.txt` or `{template}_no_notes.txt`
2. Try to read from `prompts_dir/{filename}`
3. If file doesn't exist → return hardcoded prompt
4. Replace `{{TRANSCRIPTION}}` and `{{NOTES}}`

### 7.3. Hardcoded Fallbacks

**With notes:**
```
Tienes una transcripción de una sesión y las notas tomadas por el usuario.

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

Mantén el formato limpio y profesional. Usa encabezados claros.
```

**Without notes:**
```
Tienes la transcripción de una sesión. Analízala y genera un documento estructurado.

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

Mantén el formato limpio y profesional.
```

## 8. Error Handling

### 8.1. Transcription Errors

**Local Whisper:**
- Binary not found → clear error message with path
- Model not found → clear error message with path
- Transcription failed → save stderr to `error.log`, notify user

**OpenAI API:**
- Missing API key → "OPENAI_API_KEY not set"
- HTTP errors → include status code and response body
- Network timeout → clear timeout message

### 8.2. LLM Errors

**Claude API:**
- Missing API key → "ANTHROPIC_API_KEY not set"
- API error response → extract `.error.message`, save to `resumen.md`
- Network issues → save error to `resumen.md`, notify user

### 8.3. Audio Errors

- Module loading fails → unload any loaded modules, return error
- Recording process fails → cleanup modules, return error
- Stop called with no active recording → ignore silently

### 8.4. Session Errors

- Start with active session → notify "Ya hay una sesión en curso: {title}", exit
- Stop with no active session → notify "No hay sesión activa", exit
- File system errors → propagate with context

## 9. File Structure

### 9.1. Session Directory

```
~/.config/trani/sessions/YYYY-MM-DD-{title}/
├── transcripcion.txt    # Whisper output
├── notas.md            # User notes (may be empty)
└── resumen.md          # Claude output or error message
```

**Note:** If user specifies custom `sessions_dir` in config, sessions are created there instead.

### 9.2. Temp Directory

```
~/.config/trani/temp/
├── recording.wav           # Active recording
├── current_session.json    # Session state
└── error.log              # Transcription errors (if any)
```

**Note:** Temp directory location is configurable via `paths.temp_dir`.

### 9.3. State File Format

```json
{
  "active": true,
  "title": "sprint_planning",
  "started_at": "2025-10-10T14:30:00-06:00",
  "path": "/home/user/.config/trani/sessions/2025-10-10-sprint_planning",
  "prompt_template": "default",
  "preserve_audio": false
}
```

**Note:** Session `path` reflects the configured `sessions_dir`.

## 10. Dependencies

### 10.1. Go Modules

```go
require (
    github.com/spf13/cobra v1.8.0
    gopkg.in/yaml.v3 v3.0.1
)
```

**No other external dependencies.** Use stdlib for HTTP, JSON, file I/O.

### 10.2. System Requirements

**Required:**
- `pipewire-pulse` (pactl)
- `pipewire-utils` (pw-record)
- `libnotify` (notify-send)
- `neovim`
- `curl`, `jq` (for debugging, not required by Go code)

**Conditional:**
- `whisper.cpp` compiled (if using local backend)
- OpenAI API key (if using openai backend)
- Claude API key (always required)

## 11. Build & Installation

### 11.1. Development

```bash
go run main.go start "test"
```

### 11.2. Build Binary

```bash
go build -o trani
```

### 11.3. Install

```bash
go install
# Or
cp trani ~/bin/
chmod +x ~/bin/trani
```

### 11.4. Release Build

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o trani
# Results in ~5-8MB binary
```

## 12. Testing Strategy

### 12.1. Unit Tests

- Config loading with missing file
- Config loading with invalid YAML
- Prompt template variable replacement
- Session state serialization/deserialization

### 12.2. Integration Tests

- Full workflow: start → record → stop → transcribe → summarize
- Error scenarios: missing API keys, invalid paths
- Toggle behavior: start when inactive, stop when active

### 12.3. Manual Testing Checklist

- [ ] Start session without title
- [ ] Start session with custom title
- [ ] Start with custom prompt template
- [ ] Start with --preserve-audio
- [ ] Stop active session
- [ ] Toggle when no session active
- [ ] Toggle when session active
- [ ] Transcription with local whisper.cpp
- [ ] Transcription with OpenAI API
- [ ] Claude API success
- [ ] Claude API error (invalid key)
- [ ] Empty notes file
- [ ] Notes file with content
- [ ] Session naming (date + title)
- [ ] Custom sessions_dir in config
- [ ] Default sessions_dir (~/.config/trani/sessions)
- [ ] Path expansion (~ to $HOME)

## 13. Migration Plan

### Phase 1: Core Framework (Day 1)
- [ ] Project structure setup
- [ ] Config loading with YAML
- [ ] Session state management
- [ ] CLI commands skeleton with Cobra

### Phase 2: Audio & Transcription (Day 2)
- [ ] Audio recorder implementation
- [ ] Local Whisper transcriber
- [ ] OpenAI transcriber
- [ ] Integration with session workflow

### Phase 3: LLM & Polish (Day 3)
- [ ] Claude API client
- [ ] Prompt template system
- [ ] Notifications
- [ ] Error handling refinement
- [ ] Testing

### Phase 4: Validation (Day 4)
- [ ] Side-by-side testing with bash version
- [ ] Fix any discrepancies
- [ ] Documentation
- [ ] Release build

## 14. Success Criteria

**Functional Parity:**
- ✅ All bash commands work identically in Go version
- ✅ Same file structure and naming conventions
- ✅ Same notifications and user feedback
- ✅ Same error messages and handling

**New Capabilities:**
- ✅ OpenAI Whisper API as alternative to local
- ✅ YAML configuration file support
- ✅ Single binary distribution

**Code Quality:**
- ✅ No unnecessary comments (only godoc)
- ✅ Descriptive function names
- ✅ Minimal dependencies (only cobra + yaml)
- ✅ Lean codebase (<1500 lines total)

**Performance:**
- ✅ CLI startup <10ms
- ✅ Binary size <10MB
- ✅ Memory usage <50MB during recording

## 15. Code Style Guidelines

### 15.1. Function Naming

Use descriptive names that explain intent:

✅ Good:
```go
func expandHomeDirectory(path string) string
func isSessionActive() bool
func loadPromptTemplateWithNotes(name string) (string, error)
```

❌ Avoid:
```go
func expand(p string) string  // Too vague
func check() bool             // What are we checking?
func load(n string) string    // Load what?
```

### 15.2. Error Messages

Be specific and actionable:

✅ Good:
```go
return fmt.Errorf("whisper binary not found at %s", w.binaryPath)
return fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
```

❌ Avoid:
```go
return errors.New("binary not found")
return errors.New("missing key")
```

### 15.3. Comments

Only use godoc for exported functions/types:

✅ Good:
```go
// New creates a configured Claude API client.
func New(cfg ClaudeConfig, apiKey string) *Claude

// Transcribe converts audio to text using local whisper.cpp.
func (w *WhisperLocal) Transcribe(ctx context.Context, path string) (string, error)
```

❌ Avoid:
```go
// Load modules
pactl load-module ...

// Check if error
if err != nil {
```

### 15.4. Code Organization

One logical operation per function. If a function does multiple things, split it:

✅ Good:
```go
func (s *Session) Start(ctx context.Context) error {
    if err := s.checkNotAlreadyActive(); err != nil {
        return err
    }
    if err := s.createDirectory(); err != nil {
        return err
    }
    if err := s.startRecording(ctx); err != nil {
        return err
    }
    if err := s.saveState(); err != nil {
        return err
    }
    s.notifyStarted()
    return s.openNotesInEditor()
}
```

### 15.5. Error Handling

Always handle errors immediately, don't accumulate:

✅ Good:
```go
resp, err := http.Post(url, body)
if err != nil {
    return fmt.Errorf("API request failed: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != 200 {
    return fmt.Errorf("API returned status %d", resp.StatusCode)
}
```

## 16. Non-Goals

**Explicitly NOT doing:**
- Multiple LLM provider support
- Local LLM integration (llama.cpp, etc.)
- GUI or web interface
- Plugin system
- Database for sessions
- Search functionality
- Alternative audio backends
- Windows or macOS support (Linux only for MVP)
- Configuration validation beyond basics
- Logging framework (use stdlib log)
- Metrics or telemetry

## 17. Future Considerations

**Post-MVP features to consider:**
- Local LLM support via llama.cpp HTTP server
- Multiple LLM provider abstraction
- Session search and filtering
- Export to other formats (PDF, DOCX)
- Audio quality settings (sample rate, bitrate)
- Diarization (speaker identification)
- Streaming transcription for long sessions

---

**Status:** Ready for implementation
**Estimated effort:** 3-4 days
**Target:** Single Go binary with feature parity to bash version
**Platforms:** Linux (Fedora) only
