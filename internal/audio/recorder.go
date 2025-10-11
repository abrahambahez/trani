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

type Recorder struct {
	tempDir      string
	recordingPID int
}

func New(cfg config.AudioConfig, tempDir string) *Recorder {
	return &Recorder{
		tempDir: tempDir,
	}
}

func (r *Recorder) RecordingPath() string {
	return filepath.Join(r.tempDir, "recording.wav")
}

func getActiveMonitorSource() (string, error) {
	cmd := exec.Command("pactl", "get-default-sink")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default sink: %w", err)
	}

	sink := strings.TrimSpace(string(output))
	if sink == "" {
		return "", fmt.Errorf("no default sink found")
	}

	return sink + ".monitor", nil
}

func (r *Recorder) Start(ctx context.Context) error {
	monitor, err := getActiveMonitorSource()
	if err != nil {
		return err
	}

	cmd := exec.Command(
		"pw-record",
		"--target", monitor,
		"--rate", "48000",
		"--channels", "2",
		r.RecordingPath(),
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pw-record: %w", err)
	}

	r.recordingPID = cmd.Process.Pid
	return nil
}

func (r *Recorder) Stop() error {
	if r.recordingPID == 0 {
		return nil
	}

	process, err := os.FindProcess(r.recordingPID)
	if err != nil {
		return nil
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill recording process: %w", err)
	}

	process.Wait()
	r.recordingPID = 0
	return nil
}
