# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `--preserve-audio` flag to keep original audio file after processing

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

