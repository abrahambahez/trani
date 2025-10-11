package transcribe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sabhz/trani/internal/config"
)

// OpenAI implements Transcriber using OpenAI Whisper API.
type OpenAI struct {
	apiKey   string
	model    string
	language string
	client   *http.Client
}

// NewOpenAI creates a new OpenAI transcriber.
// The apiKey parameter must be non-empty and model must be configured.
func NewOpenAI(cfg config.OpenAIConfig, apiKey string) *OpenAI {
	// Note: apiKey validation is done in New() factory function
	// Model validation is also done in New() factory function
	return &OpenAI{
		apiKey:   apiKey,
		model:    cfg.Model,
		language: cfg.Language,
		client:   &http.Client{},
	}
}

// openaiResponse represents the API response from OpenAI Whisper.
type openaiResponse struct {
	Text string `json:"text"`
}

// Transcribe converts audio to text using OpenAI Whisper API.
func (o *OpenAI) Transcribe(ctx context.Context, audioPath string) (string, error) {
	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key is required")
	}

	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return "", fmt.Errorf("audio file not found at %s", audioPath)
	}

	// Open the audio file
	file, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Create multipart form body
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the file field
	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", fmt.Errorf("failed to copy file data: %w", err)
	}

	// Add model field
	if err := writer.WriteField("model", o.model); err != nil {
		return "", fmt.Errorf("failed to write model field: %w", err)
	}

	// Add language field if specified
	if o.language != "" {
		if err := writer.WriteField("language", o.language); err != nil {
			return "", fmt.Errorf("failed to write language field: %w", err)
		}
	}

	// Close the writer to finalize the multipart message
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result openaiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Text, nil
}
