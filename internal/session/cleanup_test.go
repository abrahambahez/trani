package session

import "testing"

func TestRemoveConsecutiveDuplicateLines(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no duplicates",
			input:    "hola\nmundo\nchau",
			expected: "hola\nmundo\nchau",
		},
		{
			name:     "consecutive duplicate removed",
			input:    "hola\nhola\nmundo",
			expected: "hola\nmundo",
		},
		{
			name:     "non-consecutive duplicate kept",
			input:    "hola\nmundo\nhola",
			expected: "hola\nmundo\nhola",
		},
		{
			name:     "multiple consecutive duplicates collapse to one",
			input:    "hola\nhola\nhola\nmundo",
			expected: "hola\nmundo",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := removeConsecutiveDuplicateLines(c.input)
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}
