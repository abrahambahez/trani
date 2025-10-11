package transcribe

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sabhz/trani/internal/config"
)

// Test New() factory function

func TestNew_EmptyBackend(t *testing.T) {
	cfg := config.TranscriptionConfig{}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error for empty backend")
	}
}

func TestNew_UnknownBackend(t *testing.T) {
	cfg := config.TranscriptionConfig{
		Backend: "unknown",
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error for unknown backend")
	}
}

func TestNew_LocalBackend_MissingBinaryPath(t *testing.T) {
	cfg := config.TranscriptionConfig{
		Backend: "local",
		Local: config.LocalWhisperConfig{
			ModelPath: "/path/to/model",
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error when binary path is missing")
	}
}

func TestNew_LocalBackend_MissingModelPath(t *testing.T) {
	cfg := config.TranscriptionConfig{
		Backend: "local",
		Local: config.LocalWhisperConfig{
			BinaryPath: "/path/to/binary",
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error when model path is missing")
	}
}

func TestNew_LocalBackend_ValidConfig(t *testing.T) {
	cfg := config.TranscriptionConfig{
		Backend: "local",
		Local: config.LocalWhisperConfig{
			BinaryPath: "/usr/bin/whisper",
			ModelPath:  "/models/model.bin",
			Threads:    4,
			Language:   "en",
		},
	}

	transcriber, err := New(cfg)
	if err != nil {
		t.Fatalf("New() should not error with valid local config: %v", err)
	}

	if transcriber == nil {
		t.Fatal("New() returned nil transcriber")
	}

	whisper, ok := transcriber.(*WhisperLocal)
	if !ok {
		t.Fatal("New() should return WhisperLocal for local backend")
	}

	if whisper.binaryPath != "/usr/bin/whisper" {
		t.Errorf("binaryPath: expected /usr/bin/whisper, got %s", whisper.binaryPath)
	}
	if whisper.modelPath != "/models/model.bin" {
		t.Errorf("modelPath: expected /models/model.bin, got %s", whisper.modelPath)
	}
	if whisper.threads != 4 {
		t.Errorf("threads: expected 4, got %d", whisper.threads)
	}
	if whisper.language != "en" {
		t.Errorf("language: expected en, got %s", whisper.language)
	}
}

func TestNew_OpenAIBackend_MissingAPIKey(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", originalKey)

	os.Unsetenv("OPENAI_API_KEY")

	cfg := config.TranscriptionConfig{
		Backend: "openai",
		OpenAI: config.OpenAIConfig{
			Model: "whisper-1",
		},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error when OPENAI_API_KEY is not set")
	}
}

func TestNew_OpenAIBackend_MissingModel(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", originalKey)

	os.Setenv("OPENAI_API_KEY", "test-key")

	cfg := config.TranscriptionConfig{
		Backend: "openai",
		OpenAI:  config.OpenAIConfig{},
	}

	_, err := New(cfg)
	if err == nil {
		t.Fatal("New() should return error when model is not configured")
	}
}

func TestNew_OpenAIBackend_ValidConfig(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", originalKey)

	os.Setenv("OPENAI_API_KEY", "test-api-key")

	cfg := config.TranscriptionConfig{
		Backend: "openai",
		OpenAI: config.OpenAIConfig{
			Model:    "whisper-1",
			Language: "en",
		},
	}

	transcriber, err := New(cfg)
	if err != nil {
		t.Fatalf("New() should not error with valid OpenAI config: %v", err)
	}

	if transcriber == nil {
		t.Fatal("New() returned nil transcriber")
	}

	openai, ok := transcriber.(*OpenAI)
	if !ok {
		t.Fatal("New() should return OpenAI for openai backend")
	}

	if openai.apiKey != "test-api-key" {
		t.Errorf("apiKey: expected test-api-key, got %s", openai.apiKey)
	}
	if openai.model != "whisper-1" {
		t.Errorf("model: expected whisper-1, got %s", openai.model)
	}
	if openai.language != "en" {
		t.Errorf("language: expected en, got %s", openai.language)
	}
}

// Test WhisperLocal

func TestWhisperLocal_TranscribeMissingBinary(t *testing.T) {
	tempDir := t.TempDir()

	whisper := &WhisperLocal{
		binaryPath: filepath.Join(tempDir, "nonexistent-binary"),
		modelPath:  filepath.Join(tempDir, "model.bin"),
		threads:    4,
		language:   "en",
	}

	// Create a dummy audio file
	audioPath := filepath.Join(tempDir, "test.wav")
	if err := os.WriteFile(audioPath, []byte("dummy audio"), 0644); err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}

	_, err := whisper.Transcribe(context.Background(), audioPath)
	if err == nil {
		t.Fatal("Transcribe() should error when binary doesn't exist")
	}
}

func TestWhisperLocal_TranscribeMissingModel(t *testing.T) {
	tempDir := t.TempDir()

	// Create a dummy binary file
	binaryPath := filepath.Join(tempDir, "whisper")
	if err := os.WriteFile(binaryPath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	whisper := &WhisperLocal{
		binaryPath: binaryPath,
		modelPath:  filepath.Join(tempDir, "nonexistent-model.bin"),
		threads:    4,
		language:   "en",
	}

	// Create a dummy audio file
	audioPath := filepath.Join(tempDir, "test.wav")
	if err := os.WriteFile(audioPath, []byte("dummy audio"), 0644); err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}

	_, err := whisper.Transcribe(context.Background(), audioPath)
	if err == nil {
		t.Fatal("Transcribe() should error when model doesn't exist")
	}
}

func TestWhisperLocal_TranscribeMissingAudioFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create dummy binary and model
	binaryPath := filepath.Join(tempDir, "whisper")
	if err := os.WriteFile(binaryPath, []byte("dummy"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	modelPath := filepath.Join(tempDir, "model.bin")
	if err := os.WriteFile(modelPath, []byte("dummy"), 0644); err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	whisper := &WhisperLocal{
		binaryPath: binaryPath,
		modelPath:  modelPath,
		threads:    4,
		language:   "en",
	}

	_, err := whisper.Transcribe(context.Background(), filepath.Join(tempDir, "nonexistent.wav"))
	if err == nil {
		t.Fatal("Transcribe() should error when audio file doesn't exist")
	}
}

// Test OpenAI

func TestOpenAI_TranscribeMissingAPIKey(t *testing.T) {
	tempDir := t.TempDir()

	openai := &OpenAI{
		apiKey:   "",
		model:    "whisper-1",
		language: "en",
	}

	// Create a dummy audio file
	audioPath := filepath.Join(tempDir, "test.wav")
	if err := os.WriteFile(audioPath, []byte("dummy audio"), 0644); err != nil {
		t.Fatalf("Failed to create test audio file: %v", err)
	}

	_, err := openai.Transcribe(context.Background(), audioPath)
	if err == nil {
		t.Fatal("Transcribe() should error when API key is empty")
	}
}

func TestOpenAI_TranscribeMissingAudioFile(t *testing.T) {
	tempDir := t.TempDir()

	openai := &OpenAI{
		apiKey:   "test-key",
		model:    "whisper-1",
		language: "en",
	}

	_, err := openai.Transcribe(context.Background(), filepath.Join(tempDir, "nonexistent.wav"))
	if err == nil {
		t.Fatal("Transcribe() should error when audio file doesn't exist")
	}
}

func TestNewWhisperLocal_ValidatesConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.LocalWhisperConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: config.LocalWhisperConfig{
				BinaryPath: "/usr/bin/whisper",
				ModelPath:  "/models/model.bin",
				Threads:    4,
				Language:   "en",
			},
			wantErr: false,
		},
		{
			name: "missing binary path",
			cfg: config.LocalWhisperConfig{
				ModelPath: "/models/model.bin",
			},
			wantErr: true,
		},
		{
			name: "missing model path",
			cfg: config.LocalWhisperConfig{
				BinaryPath: "/usr/bin/whisper",
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			cfg:     config.LocalWhisperConfig{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWhisperLocal(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWhisperLocal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewOpenAI_CreatesClient(t *testing.T) {
	cfg := config.OpenAIConfig{
		Model:    "whisper-1",
		Language: "en",
	}

	openai := NewOpenAI(cfg, "test-api-key")

	if openai == nil {
		t.Fatal("NewOpenAI() returned nil")
	}

	if openai.apiKey != "test-api-key" {
		t.Errorf("apiKey: expected test-api-key, got %s", openai.apiKey)
	}
	if openai.model != "whisper-1" {
		t.Errorf("model: expected whisper-1, got %s", openai.model)
	}
	if openai.language != "en" {
		t.Errorf("language: expected en, got %s", openai.language)
	}
	if openai.client == nil {
		t.Error("client should not be nil")
	}
}
