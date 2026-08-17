package session

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sabhz/trani/internal/config"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Paths: config.PathsConfig{TempDir: t.TempDir()},
	}
}

func TestAcquireOnlyOneWinnerUnderConcurrency(t *testing.T) {
	cfg := testConfig(t)

	const attempts = 20
	var wins int32
	var wg sync.WaitGroup

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lock := &RecordingLock{PID: os.Getpid(), Title: "concurrent"}
			if err := lock.Acquire(cfg); err == nil {
				atomic.AddInt32(&wins, 1)
			}
		}(i)
	}

	wg.Wait()

	if wins != 1 {
		t.Errorf("expected exactly 1 winner out of %d concurrent Acquire calls, got %d", attempts, wins)
	}
}

func TestAcquireFailsWhileLockIsHeld(t *testing.T) {
	cfg := testConfig(t)

	first := &RecordingLock{PID: os.Getpid(), Title: "first"}
	if err := first.Acquire(cfg); err != nil {
		t.Fatalf("first Acquire should succeed, got: %v", err)
	}

	second := &RecordingLock{PID: os.Getpid(), Title: "second"}
	if err := second.Acquire(cfg); err == nil {
		t.Error("second Acquire should fail while the first lock is still held")
	}
}

func TestAcquireSucceedsAfterClear(t *testing.T) {
	cfg := testConfig(t)

	first := &RecordingLock{PID: os.Getpid(), Title: "first"}
	if err := first.Acquire(cfg); err != nil {
		t.Fatalf("first Acquire should succeed, got: %v", err)
	}
	if err := ClearLock(cfg); err != nil {
		t.Fatalf("ClearLock failed: %v", err)
	}

	second := &RecordingLock{PID: os.Getpid(), Title: "second"}
	if err := second.Acquire(cfg); err != nil {
		t.Errorf("Acquire after ClearLock should succeed, got: %v", err)
	}
}

func TestAcquireReplacesStaleLock(t *testing.T) {
	cfg := testConfig(t)

	// PID 0 is never a real process, so isProcessAlive treats it as dead.
	stale := &RecordingLock{PID: 0, Title: "stale"}
	if err := stale.Acquire(cfg); err != nil {
		t.Fatalf("failed to seed a stale lock: %v", err)
	}

	fresh := &RecordingLock{PID: os.Getpid(), Title: "fresh"}
	if err := fresh.Acquire(cfg); err != nil {
		t.Errorf("Acquire should replace a stale lock, got: %v", err)
	}

	lock, err := ReadLock(cfg)
	if err != nil {
		t.Fatalf("ReadLock failed: %v", err)
	}
	if lock == nil || lock.Title != "fresh" {
		t.Errorf("expected the fresh lock to be in place, got: %+v", lock)
	}
}
