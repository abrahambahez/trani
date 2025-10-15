package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sabhz/trani/internal/config"
)

type Ollama struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllama(cfg config.OllamaConfig) (Generator, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("ollama model not configured")
	}

	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	return &Ollama{
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{},
	}, nil
}

type ollamaRequest struct {
	Model    string           `json:"model"`
	Messages []ollamaMessage  `json:"messages"`
	Stream   bool             `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error,omitempty"`
}

func (o *Ollama) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model: o.model,
		Messages: []ollamaMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := o.baseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ollamaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("ollama API error: %s", result.Error)
	}

	return result.Message.Content, nil
}
