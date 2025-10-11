package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with missing file should not error, got: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, ".config", "trani")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	invalidYAML := "this is not: valid: yaml: content:"
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid YAML")
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)

	configDir := filepath.Join(tempDir, ".config", "trani")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	validYAML := `
transcription:
  backend: local
  local:
    model_path: ~/whisper.cpp/models/model.bin
    binary_path: ~/whisper.cpp/whisper-cli
    threads: 8
    language: es
audio:
  sample_rate: 16000
  channels: 1
paths:
  sessions_dir: ~/trani/sessions
`
	if err := os.WriteFile(configPath, []byte(validYAML), 0644); err != nil {
		t.Fatalf("Failed to write valid YAML: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed with valid YAML: %v", err)
	}

	if cfg.Transcription.Backend != "local" {
		t.Errorf("Expected backend 'local', got '%s'", cfg.Transcription.Backend)
	}
	if cfg.Transcription.Local.Threads != 8 {
		t.Errorf("Expected threads 8, got %d", cfg.Transcription.Local.Threads)
	}
	if cfg.Audio.SampleRate != 16000 {
		t.Errorf("Expected sample rate 16000, got %d", cfg.Audio.SampleRate)
	}
}

func TestExpandPaths(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", "/home/testuser")

	cfg := &Config{
		Transcription: TranscriptionConfig{
			Local: LocalWhisperConfig{
				ModelPath:  "~/models/model.bin",
				BinaryPath: "~/bin/whisper",
			},
		},
		Paths: PathsConfig{
			SessionsDir: "~/sessions",
			TempDir:     "~/temp",
			PromptsDir:  "~/prompts",
		},
	}

	cfg.ExpandPaths()

	expected := map[string]string{
		"ModelPath":   "/home/testuser/models/model.bin",
		"BinaryPath":  "/home/testuser/bin/whisper",
		"SessionsDir": "/home/testuser/sessions",
		"TempDir":     "/home/testuser/temp",
		"PromptsDir":  "/home/testuser/prompts",
	}

	if cfg.Transcription.Local.ModelPath != expected["ModelPath"] {
		t.Errorf("ModelPath: expected %s, got %s", expected["ModelPath"], cfg.Transcription.Local.ModelPath)
	}
	if cfg.Transcription.Local.BinaryPath != expected["BinaryPath"] {
		t.Errorf("BinaryPath: expected %s, got %s", expected["BinaryPath"], cfg.Transcription.Local.BinaryPath)
	}
	if cfg.Paths.SessionsDir != expected["SessionsDir"] {
		t.Errorf("SessionsDir: expected %s, got %s", expected["SessionsDir"], cfg.Paths.SessionsDir)
	}
	if cfg.Paths.TempDir != expected["TempDir"] {
		t.Errorf("TempDir: expected %s, got %s", expected["TempDir"], cfg.Paths.TempDir)
	}
	if cfg.Paths.PromptsDir != expected["PromptsDir"] {
		t.Errorf("PromptsDir: expected %s, got %s", expected["PromptsDir"], cfg.Paths.PromptsDir)
	}
}

func TestApplyDefaults(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", "/home/testuser")

	cfg := &Config{}
	cfg.ApplyDefaults()

	expected := map[string]string{
		"SessionsDir": "/home/testuser/.config/trani/sessions",
		"TempDir":     "/home/testuser/.config/trani/temp",
		"PromptsDir":  "/home/testuser/.config/trani/prompts",
	}

	if cfg.Paths.SessionsDir != expected["SessionsDir"] {
		t.Errorf("SessionsDir: expected %s, got %s", expected["SessionsDir"], cfg.Paths.SessionsDir)
	}
	if cfg.Paths.TempDir != expected["TempDir"] {
		t.Errorf("TempDir: expected %s, got %s", expected["TempDir"], cfg.Paths.TempDir)
	}
	if cfg.Paths.PromptsDir != expected["PromptsDir"] {
		t.Errorf("PromptsDir: expected %s, got %s", expected["PromptsDir"], cfg.Paths.PromptsDir)
	}
}

func TestApplyDefaults_PreservesExistingValues(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", "/home/testuser")

	cfg := &Config{
		Paths: PathsConfig{
			SessionsDir: "/custom/sessions",
		},
	}
	cfg.ApplyDefaults()

	if cfg.Paths.SessionsDir != "/custom/sessions" {
		t.Errorf("SessionsDir should be preserved, got %s", cfg.Paths.SessionsDir)
	}
	if cfg.Paths.TempDir != "/home/testuser/.config/trani/temp" {
		t.Errorf("TempDir should be defaulted, got %s", cfg.Paths.TempDir)
	}
}
