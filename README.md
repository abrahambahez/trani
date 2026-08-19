# Trani - AI-Powered Meeting Assistant

A streamlined tool for recording, transcribing, and summarizing audio sessions with AI-generated insights.

## Overview

Trani captures microphone and/or system audio, transcribes it progressively while the session is still running, and generates a structured summary through an LLM. Sessions require an Obsidian vault to be configured, and open the note there; most of the transcription happens in the background while you're still in the meeting, not all at once afterward.

## Features

- **Mic or mic + system capture**: record just the microphone (dictation-style sessions) or the microphone together with system output (meetings), as two independent direct streams, never through a virtual sink
- **Progressive, chunked transcription**: audio is segmented and transcribed while the session is live, not all at once when it stops
- **Dual transcription backends**: local whisper.cpp or OpenAI Whisper API
- **AI-powered summaries**: pluggable LLM backend (Claude or Ollama) with customizable prompts
- **Obsidian required**: notes open in your Obsidian vault; `start`/`toggle` fail immediately if no vault is configured
- **Concurrent-safe sessions**: starting a new session doesn't wait for the previous one's summary to finish generating
- **Flexible commands**: start, stop, or toggle recording with keyboard shortcuts

## Installation

### Prerequisites

```bash
# Fedora/RHEL
sudo dnf install pipewire pipewire-pulse pipewire-utils sox ffmpeg

# Ubuntu/Debian
sudo apt install pipewire pipewire-pulse pipewire-audio-client-utils sox ffmpeg
```

