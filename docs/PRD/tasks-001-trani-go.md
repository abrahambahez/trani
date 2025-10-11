## Relevant Files

- `main.go` - Entry point for the CLI application
- `go.mod` - Go module definition with dependencies (cobra, yaml)
- `cmd/root.go` - Cobra root command setup
- `cmd/start.go` - Implementation of start command
- `cmd/stop.go` - Implementation of stop command
- `cmd/toggle.go` - Implementation of toggle command
- `internal/config/config.go` - Configuration structs and loading logic
- `internal/config/config_test.go` - Unit tests for config package
- `internal/audio/recorder.go` - PipeWire audio recording implementation
- `internal/audio/recorder_test.go` - Unit tests for audio recorder
- `internal/transcribe/transcriber.go` - Transcriber interface and factory
- `internal/transcribe/whisper.go` - Local whisper.cpp implementation
- `internal/transcribe/openai.go` - OpenAI Whisper API implementation
- `internal/transcribe/transcribe_test.go` - Unit tests for transcribers
- `internal/session/session.go` - Session management and workflow orchestration
- `internal/session/session_test.go` - Unit tests for session management
- `internal/llm/claude.go` - Claude API client implementation
- `internal/llm/claude_test.go` - Unit tests for Claude client
- `pkg/notify/notify.go` - Desktop notification wrapper

### Notes

- Unit tests should be placed alongside the code files they are testing
- Use `go test ./...` to run all tests in the project
- Use `go test -v ./internal/config` to run tests for a specific package
- The project follows Go standard layout with `cmd/`, `internal/`, and `pkg/` directories
- All internal packages are not importable by external projects

## Tasks

- [ ] 1.0 Setup Go project structure and configuration system
  - [ ] 1.1 Initialize Go module with `go mod init` and create directory structure (cmd/, internal/, pkg/)
  - [ ] 1.2 Add dependencies: `go get github.com/spf13/cobra` and `go get gopkg.in/yaml.v3`
  - [ ] 1.3 Create Config structs in internal/config/config.go (Config, TranscriptionConfig, LLMConfig, etc.)
  - [ ] 1.4 Implement Load() function to read YAML from ~/.config/trani/config.yaml
  - [ ] 1.5 Implement ExpandPaths() method to replace ~ with $HOME in all path fields
  - [ ] 1.6 Implement ApplyDefaults() method to set default paths when not specified in config
  - [ ] 1.7 Create notifier package in pkg/notify/notify.go with Info() and Error() methods
  - [ ] 1.8 Write unit tests in internal/config/config_test.go for missing file, invalid YAML, and defaults

- [ ] 2.0 Implement audio recording and PipeWire integration
  - [ ] 2.1 [depends on: 1.0] Create Recorder struct in internal/audio/recorder.go with fields for module IDs and PID
  - [ ] 2.2 Implement helper functions to execute pactl commands (load/unload modules)
  - [ ] 2.3 Implement function to start pw-record process and capture PID
  - [ ] 2.4 Implement Start() method that loads null-sink, loopback modules, and starts recording
  - [ ] 2.5 Implement Stop() method that kills pw-record and unloads all modules
  - [ ] 2.6 Implement RecordingPath() method to return temp recording file path
  - [ ] 2.7 Write unit tests in internal/audio/recorder_test.go for module management logic

- [ ] 3.0 Implement transcription backends (local Whisper and OpenAI)
  - [ ] 3.1 [depends on: 1.0] Create Transcriber interface in internal/transcribe/transcriber.go
  - [ ] 3.2 Implement New() factory function that returns appropriate transcriber based on config
  - [ ] 3.3 Implement WhisperLocal struct and Transcribe() method in internal/transcribe/whisper.go
  - [ ] 3.4 Implement OpenAI struct and Transcribe() method in internal/transcribe/openai.go with multipart upload
  - [ ] 3.5 Add comprehensive error handling for missing binaries, models, and API keys
  - [ ] 3.6 Write unit tests in internal/transcribe/transcribe_test.go for both implementations

- [ ] 4.0 Implement session management and workflow orchestration
  - [ ] 4.1 [depends on: 1.0, 2.0, 3.0] Create Session and State structs in internal/session/session.go
  - [ ] 4.2 Implement New() constructor that initializes recorder, transcriber, and llm
  - [ ] 4.3 Implement session directory creation with YYYY-MM-DD-title format
  - [ ] 4.4 Implement SaveState() to write current_session.json and ClearState() to remove it
  - [ ] 4.5 Implement LoadActive() to restore session from current_session.json
  - [ ] 4.6 Implement Start() workflow: check active, create dir, start recording, save state, open neovim
  - [ ] 4.7 [depends on: 5.0] Implement Stop() workflow: stop recording, transcribe, load prompt, call LLM, save results
  - [ ] 4.8 Implement audio file cleanup logic that respects preserveAudio flag
  - [ ] 4.9 Write unit tests in internal/session/session_test.go for state serialization and workflow

- [ ] 5.0 Implement Claude API integration and prompt system
  - [ ] 5.1 [depends on: 1.0] Create Claude struct in internal/llm/claude.go with API key and config
  - [ ] 5.2 Implement New() constructor with ANTHROPIC_API_KEY validation
  - [ ] 5.3 Implement Generate() method with HTTP POST to https://api.anthropic.com/v1/messages
  - [ ] 5.4 Implement error response parsing and extract error.message from API responses
  - [ ] 5.5 Add loadPromptTemplate() method to session that reads from prompts_dir
  - [ ] 5.6 Implement variable replacement for {{TRANSCRIPTION}} and {{NOTES}} placeholders
  - [ ] 5.7 Add hardcoded fallback prompts for with-notes and no-notes scenarios
  - [ ] 5.8 Write unit tests in internal/llm/claude_test.go for API client and prompt substitution

- [ ] 6.0 Implement CLI commands and complete integration testing
  - [ ] 6.1 [depends on: 1.0] Setup Cobra root command in cmd/root.go with app description
  - [ ] 6.2 [depends on: 4.0] Implement start command in cmd/start.go with title argument and flags
  - [ ] 6.3 [depends on: 4.0] Implement stop command in cmd/stop.go to stop active session
  - [ ] 6.4 [depends on: 4.0] Implement toggle command in cmd/toggle.go with conditional start/stop logic
  - [ ] 6.5 Add persistent flags --prompt and --preserve-audio to relevant commands
  - [ ] 6.6 Create main.go that calls cmd.Execute()
  - [ ] 6.7 [depends on: 5.0] Perform end-to-end integration testing with real audio recording
  - [ ] 6.8 Test all scenarios from PRD section 12.3 (custom titles, prompts, both transcription backends, etc.)
  - [ ] 6.9 Build release binary with CGO_ENABLED=0 and verify <10MB size and <10ms startup time
