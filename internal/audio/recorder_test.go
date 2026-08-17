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

func TestNewDefaultsChunkSeconds(t *testing.T) {
	cfg := config.AudioConfig{}
	recorder := New(cfg, "/tmp/test")

	if recorder.chunkSeconds != 300 {
		t.Errorf("expected default chunkSeconds 300, got %d", recorder.chunkSeconds)
	}
}

func TestChunkPatternsAndSegmentLists(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMicSystem}
	recorder := New(cfg, "/tmp/trani")

	cases := []struct {
		name     string
		got      string
		expected string
	}{
		{"MicChunkPattern", recorder.MicChunkPattern(), "/tmp/trani/chunk-mic-%03d.wav"},
		{"MicSegmentList", recorder.MicSegmentList(), "/tmp/trani/chunk-mic-segments.txt"},
		{"SystemChunkPattern", recorder.SystemChunkPattern(), "/tmp/trani/chunk-system-%03d.wav"},
		{"SystemSegmentList", recorder.SystemSegmentList(), "/tmp/trani/chunk-system-segments.txt"},
	}

	for _, c := range cases {
		if c.got != c.expected {
			t.Errorf("%s: expected %s, got %s", c.name, c.expected, c.got)
		}
	}
}

func TestStopWithoutStart(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMicSystem}
	recorder := New(cfg, "/tmp/test")

	if err := recorder.Stop(); err != nil {
		t.Errorf("Stop() without Start() should not error, got: %v", err)
	}
}

func TestStopClearsCommands(t *testing.T) {
	cfg := config.AudioConfig{Mode: config.AudioModeMicSystem}
	recorder := New(cfg, "/tmp/test")

	micProc := exec.Command("sleep", "30")
	if err := micProc.Start(); err != nil {
		t.Fatalf("failed to start placeholder process: %v", err)
	}
	recorder.micCmd = micProc

	systemProc := exec.Command("sleep", "30")
	if err := systemProc.Start(); err != nil {
		t.Fatalf("failed to start placeholder process: %v", err)
	}
	recorder.systemCmd = systemProc

	if err := recorder.Stop(); err != nil {
		t.Errorf("Stop() should not error, got: %v", err)
	}

	if recorder.micCmd != nil {
		t.Error("micCmd should be cleared")
	}
	if recorder.systemCmd != nil {
		t.Error("systemCmd should be cleared")
	}
}
