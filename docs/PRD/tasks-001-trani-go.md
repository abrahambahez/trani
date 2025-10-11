## Relevant Files

- `go.mod` - Go module definition with dependencies (cobra v1.10.1, yaml.v3)
- `go.sum` - Go dependency checksums
- `main.go` - Entry point that calls cmd.Execute()
- `cmd/root.go` - Cobra root command setup
- `cmd/start.go` - Start command with title arg and --prompt, --preserve-audio flags
- `cmd/stop.go` - Stop command to stop active session
- `cmd/toggle.go` - Toggle command with conditional start/stop logic
- `internal/config/config.go` - Configuration structs with YAML tags, Load(), ExpandPaths(), and ApplyDefaults() methods
- `internal/config/config_test.go` - Unit tests for config loading, path expansion, and defaults (6 tests, all passing)
- `internal/audio/recorder.go` - Recorder with Start() and Stop() methods for PipeWire audio recording lifecycle management
- `internal/audio/recorder_test.go` - Unit tests for recorder (5 tests, all passing)
- `internal/transcribe/transcriber.go` - Transcriber interface and New() factory function
- `internal/transcribe/whisper.go` - WhisperLocal implementation with Transcribe() executing whisper.cpp CLI
- `internal/transcribe/openai.go` - OpenAI implementation with multipart upload to OpenAI Whisper API
- `internal/transcribe/transcribe_test.go` - Unit tests for transcribers (15 tests, all passing)
- `internal/session/session.go` - Session and State structs with New(), SaveState(), ClearState(), and LoadActive() methods
- `internal/session/session_test.go` - Unit tests for session management
- `internal/llm/claude.go` - Claude API client stub (New() and Generate() - to be implemented in task 5.0)
- `internal/llm/claude_test.go` - Unit tests for Claude client
- `pkg/notify/notify.go` - Desktop notification wrapper using notify-send (Info and Error methods)

### Notes

- Unit tests should be placed alongside the code files they are testing
- Use `go test ./...` to run all tests in the project
- Use `go test -v ./internal/config` to run tests for a specific package
- The project follows Go standard layout with `cmd/`, `internal/`, and `pkg/` directories
- All internal packages are not importable by external projects

## Tasks

- [x] 1.0 Setup Go project structure and configuration system
  - [x] 1.1 Initialize Go module with `go mod init` and create directory structure (cmd/, internal/, pkg/)
  - [x] 1.2 Add dependencies: `go get github.com/spf13/cobra` and `go get gopkg.in/yaml.v3`
  - [x] 1.3 Create Config structs in internal/config/config.go (Config, TranscriptionConfig, LLMConfig, etc.)
  - [x] 1.4 Implement Load() function to read YAML from ~/.config/trani/config.yaml
  - [x] 1.5 Implement ExpandPaths() method to replace ~ with $HOME in all path fields
  - [x] 1.6 Implement ApplyDefaults() method to set default paths when not specified in config
  - [x] 1.7 Create notifier package in pkg/notify/notify.go with Info() and Error() methods
  - [x] 1.8 Write unit tests in internal/config/config_test.go for missing file, invalid YAML, and defaults

- [x] 2.0 Implement audio recording and PipeWire integration
  - [x] 2.1 [depends on: 1.0] Create Recorder struct in internal/audio/recorder.go with fields for module IDs and PID
  - [x] 2.2 Implement helper functions to execute pactl commands (load/unload modules)
  - [x] 2.3 Implement function to start pw-record process and capture PID
  - [x] 2.4 Implement Start() method that loads null-sink, loopback modules, and starts recording
  - [x] 2.5 Implement Stop() method that kills pw-record and unloads all modules
  - [x] 2.6 Implement RecordingPath() method to return temp recording file path
  - [x] 2.7 Write unit tests in internal/audio/recorder_test.go for module management logic

- [x] 3.0 Implement transcription backends (local Whisper and OpenAI)
  - [x] 3.1 [depends on: 1.0] Create Transcriber interface in internal/transcribe/transcriber.go
  - [x] 3.2 Implement New() factory function that returns appropriate transcriber based on config
  - [x] 3.3 Implement WhisperLocal struct and Transcribe() method in internal/transcribe/whisper.go
  - [x] 3.4 Implement OpenAI struct and Transcribe() method in internal/transcribe/openai.go with multipart upload
  - [x] 3.5 Add comprehensive error handling for missing binaries, models, and API keys
  - [x] 3.6 Write unit tests in internal/transcribe/transcribe_test.go for both implementations

- [x] 4.0 Implement session management and workflow orchestration
  - [x] 4.1 [depends on: 1.0, 2.0, 3.0] Create Session and State structs in internal/session/session.go
  - [x] 4.2 Implement New() constructor that initializes recorder, transcriber, and llm
  - [x] 4.3 Implement session directory creation with YYYY-MM-DD-title format
  - [x] 4.4 Implement SaveState() to write current_session.json and ClearState() to remove it
  - [x] 4.5 Implement LoadActive() to restore session from current_session.json
  - [x] 4.6 Implement Start() workflow: check active, create dir, start recording, save state, open neovim
  - [x] 4.7 [depends on: 5.0] Implement Stop() workflow: stop recording, transcribe, load prompt, call LLM, save results
  - [x] 4.8 Implement audio file cleanup logic that respects preserveAudio flag
  - [ ] 4.9 Write unit tests in internal/session/session_test.go for state serialization and workflow

- [ ] 5.0 Implement Claude API integration and prompt system
  - [x] 5.1 [depends on: 1.0] Create Claude struct in internal/llm/claude.go with API key and config
  - [x] 5.2 Implement New() constructor with ANTHROPIC_API_KEY validation
  - [x] 5.3 Implement Generate() method with HTTP POST to https://api.anthropic.com/v1/messages
  - [x] 5.4 Implement error response parsing and extract error.message from API responses
  - [x] 5.5 Add loadPromptTemplate() method to session that reads from prompts_dir
  - [x] 5.6 Implement variable replacement for {{TRANSCRIPTION}} and {{NOTES}} placeholders
  - [x] 5.7 Add hardcoded fallback prompts for with-notes and no-notes scenarios
  - [ ] 5.8 Write unit tests in internal/llm/claude_test.go for API client and prompt substitution

- [ ] 6.0 Implement CLI commands and complete integration testing
  - [x] 6.1 [depends on: 1.0] Setup Cobra root command in cmd/root.go with app description
  - [x] 6.2 [depends on: 4.0] Implement start command in cmd/start.go with title argument and flags
  - [x] 6.3 [depends on: 4.0] Implement stop command in cmd/stop.go to stop active session
  - [x] 6.4 [depends on: 4.0] Implement toggle command in cmd/toggle.go with conditional start/stop logic
  - [x] 6.5 Add persistent flags --prompt and --preserve-audio to relevant commands
  - [x] 6.6 Create main.go that calls cmd.Execute()
  - [ ] 6.7 [depends on: 5.0] Perform end-to-end integration testing with real audio recording
  - [ ] 6.8 Test all scenarios from PRD section 12.3 (custom titles, prompts, both transcription backends, etc.)
  - [x] 6.9 Build release binary with CGO_ENABLED=0 and verify <10MB size and <10ms startup time
