package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sabhz/trani/internal/config"
)

// RecordingLock is the on-disk marker for an in-progress recording. It only
// exists while audio is actively being captured; it is removed as soon as
// recording stops, before transcription/summary post-processing begins, so a
// new session can start while the previous one is still being processed.
type RecordingLock struct {
	PID            int       `json:"pid"`
	Title          string    `json:"title"`
	Path           string    `json:"path"`
	StartedAt      time.Time `json:"started_at"`
	PromptTemplate string    `json:"prompt_template"`
	PreserveAudio  bool      `json:"preserve_audio"`
	NotifyID       string    `json:"notify_id"`
}

func lockPath(cfg *config.Config) string {
	return filepath.Join(cfg.Paths.TempDir, "active_recording.json")
}

// Save writes the lock to disk, creating the temp directory if needed.
func (l *RecordingLock) Save(cfg *config.Config) error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal recording lock: %w", err)
	}

	if err := os.MkdirAll(cfg.Paths.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	if err := os.WriteFile(lockPath(cfg), data, 0644); err != nil {
		return fmt.Errorf("failed to write recording lock: %w", err)
	}

	return nil
}

// ReadLock returns the active recording lock, or nil if there is none. A
// lock whose PID is no longer alive is treated as stale and removed.
func ReadLock(cfg *config.Config) (*RecordingLock, error) {
	data, err := os.ReadFile(lockPath(cfg))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read recording lock: %w", err)
	}

	var lock RecordingLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse recording lock: %w", err)
	}

	if !isProcessAlive(lock.PID) {
		os.Remove(lockPath(cfg))
		return nil, nil
	}

	return &lock, nil
}

// ClearLock removes the recording lock, if present.
func ClearLock(cfg *config.Config) error {
	if err := os.Remove(lockPath(cfg)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear recording lock: %w", err)
	}
	return nil
}

// SignalStop asks the process holding the lock to stop recording.
func SignalStop(lock *RecordingLock) error {
	process, err := os.FindProcess(lock.PID)
	if err != nil {
		return fmt.Errorf("failed to find recording process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to signal recording process: %w", err)
	}

	return nil
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}
