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
type Recorder struct {
	tempDir   string
	mode      string
	micDevice string

	micPID    int
	systemPID int
}

func New(cfg config.AudioConfig, tempDir string) *Recorder {
	mode := cfg.Mode
	if mode == "" {
		mode = config.AudioModeMic
	}

	return &Recorder{
		tempDir:   tempDir,
		mode:      mode,
		micDevice: cfg.MicDevice,
	}
}

// HasSystemAudio reports whether this recorder also captures system output.
func (r *Recorder) HasSystemAudio() bool {
	return r.mode == config.AudioModeMicSystem
}

// MicPath returns the temp path for the raw microphone recording.
func (r *Recorder) MicPath() string {
	return filepath.Join(r.tempDir, "recording-mic.wav")
}

// SystemPath returns the temp path for the raw system output recording.
func (r *Recorder) SystemPath() string {
	return filepath.Join(r.tempDir, "recording-system.wav")
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

func startPwRecord(ctx context.Context, target, outputPath string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(
		ctx,
		"pw-record",
		"--target", target,
		"--rate", "48000",
		"--channels", "2",
		outputPath,
	)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return cmd, nil
}

// Start begins capturing the microphone and, in mic_system mode, the system
// output monitor in parallel, as two independent direct streams.
func (r *Recorder) Start(ctx context.Context) error {
	micSource, err := activeMicSource(r.micDevice)
	if err != nil {
		return fmt.Errorf("failed to resolve microphone source: %w", err)
	}

	micCmd, err := startPwRecord(ctx, micSource, r.MicPath())
	if err != nil {
		return fmt.Errorf("failed to start microphone recording: %w", err)
	}
	r.micPID = micCmd.Process.Pid

	if !r.HasSystemAudio() {
		return nil
	}

	systemSource, err := activeMonitorSource()
	if err != nil {
		stopProcess(&r.micPID)
		return fmt.Errorf("failed to resolve system output source: %w", err)
	}

	systemCmd, err := startPwRecord(ctx, systemSource, r.SystemPath())
	if err != nil {
		stopProcess(&r.micPID)
		return fmt.Errorf("failed to start system output recording: %w", err)
	}
	r.systemPID = systemCmd.Process.Pid

	return nil
}

// Stop stops any active recording streams.
func (r *Recorder) Stop() error {
	if err := stopProcess(&r.micPID); err != nil {
		return fmt.Errorf("failed to stop microphone recording: %w", err)
	}

	if err := stopProcess(&r.systemPID); err != nil {
		return fmt.Errorf("failed to stop system output recording: %w", err)
	}

	return nil
}

func stopProcess(pid *int) error {
	if *pid == 0 {
		return nil
	}

	process, err := os.FindProcess(*pid)
	if err != nil {
		*pid = 0
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	process.Wait()
	*pid = 0
	return nil
}
