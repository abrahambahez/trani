package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newTestClaude(t *testing.T, handler http.HandlerFunc) *Claude {
	t.Helper()

	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	t.Cleanup(func() { os.Setenv("ANTHROPIC_API_KEY", originalKey) })

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &Claude{
		apiKey:    "test-key",
		model:     "claude-sonnet-5",
		maxTokens: 4000,
		client:    server.Client(),
		baseURL:   server.URL,
	}
}

// TestGenerate_SkipsThinkingBlock covers a real incident: adaptive thinking
// (the default on claude-sonnet-5 when the "thinking" param is omitted, as
// this client does) can put a "thinking" content block before the "text"
// block. Blindly returning content[0] returned the empty thinking block's
// text field, which then silently overwrote a user's session note with an
// empty string instead of the actual summary.
func TestGenerate_SkipsThinkingBlock(t *testing.T) {
	claude := newTestClaude(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"content": [
				{"type": "thinking", "text": ""},
				{"type": "text", "text": "the real summary"}
			]
		}`))
	})

	got, err := claude.Generate(context.Background(), "prompt")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if got != "the real summary" {
		t.Errorf("expected %q, got %q", "the real summary", got)
	}
}

func TestGenerate_NoTextBlock(t *testing.T) {
	claude := newTestClaude(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": [{"type": "thinking", "text": ""}]}`))
	})

	if _, err := claude.Generate(context.Background(), "prompt"); err == nil {
		t.Error("expected an error when the response has no text block")
	}
}

func TestGenerate_EmptyContent(t *testing.T) {
	claude := newTestClaude(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content": []}`))
	})

	if _, err := claude.Generate(context.Background(), "prompt"); err == nil {
		t.Error("expected an error when the response has no content blocks")
	}
}
