package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/sabhz/trani/internal/config"
)

// spawnDetached launches trani with the given hidden-subcommand args as an
// independent background process (own session, stdio discarded) so the
// caller can return without waiting for it.
func spawnDetached(args ...string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve trani executable: %w", err)
	}

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if devnull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0); err == nil {
		cmd.Stdin = devnull
		cmd.Stdout = devnull
		cmd.Stderr = devnull
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", args[0], err)
	}

	return cmd.Process.Release()
}

// SpawnPostprocess launches a detached background process to transcribe,
// summarize, and finalize a session whose recording has already stopped.
// It runs independently of the recording lock (already cleared by the
// caller) so starting a new session does not wait for it to finish.
func SpawnPostprocess(notePath, sourcesTitle, promptTemplate string, preserveAudio bool, notifyID string) error {
	args := []string{
		"__postprocess-worker",
		"--note-path", notePath,
		"--sources-title", sourcesTitle,
		"--prompt", promptTemplate,
		"--preserve-audio=" + strconv.FormatBool(preserveAudio),
	}
	if notifyID != "" {
		args = append(args, "--notify-id", notifyID)
	}

	return spawnDetached(args...)
}

// SpawnRecorder launches a detached background process that owns the whole
// recording session (Session.Start). It's used when the editor doesn't
// block (Obsidian is a separate GUI app), since there's no editor process
// for the invoking CLI command to wait on.
func SpawnRecorder(promptTemplate string, preserveAudio bool) error {
	args := []string{
		"__record-worker",
		"--prompt", promptTemplate,
		"--preserve-audio=" + strconv.FormatBool(preserveAudio),
	}

	return spawnDetached(args...)
}

// Launch starts a session, either blocking in the foreground (nvim
// fallback, since it needs a terminal) or as a detached background worker
// (Obsidian), depending on whether an Obsidian vault is configured.
func Launch(promptTemplate string, preserveAudio bool, cfg *config.Config) error {
	if lock, err := ReadLock(cfg); err != nil {
		return err
	} else if lock != nil {
		return fmt.Errorf("session already active: %s", lock.Title)
	}

	if cfg.Obsidian.VaultPath != "" {
		return SpawnRecorder(promptTemplate, preserveAudio)
	}

	sess, err := New(promptTemplate, preserveAudio, cfg)
	if err != nil {
		return err
	}

	return sess.Start(context.Background())
}