Required, for the live session flow (`start`/`toggle`/`stop`):
- `xdg-open` (standard on most Linux desktops already) able to resolve the `obsidian://` URI scheme, which Obsidian itself registers on install — no separate CLI tool needed
- Obsidian running with the target vault open (trani degrades gracefully — logs a warning and keeps going — if it isn't)

`trani process` (reprocessing an existing audio file) doesn't need any of the above.

### Setup

1. **Download binary** (or build from source, see below)
2. **Configure API keys**:
```bash
export ANTHROPIC_API_KEY="your-claude-api-key"
export OPENAI_API_KEY="your-openai-api-key"  # Optional, for OpenAI transcription
```

3. **Create configuration** at `~/.config/trani/config.yaml`:
```yaml
transcription:
  backend: openai  # or "local" for whisper.cpp

  local:
    model_path: ~/whisper.cpp/models/ggml-large-v3-turbo.bin
    binary_path: ~/whisper.cpp/build/bin/whisper-cli
    threads: 12
    language: es

  openai:
    model: whisper-1
    language: es

llm:
  backend: claude  # or "ollama" for local models

  claude:
    model: claude-sonnet-5
    max_tokens: 4000

  ollama:
    base_url: http://localhost:11434
    model: llama3.2

audio:
  mode: mic_system        # mic | mic_system
  mic_device: ""           # pactl source name; empty uses the default source
  mix_strategy: post_mix   # post_mix | separate_transcribe (mic_system only)
  chunk_seconds: 300       # how often to segment and transcribe progressively
  preserved: false         # keep the archived audio in .sources/ after processing (live session flow only)

obsidian:
  vault_path: ~/vault      # required for start/toggle/stop

paths:
  sessions_dir: ~/vault/sessions  # must live inside vault_path if obsidian is configured
  temp_dir: ~/.config/trani/temp
  prompts_dir: ~/.config/trani/prompts
```

## Usage

### Basic Workflow

**Toggle recording:**
```bash
trani toggle
```

Requires `obsidian.vault_path` to be configured:
1. Starts recording in the background and returns immediately
2. Opens the session note in Obsidian
3. Audio is segmented and transcribed progressively as the session runs
4. Take notes in the note Obsidian opened
5. Run `trani toggle` (or `trani stop`) again to stop: recording stops and a summary is generated and written into the same note
6. A new `trani toggle` can be run immediately, even while the previous session's summary is still being generated

**Manual stop:**
```bash
trani stop
```

**Process existing audio:**
```bash
trani process audio.wav
trani process audio.wav --notes notes.md --title "meeting-summary"
```

### Command Options

**start/toggle:**
```bash
trani start --prompt TEMPLATE
trani toggle --prompt TEMPLATE
```

- `--prompt`: Use custom prompt template (default: "default")

Whether the archived audio in `.sources/` is kept after processing is set via `audio.preserved` in the config, not a flag.

**process:**
```bash
trani process <audio-file> --notes FILE --prompt TEMPLATE
```

- `<audio-file>`: Path to audio file to process (required)
- `--notes`: Path to notes file to include in summary
- `--prompt`: Use custom prompt template (default: "default")

`process` is a standalone, one-shot command for reprocessing an existing recording — it isn't part of the live session flow above, but writes into the same `sessions_dir` and postprocesses identically (notes preserved, summary appended below them).

### Output Structure

Sessions:
```
<sessions_dir>/2026-01-15 1430.md                  # notes + appended summary, same file
<sessions_dir>/.sources/2026-01-15 1430.txt        # accumulated raw transcript
<sessions_dir>/.sources/2026-01-15 1430.wav        # archived audio (deleted unless audio.preserved is true)
```

`process`:
```
<sessions_dir>/2026-01-15 1430.md                  # notes (if --notes given) + appended summary, same file
<sessions_dir>/.sources/2026-01-15 1430.txt        # full transcription
```
`process` always removes its working copy of the audio file once done; `audio.preserved` only affects the live session flow.

## Advanced Features

### Custom Prompts

Create templates in `~/.config/trani/prompts/`:

- `template-name.txt` - Used when notes exist
- `template-name_no_notes.txt` - Used without notes

Templates support variables:
- `{{TRANSCRIPTION}}` - Full audio transcript
- `{{NOTES}}` - User-provided notes

### Keyboard Shortcuts

Bind commands to keyboard shortcuts for quick access:

```bash
# Example: GNOME Settings → Keyboard → Custom Shortcuts
Command: trani toggle
Shortcut: Super+T
```

## Technical Details

### Audio Processing Pipeline

1. **Capture**: mic and, in `mic_system` mode, the system output monitor, as two independent direct streams (never mixed through a virtual sink — see `docs/ADR/001-audio-strategy.md`), segmented into `audio.chunk_seconds` chunks via ffmpeg
2. **Per-chunk post-processing** (sox):
   - Downsample to 16kHz mono (optimal for Whisper)
   - Normalize to 0dB (maximum safe volume)
   - High-pass filter at 80Hz (remove rumble)
   - Low-pass filter at 8kHz (remove high-frequency noise)
3. **`mic_system` combination**: both streams are normalized independently before being combined, per `audio.mix_strategy` (see `docs/ADR/002-progressive-sessions.md`)
4. **Progressive transcription**: each chunk is transcribed as it closes and appended to `.sources/<timestamp>.txt`; consecutive duplicate lines (a common Whisper hallucination) are dropped before the transcript goes into the summary prompt

### Transcription Backends

**Local (whisper.cpp)**:
- No API costs
- Privacy-focused
- Requires local model installation
- CPU/GPU processing

**OpenAI API**:
- Pay-per-use
- Faster processing
- No local setup required
- Network-dependent

### LLM Integration

**Claude (default)**:
- Cloud-based API from Anthropic
- High-quality structured summaries
- Requires ANTHROPIC_API_KEY environment variable
- No built-in default model — set `llm.claude.model` in the config (e.g. `claude-sonnet-5`)

**Ollama**:
- Local model inference
- No API costs
- Privacy-focused
- Requires Ollama running locally
- Compatible with llama3.2, mistral, and other models

## Build from Source

```bash
git clone https://github.com/abrahambahez/trani
cd trani
go build -o trani
```

**Optimized build**:
```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o trani
```

## Troubleshooting

**Low audio volume**: Ensure PipeWire is properly configured and sox is installed. If using `mic_system`, check that the current default sink/source is actually carrying signal (`pactl list short sinks`/`sources`) — a suspended or unused device can silently produce empty captures.

**Transcription errors**: Check API keys and network connectivity for OpenAI backend.

**Recording fails**: Verify PipeWire/PulseAudio is running:
```bash
systemctl --user status pipewire pipewire-pulse
```

**Note doesn't open in Obsidian**: trani logs a warning and keeps recording regardless. Check that Obsidian is running with the target vault already open (a cold start silently ignores the URI's `file=` argument, it just restores whatever was last open) and that `xdg-open` resolves `obsidian://` on your system (`xdg-open "obsidian://open?vault=<name>&file=<note>"` should jump to that note if Obsidian is already open).

**"session already active" but nothing seems to be recording**: check for a stale lock at `<temp_dir>/active_recording.json`; if its PID isn't running anymore, trani clears it automatically on the next command.

**Errors that only flashed by in a desktop notification**: the live session flow (`start`/`toggle`/`stop`) runs as detached background workers with no visible stdout/stderr, so failures show up as a `notify-send` popup that's easy to miss. Every such failure is also appended as a JSON line to `~/.config/trani/logs.jsonl` (`{"time":...,"level":"ERROR","msg":...,"event":...,"session":...}`), so it's not lost — inspect with `tail -f ~/.config/trani/logs.jsonl | jq .` or filter by cause with `jq 'select(.event=="obsidian_open")' ~/.config/trani/logs.jsonl`.

## Configuration Reference

See `docs/ADR/` for architecture decisions and rationale behind the audio capture and session design.

## License

MIT

## Contributing

Contributions welcome. Please ensure code follows existing patterns and includes tests.
