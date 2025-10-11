package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sabhz/trani/internal/config"
)

// Claude is a client for the Claude API.
type Claude struct {
	apiKey    string
	model     string
	maxTokens int
	client    *http.Client
}

// New creates a new Claude API client.
func New(cfg config.ClaudeConfig, apiKey string) *Claude {
	return &Claude{
		apiKey:    apiKey,
		model:     cfg.Model,
		maxTokens: cfg.MaxTokens,
		client:    &http.Client{},
	}
}

// claudeRequest represents the request body for Claude API.
type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []claudeMessage `json:"messages"`
}

// claudeMessage represents a message in the conversation.
type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse represents the response from Claude API.
type claudeResponse struct {
	Content []claudeContent `json:"content"`
	Error   *claudeError    `json:"error,omitempty"`
}

// claudeContent represents a content block in the response.
type claudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// claudeError represents an error response from Claude API.
type claudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Generate sends a prompt to Claude and returns the response text.
func (c *Claude) Generate(ctx context.Context, prompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp claudeResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error != nil {
			return "", fmt.Errorf("Claude API error: %s", errResp.Error.Message)
		}
		return "", fmt.Errorf("Claude API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result claudeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content in Claude response")
	}

	return result.Content[0].Text, nil
}
