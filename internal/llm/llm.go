package llm

import (
	"context"
	"fmt"

	"github.com/sabhz/trani/internal/config"
)

type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

func New(cfg config.LLMConfig) (Generator, error) {
	if cfg.Backend == "" {
		return nil, fmt.Errorf("llm backend not configured")
	}

	switch cfg.Backend {
	case "claude":
		return NewClaude(cfg.Claude)
	case "ollama":
		return NewOllama(cfg.Ollama)
	default:
		return nil, fmt.Errorf("unknown llm backend: %s (supported: claude, ollama)", cfg.Backend)
	}
}
