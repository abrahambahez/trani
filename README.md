# Trani - Simple AI meeting assistnant

Transcribe your meetings and generate automatic smart summaries.

## What it does?

Trani records system and mic audio, transcribes it using Whisper, then uses Claude API to generate a structured summary using your notes.

### NOTE: This is a prototype

This prototype is made for my personal needs (transcribe heavy business meetings with complex information), preferences (Whisper.cc, Claude, Neovim -notes-) and hardware. No model (transciption or LLM) or software is configurable yet.

Also, this is an opinionated workflow. The process is tied to the user note. When the user closes the note, the process finish.

## Quick installation

Add to path and set env varables.

```bash
echo 'export PATH="$HOME/trani:$PATH"' >> ~/.zshrc
source ~/.zshrc

echo 'export ANTHROPIC_API_KEY="tu-api-key"' >> ~/.zshrc
source ~/.zshrc
```

## Use
Start a session

```bash
trani start "session_name"
```

- Neovim opens the note file
- When closing Neovim:
   - Recording stops
   - Transcription starts
   - Claude generates smart summary 
   - Temp files are cleaned

As a result, session is stored with this structure:


```
sessions/2025-10-01-session_name/
├── transcripcion.txt  # Transcript
├── notas.md          # User notes
└── resumen.md        # AI Summary
```

## Requirements

- PipeWire (installed by default in Fedora)
- Whisper.cpp installed in `~/whisper.cpp`
- Claude API key
- `jq`, `curl`, `notify-send` (Gnome)

## Notes

- Original audio will be removed after transcrioption, this is by design, only transcription will persist
- When user note is empty or non-existent, Claude infers the summary themes
- You can toogle trani by binding the command to a keyboard shortcut, like `Super+S`

