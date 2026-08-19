# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.3.0] - 2026-08-19

### Fixed
- Postprocessing (both the live session flow and `process`) no longer overwrites the note's existing content; the generated summary is now appended below it under a `## Resumen` heading, preserving any frontmatter or notes a template or the user already put there

### Changed
- **Breaking**: `process` now writes into the same `sessions_dir` as live sessions instead of its own per-audio directory, and no longer produces separate `transcripcion.txt`/`notas.md`/`resumen.md` files — output is a single note (named by timestamp, like live sessions) plus a `.sources/<timestamp>.txt` transcript
- **Breaking**: `process` now exits with a non-zero status when the prompt template is missing or summary generation fails, instead of reporting success with the error message written in place of the summary
- **Breaking**: Removed `--title` from `process` (it named the output directory but was never actually wired into the path); the output note is now named purely by timestamp, matching live sessions

## [2.2.0] - 2026-08-18

### Changed
- **Breaking**: `--preserve-audio` is no longer a CLI flag on `start`/`toggle` (or the hidden worker subcommands); whether the archived audio in `.sources/` is kept is now `audio.preserved` in `config.yaml`, applied consistently instead of having to be repeated on every invocation

## [2.1.0] - 2026-08-18

### Added
- Error-only JSON Lines log at `~/.config/trani/logs.jsonl` (`pkg/errlog`, `log/slog` with a JSON handler): every failure that today only surfaces as an ephemeral desktop notification, or (for the Obsidian-open warning) not even that, since detached-worker stdio goes to `/dev/null`, is now also appended there with an `event` code and the session title

## [2.0.2] - 2026-08-18

### Fixed
- Post-processing (both `process` and the live session flow) no longer sends a broken placeholder prompt to the model when the requested prompt template and its `default` fallback are both missing; it now aborts before calling the LLM

## [2.0.1] - 2026-08-18

### Removed
- `PRD.md` (redundant with the README as project entry point) and the stale root-level `config.json` (superseded by `~/.config/trani/config.yaml`)

### Fixed
- README "Build from Source" pointed at a placeholder `yourusername/trani` clone URL instead of the real repository

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

