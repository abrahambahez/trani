package session

import "testing"

func TestAppendResumenSection(t *testing.T) {
	cases := []struct {
		name            string
		existingContent string
		resumen         string
		expected        string
	}{
		{
			name:            "empty note",
			existingContent: "",
			resumen:         "El resumen generado.",
			expected:        "## Resumen\n\nEl resumen generado.",
		},
		{
			name:            "preserves frontmatter and Notas section",
			existingContent: "---\nasistentes: []\n---\n\n## Notas\n\nAlgo que anoté.",
			resumen:         "El resumen generado.",
			expected:        "---\nasistentes: []\n---\n\n## Notas\n\nAlgo que anoté.\n\n## Resumen\n\nEl resumen generado.",
		},
		{
			name:            "trims trailing newlines before appending",
			existingContent: "## Notas\n\nAlgo.\n\n\n",
			resumen:         "El resumen generado.",
			expected:        "## Notas\n\nAlgo.\n\n## Resumen\n\nEl resumen generado.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := appendResumenSection(c.existingContent, c.resumen)
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}
