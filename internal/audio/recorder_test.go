package audio

import (
	"os/exec"
	"testing"

	"github.com/sabhz/trani/internal/config"
)

func TestNewDefaultsToMicMode(t *testing.T) {
	cfg := config.AudioConfig{}
	recorder := New(cfg, "/tmp/test")

	if recorder == nil {
		t.Fatal("New() returned nil")
	}

	if recorder.HasSystemAudio() {
		t.Error("expected mic-only mode by default, got mic_system")
	}
}

func TestNewMicSystemMode(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMicSystem}
	recorder := New(cfg, "/tmp/test")

	if !recorder.HasSystemAudio() {
		t.Error("expected mic_system mode, got mic-only")
	}
}

func TestMicPath(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMic}
	recorder := New(cfg, "/tmp/trani")

	expected := "/tmp/trani/recording-mic.wav"
	if path := recorder.MicPath(); path != expected {
		t.Errorf("MicPath: expected %s, got %s", expected, path)
	}
}

func TestSystemPath(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMicSystem}
	recorder := New(cfg, "/tmp/trani")

	expected := "/tmp/trani/recording-system.wav"
	if path := recorder.SystemPath(); path != expected {
		t.Errorf("SystemPath: expected %s, got %s", expected, path)
	}
}

func TestStopWithoutStart(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMicSystem}
	recorder := New(cfg, "/tmp/test")

	if err := recorder.Stop(); err != nil {
		t.Errorf("Stop() without Start() should not error, got: %v", err)
	}
}

func TestStopClearsRecordingPIDs(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMicSystem}
	recorder := New(cfg, "/tmp/test")

	micProc := exec.Command("sleep", "30")
	if err := micProc.Start(); err != nil {
		t.Fatalf("failed to start placeholder process: %v", err)
	}
	recorder.micPID = micProc.Process.Pid

	systemProc := exec.Command("sleep", "30")
	if err := systemProc.Start(); err != nil {
		t.Fatalf("failed to start placeholder process: %v", err)
	}
	recorder.systemPID = systemProc.Process.Pid

	if err := recorder.Stop(); err != nil {
		t.Errorf("Stop() should not error, got: %v", err)
	}

	if recorder.micPID != 0 {
		t.Errorf("micPID should be cleared, got: %d", recorder.micPID)
	}
	if recorder.systemPID != 0 {
		t.Errorf("systemPID should be cleared, got: %d", recorder.systemPID)
	}
}
