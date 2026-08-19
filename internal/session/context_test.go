package session

import (
	"strings"
	"testing"
)

func TestBuildTranscriptionPrompt(t *testing.T) {
	t.Run("no frontmatter", func(t *testing.T) {
		got := buildTranscriptionPrompt("## Notas\n\nAlgo que anoté.")
		if got != "" {
			t.Errorf("expected empty prompt, got %q", got)
		}
	})

	t.Run("malformed yaml", func(t *testing.T) {
		got := buildTranscriptionPrompt("---\nasistentes: [unterminated\n---\n\n## Notas")
		if got != "" {
			t.Errorf("expected empty prompt, got %q", got)
		}
	})

	t.Run("attendees and purpose flattened", func(t *testing.T) {
		note := "---\nasistentes:\n  - Abraham Bahez\n  - Trani\npropósito: Test de Trani\n---\n\n## Notas\n"
		got := buildTranscriptionPrompt(note)
		if !strings.Contains(got, "Abraham Bahez, Trani") {
			t.Errorf("expected attendee list in prompt, got %q", got)
		}
		if !strings.Contains(got, "Test de Trani") {
			t.Errorf("expected purpose in prompt, got %q", got)
		}
	})

	t.Run("long field dropped before short fields are cut", func(t *testing.T) {
		longField := strings.Repeat("palabra ", 200) // well over maxPromptChars
		note := "---\nexcerpt: " + longField + "\nasistentes: Trani, Abraham\n---\n"
		got := buildTranscriptionPrompt(note)
		if !strings.Contains(got, "Trani, Abraham") {
			t.Errorf("expected short field to survive truncation, got %q", got)
		}
		if strings.Contains(got, longField) {
			t.Errorf("expected long field to be dropped, got %q", got)
		}
	})
}

func TestExtractFrontmatter(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		expected string
		ok       bool
	}{
		{
			name:     "valid frontmatter",
			content:  "---\nasistentes: []\n---\n\n## Notas",
			expected: "asistentes: []",
			ok:       true,
		},
		{
			name:     "no frontmatter",
			content:  "## Notas\n\nAlgo.",
			expected: "",
			ok:       false,
		},
		{
			name:     "unterminated frontmatter",
			content:  "---\nasistentes: []\n\n## Notas",
			expected: "",
			ok:       false,
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
			ok:       false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := extractFrontmatter(c.content)
			if ok != c.ok || got != c.expected {
				t.Errorf("expected (%q, %v), got (%q, %v)", c.expected, c.ok, got, ok)
			}
		})
	}
}
