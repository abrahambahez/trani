package audio

import (
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

	if recorder.sampleRate != 16000 {
		t.Errorf("sampleRate: expected 16000, got %d", recorder.sampleRate)
	}

	if recorder.channels != 1 {
		t.Errorf("channels: expected 1, got %d", recorder.channels)
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

func TestStopClearsModuleIDs(t *testing.T) {
	cfg := config.AudioConfig{
		SampleRate: 16000,
		Channels:   1,
	}
	recorder := New(cfg, "/tmp/test")

	recorder.sinkModuleID = "123"
	recorder.loopMicModuleID = "456"
	recorder.loopSysModuleID = "789"
	recorder.recordingPID = 9999

	recorder.Stop()

	if recorder.sinkModuleID != "" {
		t.Errorf("sinkModuleID should be cleared, got: %s", recorder.sinkModuleID)
	}
	if recorder.loopMicModuleID != "" {
		t.Errorf("loopMicModuleID should be cleared, got: %s", recorder.loopMicModuleID)
	}
	if recorder.loopSysModuleID != "" {
		t.Errorf("loopSysModuleID should be cleared, got: %s", recorder.loopSysModuleID)
	}
	if recorder.recordingPID != 0 {
		t.Errorf("recordingPID should be cleared, got: %d", recorder.recordingPID)
	}
}
