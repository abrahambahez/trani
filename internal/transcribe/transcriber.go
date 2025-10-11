package transcribe

import (
	"context"
	"fmt"
	"os"

	"github.com/sabhz/trani/internal/config"
)

// Transcriber converts audio files to text.
type Transcriber interface {
	Transcribe(ctx context.Context, audioPath string) (string, error)
}

// New creates a Transcriber based on the configured backend.
// Returns error if backend is unknown or required configuration is missing.
func New(cfg config.TranscriptionConfig) (Transcriber, error) {
	if cfg.Backend == "" {
		return nil, fmt.Errorf("transcription backend not configured")
	}

	switch cfg.Backend {
	case "local":
		return NewWhisperLocal(cfg.Local)
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		if cfg.OpenAI.Model == "" {
			return nil, fmt.Errorf("OpenAI model not configured")
		}
		return NewOpenAI(cfg.OpenAI, apiKey), nil
	default:
		return nil, fmt.Errorf("unknown transcription backend: %s (supported: local, openai)", cfg.Backend)
	}
}
