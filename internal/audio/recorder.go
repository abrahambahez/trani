package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sabhz/trani/internal/config"
)

// Recorder captures one or two audio streams directly (mic, and optionally
// the system output monitor), never through a virtual sink. Mixing virtual
// sinks with loopbacks was tried and rejected (see docs/ADR/001), it
// produced near-silent audio that caused transcription hallucinations.
//
// Each stream is segmented into fixed-length chunks by ffmpeg as it
// records, so a chunker can pick up and transcribe finished segments
// progressively instead of waiting for the whole session to end.
type Recorder struct {
	tempDir      string
	mode         string
	micDevice    string
	chunkSeconds int

	micCmd    *exec.Cmd
	systemCmd *exec.Cmd
}

func New(cfg config.AudioConfig, tempDir string) *Recorder {
	mode := cfg.Mode
	if mode == "" {
		mode = config.AudioModeMic
	}

	chunkSeconds := cfg.ChunkSeconds
	if chunkSeconds == 0 {
		chunkSeconds = 300
	}

	return &Recorder{
		tempDir:      tempDir,
		mode:         mode,
		micDevice:    cfg.MicDevice,
		chunkSeconds: chunkSeconds,
	}
}

// HasSystemAudio reports whether this recorder also captures system output.
func (r *Recorder) HasSystemAudio() bool {
	return r.mode == config.AudioModeMicSystem
}

// MicChunkPattern is the ffmpeg segment output pattern for mic chunks.
func (r *Recorder) MicChunkPattern() string {
	return filepath.Join(r.tempDir, "chunk-mic-%03d.wav")
}

// MicSegmentList is the path ffmpeg appends a line to every time it closes
// a mic chunk.
func (r *Recorder) MicSegmentList() string {
	return filepath.Join(r.tempDir, "chunk-mic-segments.txt")
}

// SystemChunkPattern is the ffmpeg segment output pattern for system chunks.
func (r *Recorder) SystemChunkPattern() string {
	return filepath.Join(r.tempDir, "chunk-system-%03d.wav")
}

// SystemSegmentList is the path ffmpeg appends a line to every time it
// closes a system audio chunk.
func (r *Recorder) SystemSegmentList() string {
	return filepath.Join(r.tempDir, "chunk-system-segments.txt")
}

func activeMonitorSource() (string, error) {
	output, err := exec.Command("pactl", "get-default-sink").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default sink: %w", err)
	}

	sink := strings.TrimSpace(string(output))
	if sink == "" {
		return "", fmt.Errorf("no default sink found")
	}

	return sink + ".monitor", nil
}

func activeMicSource(override string) (string, error) {
	if override != "" {
		return override, nil
	}

	output, err := exec.Command("pactl", "get-default-source").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default source: %w", err)
	}

	source := strings.TrimSpace(string(output))
	if source == "" {
		return "", fmt.Errorf("no default source found")
	}

	return source, nil
}

func startSegmentedCapture(ctx context.Context, source string, chunkSeconds int, segmentList, pattern string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-hide_banner", "-loglevel", "warning",
		"-f", "pulse", "-i", source,
		"-ar", "48000", "-ac", "2", "-acodec", "pcm_s16le",
		"-f", "segment",
		"-segment_time", fmt.Sprintf("%d", chunkSeconds),
		"-reset_timestamps", "1",
		"-segment_list", segmentList,
		"-segment_list_type", "flat",
		"-y",
		pattern,
	)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// Start begins capturing the microphone and, in mic_system mode, the system
// output monitor in parallel, as two independent direct streams segmented
// into chunks.
func (r *Recorder) Start(ctx context.Context) error {
	micSource, err := activeMicSource(r.micDevice)
	if err != nil {
		return fmt.Errorf("failed to resolve microphone source: %w", err)
	}

	os.Remove(r.MicSegmentList())

	micCmd, err := startSegmentedCapture(ctx, micSource, r.chunkSeconds, r.MicSegmentList(), r.MicChunkPattern())
	if err != nil {
		return fmt.Errorf("failed to start microphone recording: %w", err)
	}
	r.micCmd = micCmd

	if !r.HasSystemAudio() {
		return nil
	}

	systemSource, err := activeMonitorSource()
	if err != nil {
		stopCmd(r.micCmd)
		r.micCmd = nil
		return fmt.Errorf("failed to resolve system output source: %w", err)
	}

	os.Remove(r.SystemSegmentList())

	systemCmd, err := startSegmentedCapture(ctx, systemSource, r.chunkSeconds, r.SystemSegmentList(), r.SystemChunkPattern())
	if err != nil {
		stopCmd(r.micCmd)
		r.micCmd = nil
		return fmt.Errorf("failed to start system output recording: %w", err)
	}
	r.systemCmd = systemCmd

	return nil
}

// Stop stops any active recording streams, letting ffmpeg finalize (and
// list) whatever chunk was still open.
func (r *Recorder) Stop() error {
	if err := stopCmd(r.micCmd); err != nil {
		return fmt.Errorf("failed to stop microphone recording: %w", err)
	}
	r.micCmd = nil

	if err := stopCmd(r.systemCmd); err != nil {
		return fmt.Errorf("failed to stop system output recording: %w", err)
	}
	r.systemCmd = nil

	return nil
}

func stopCmd(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	cmd.Wait()
	return nil
}
