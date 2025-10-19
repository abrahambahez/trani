# Trani - AI-Powered Meeting Assistant

A streamlined tool for recording, transcribing, and summarizing audio sessions with AI-generated insights.

## Overview

Trani captures system and microphone audio, transcribes it using Whisper (local or OpenAI), and generates structured summaries through Claude AI. Designed for professionals who need accurate meeting documentation with minimal overhead.

## Features

- **Dual transcription backends**: Local whisper.cpp or OpenAI Whisper API
- **High-quality audio capture**: Direct PipeWire monitor recording with post-processing
- **AI-powered summaries**: Claude API integration with customizable prompts
- **Note-driven workflow**: Take notes during recording for contextual summaries
- **Session management**: Organized output with timestamps and titles
- **Flexible commands**: Start, stop, or toggle recording with keyboard shortcuts

## Installation

### Prerequisites

```bash
# Fedora/RHEL
sudo dnf install pipewire pipewire-pulse pipewire-utils sox

# Ubuntu/Debian
sudo apt install pipewire pipewire-pulse pipewire-audio-client-utils sox
```

### Setup

1. **Download binary** (or build from source)
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
    model: claude-sonnet-4-20250514
    max_tokens: 4000

  ollama:
    base_url: http://localhost:11434
    model: llama3.2

paths:
  sessions_dir: ~/.config/trani/sessions
  temp_dir: ~/.config/trani/temp
  prompts_dir: ~/.config/trani/prompts
```

## Usage

### Basic Workflow

**Start a session:**
```bash
trani start "meeting-title"
```

This will:
1. Begin audio recording
2. Open Neovim for note-taking
3. Upon closing Neovim:
   - Stop recording
   - Process audio (normalize, filter)
   - Transcribe content
   - Generate AI summary
   - Save results

**Manual stop:**
```bash
trani stop
```

**Toggle recording:**
```bash
trani toggle "meeting-title"
```

**Process existing audio:**
```bash
trani process audio.wav
trani process audio.wav --notes notes.md --title "meeting-summary"
```

### Command Options

**start/toggle:**
```bash
trani start [title] --prompt TEMPLATE --preserve-audio
trani toggle [title] --prompt TEMPLATE --preserve-audio
```

- `--prompt`: Use custom prompt template (default: "default")
- `--preserve-audio`: Keep audio file after processing

**process:**
```bash
trani process <audio-file> --notes FILE --title NAME --prompt TEMPLATE
```

- `<audio-file>`: Path to audio file to process (required)
- `--notes`: Path to notes file to include in summary
- `--title`: Output directory title (defaults to audio filename)
- `--prompt`: Use custom prompt template (default: "default")

### Output Structure

```
~/.config/trani/sessions/2025-10-11-meeting-title/
├── transcripcion.txt  # Full transcription
├── notas.md          # User notes
├── resumen.md        # AI-generated summary
└── audio.wav         # (optional, with --preserve-audio)
```

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

1. **Capture**: Direct monitor source at 48kHz stereo
2. **Post-processing**:
   - Downsample to 16kHz mono (optimal for Whisper)
   - Normalize to 0dB (maximum safe volume)
   - High-pass filter at 80Hz (remove rumble)
   - Low-pass filter at 8kHz (remove high-frequency noise)

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
- Default model: claude-sonnet-4-20250514

**Ollama**:
- Local model inference
- No API costs
- Privacy-focused
- Requires Ollama running locally
- Compatible with llama3.2, mistral, and other models

## Build from Source

```bash
git clone https://github.com/yourusername/trani
cd trani
go build -o trani
```

**Optimized build**:
```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o trani
```

## Troubleshooting

**Low audio volume**: Ensure PipeWire is properly configured and sox is installed.

**Transcription errors**: Check API keys and network connectivity for OpenAI backend.

**Recording fails**: Verify PipeWire/PulseAudio is running:
```bash
systemctl --user status pipewire pipewire-pulse
```

## Configuration Reference

See `docs/PRD/001-trani-go.md` for complete configuration options and architecture details.

## License

MIT

## Contributing

Contributions welcome. Please ensure code follows existing patterns and includes tests.
