package session

import "strings"

// removeConsecutiveDuplicateLines drops any line that is exactly equal to
// the line right before it. Whisper occasionally hallucinates by repeating
// the same line verbatim, especially across chunk boundaries.
func removeConsecutiveDuplicateLines(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))

	var prev string
	for i, line := range lines {
		if i > 0 && line == prev {
			continue
		}
		out = append(out, line)
		prev = line
	}

	return strings.Join(out, "\n")
}
