# ADR 001: Audio Recording Strategy

## Status
Accepted

## Context
Trani needs to capture system audio for transcription. Initial implementation used virtual sinks and loopbacks, resulting in low amplitude audio and poor transcription quality.

## Decision
Use direct monitor capture with high-quality recording followed by post-processing.

### Recording Strategy
1. Detect active audio sink: `pactl get-default-sink`
2. Append `.monitor` to get monitor source
3. Record directly with pw-record:
   - Sample rate: 48000 Hz
   - Channels: 2 (stereo)
   - Target: detected monitor source

### Post-Processing
After recording stops, process audio with sox:
```bash
sox input.wav -r 16000 -c 1 output.wav norm -3 highpass 80 lowpass 8000
```

Operations:
- Downsample to 16kHz mono (optimal for Whisper)
- Normalize to -3dB (boost quiet audio)
- High-pass filter at 80Hz (remove rumble)
- Low-pass filter at 8kHz (remove noise)

## Rationale

### Why Direct Monitor?
- Simpler: No virtual sink setup
- More reliable: Fewer failure points
- Better quality: No additional mixing layers
- Lower latency: Direct capture path

### Why 48kHz Stereo Recording?
- Native PipeWire sample rate (no resampling)
- Captures full audio spectrum
- Better source material for post-processing

### Why Post-Process?
- Whisper optimized for 16kHz mono
- Normalization fixes low amplitude issues
- Filters improve transcription accuracy
- Smaller file sizes for API uploads

## Alternatives Considered

### Virtual Sink + Loopbacks (rejected)
```
Create null-sink → Loopback mic → Loopback system → Record
```
- Complex setup prone to failure
- Low amplitude audio (0.000031)
- Caused Whisper hallucinations

### Direct @DEFAULT_MONITOR@ (rejected)
- Often empty/silent on modern systems
- Inconsistent across PipeWire configurations

## Dependencies
- `pactl` (pipewire-pulse)
- `pw-record` (pipewire-utils)
- `sox` (audio post-processing)

## Implementation
- `internal/audio/recorder.go`: Detection and recording (85 lines)
- `internal/session/session.go`: Post-processing integration

## Validation
```bash
# Test 1: Check monitor detection
pactl get-default-sink

# Test 2: Verify amplitude
sox session/audio.wav -n stat | grep Maximum
# Expected: > 0.1

# Test 3: Verify format
soxi session/audio.wav
# Expected: 16kHz, mono

# Test 4: Test transcription
trani start test
# Play audio, verify accurate transcription
```

## Consequences

### Positive
- Higher audio quality
- Better transcription accuracy
- Simpler codebase (-45 lines)
- More maintainable

### Negative
- Requires sox dependency
- Additional processing time (~1-2 seconds)
- Temporary disk usage (2x audio file size during processing)

## Notes
- Post-processing is non-destructive (original → temp → replace)
- Processing time negligible compared to transcription/LLM time
- Sox is widely available on Linux distributions
