package session

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// maxPromptChars approximates the ~224-token prompt budget of the OpenAI
// transcription API in characters, since no tokenizer is available here.
const maxPromptChars = 700

// buildTranscriptionPrompt extracts a note's YAML frontmatter (attendees,
// purpose, etc., filled in by an external note-taking template before trani
// ever touches the file) and flattens it into a short prompt to bias
// Whisper's word/spelling choices on ambiguous audio, e.g. proper nouns.
// Returns "" if there's no frontmatter or it doesn't parse — this must
// never block transcription.
func buildTranscriptionPrompt(noteContent string) string {
	frontmatter, ok := extractFrontmatter(noteContent)
	if !ok {
		return ""
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &raw); err != nil {
		return ""
	}

	values := flattenValues(raw)

	// Shortest first: a long free-text field (e.g. an excerpt) must never
	// crowd out the short names/terms that make this worth doing.
	sort.Slice(values, func(i, j int) bool {
		if len(values[i]) != len(values[j]) {
			return len(values[i]) < len(values[j])
		}
		return values[i] < values[j]
	})

	var parts []string
	total := 0
	for _, v := range values {
		added := len(v) + 2 // ". " separator
		if total+added > maxPromptChars {
			break // everything after this is equal or longer
		}
		parts = append(parts, v)
		total += added
	}

	return strings.Join(parts, ". ")
}

// flattenValues extracts string and string-list values from a parsed
// frontmatter map. Nested maps, booleans, and numbers aren't useful
// vocabulary for a transcription prompt, so they're skipped.
func flattenValues(raw map[string]interface{}) []string {
	var values []string
	for _, v := range raw {
		switch val := v.(type) {
		case string:
			if s := strings.TrimSpace(val); s != "" {
				values = append(values, s)
			}
		case []interface{}:
			var items []string
			for _, item := range val {
				if s, ok := item.(string); ok {
					if s = strings.TrimSpace(s); s != "" {
						items = append(items, s)
					}
				}
			}
			if len(items) > 0 {
				values = append(values, strings.Join(items, ", "))
			}
		}
	}
	return values
}

// extractFrontmatter returns the YAML block delimited by a leading "---"
// line and the next line that is exactly "---". ok is false if content
// doesn't start with a frontmatter block at all.
func extractFrontmatter(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], "\n"), true
		}
	}

	return "", false
}
