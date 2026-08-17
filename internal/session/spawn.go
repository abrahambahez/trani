package session

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// SpawnPostprocess launches a detached background process to transcribe,
// summarize, and finalize a session whose recording has already stopped.
// It runs independently of the recording lock (already cleared by the
// caller) so starting a new session does not wait for it to finish.
func SpawnPostprocess(sessionPath, promptTemplate string, preserveAudio bool, notifyID string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve trani executable: %w", err)
	}

	args := []string{
		"__postprocess-worker",
		"--session-path", sessionPath,
		"--prompt", promptTemplate,
		"--preserve-audio", strconv.FormatBool(preserveAudio),
	}
	if notifyID != "" {
		args = append(args, "--notify-id", notifyID)
	}

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0); err == nil {
		cmd.Stdin = devnull
		cmd.Stdout = devnull
		cmd.Stderr = devnull
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start postprocess worker: %w", err)
	}

	return cmd.Process.Release()
}
