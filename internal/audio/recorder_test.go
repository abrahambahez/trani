package audio

import (
	"os/exec"
	"testing"

	"github.com/sabhz/trani/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.AudioConfig{
		SampleRate: 16000,
		Channels:   1,
	}
	tempDir := "/tmp/test"

	recorder := New(cfg, tempDir)

	if recorder == nil {
		t.Fatal("New() returned nil")
	}

	if recorder.tempDir != tempDir {
		t.Errorf("tempDir: expected %s, got %s", tempDir, recorder.tempDir)
	}
}

func TestRecordingPath(t *testing.T) {
	cfg := config.AudioConfig{
		SampleRate: 16000,
		Channels:   1,
	}
	tempDir := "/tmp/trani"

	recorder := New(cfg, tempDir)
	path := recorder.RecordingPath()

	expected := "/tmp/trani/recording.wav"
	if path != expected {
		t.Errorf("RecordingPath: expected %s, got %s", expected, path)
	}
}

func TestRecordingPathWithDifferentTempDir(t *testing.T) {
	cfg := config.AudioConfig{
		SampleRate: 48000,
		Channels:   2,
	}
	tempDir := "/var/tmp/custom"

	recorder := New(cfg, tempDir)
	path := recorder.RecordingPath()

	expected := "/var/tmp/custom/recording.wav"
	if path != expected {
		t.Errorf("RecordingPath: expected %s, got %s", expected, path)
	}
}

func TestStopWithoutStart(t *testing.T) {
	cfg := config.AudioConfig{
		SampleRate: 16000,
		Channels:   1,
	}
	recorder := New(cfg, "/tmp/test")

	err := recorder.Stop()
	if err != nil {
		t.Errorf("Stop() without Start() should not error, got: %v", err)
	}
}

func TestStopClearsRecordingPID(t *testing.T) {
	cfg := config.AudioConfig{
		SampleRate: 16000,
		Channels:   1,
	}
	recorder := New(cfg, "/tmp/test")

	proc := exec.Command("sleep", "30")
	if err := proc.Start(); err != nil {
		t.Fatalf("failed to start placeholder process: %v", err)
	}
	recorder.recordingPID = proc.Process.Pid

	if err := recorder.Stop(); err != nil {
		t.Errorf("Stop() should not error, got: %v", err)
	}

	if recorder.recordingPID != 0 {
		t.Errorf("recordingPID should be cleared, got: %d", recorder.recordingPID)
	}
}
