# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] - 2026-08-18

### Added
- `--preserve-audio` flag to keep original audio file after processing
- Microphone capture, alone (`audio.mode: mic`) or together with system output (`audio.mode: mic_system`) for meetings, as two independent direct streams (never through a virtual sink)
- Two selectable strategies for combining mic + system audio (`audio.mix_strategy`): `post_mix` (normalize each stream, then mix and renormalize) and `separate_transcribe` (transcribe each stream independently and concatenate)
- Progressive, chunked recording and transcription (`audio.chunk_seconds`, default 300s): most of the transcription happens while the session is still running instead of all at once when it stops, accumulating into `sessions/.sources/<timestamp>.txt` and `.wav`
- Automatic cleanup of consecutive duplicate transcript lines (a common Whisper hallucination) before generating the summary
- Obsidian integration (`obsidian.vault_path`, required): sessions open the note via Obsidian's `obsidian://open` URI; `start`/`toggle` fail if no vault is configured
- Session notes are now a single file: the user's raw notes and the generated summary are the same note, overwritten in place once the summary is ready (left untouched if summary generation fails)

### Changed
- Recording lock now covers only active recording, not post-processing: a new session can start immediately after stopping the previous one, even while its summary is still being generated
- Lock acquisition is now atomic, closing a race where two near-simultaneous session starts could orphan an unstoppable background recording
- Session timestamps now use the `2006-01-02 1504` format (date with dashes, space, then HHMM)

### Fixed
- `internal/audio/recorder_test.go` referenced fields that no longer existed in `Recorder`, breaking `go test ./...`

## [0.2.0] - 2025-10-02

### Added
- Custom prompt template system with `--prompt` flag
- Support for paired template files (with/without notes variants)
- Variable substitution in templates (`{{TRANSCRIPTION}}`, `{{NOTES}}`)
- Template directory (`prompts/`) for custom prompts
- Prompt template selection saved in session state
- Fallback to hardcoded prompts when templates not found

### Changed
- Session state JSON now includes `prompt_template` field
- Help text updated to show available prompts and new options

## [0.1.0] - 2025-10-01

### Added
- Core CLI meeting assistant functionality
- System and microphone audio recording via PipeWire
- Automatic transcription with Whisper.cpp
- AI-powered summary generation via Claude API
- Session management with start/stop/toggle commands
- Neovim integration for note-taking during sessions
- Structured output: transcriptions, notes, and summaries
- Session organization by date and title
- Spanish language support for transcription and prompts
- Desktop notifications for recording status
- Session state persistence with JSON
- Configuration file support

