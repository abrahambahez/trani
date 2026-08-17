package session

import "testing"

func TestBuildObsidianURI(t *testing.T) {
	cases := []struct {
		name      string
		vaultPath string
		notePath  string
		expected  string
	}{
		{
			name:      "note directly in vault root",
			vaultPath: "/home/user/vault",
			notePath:  "/home/user/vault/2026-08-16.md",
			expected:  "obsidian://open?vault=vault&file=2026-08-16",
		},
		{
			name:      "note nested in a subdirectory",
			vaultPath: "/home/user/vault",
			notePath:  "/home/user/vault/logs/2026-08-16-1200.md",
			expected:  "obsidian://open?vault=vault&file=logs%2F2026-08-16-1200",
		},
		{
			name:      "vault name with a space",
			vaultPath: "/home/user/my vault",
			notePath:  "/home/user/my vault/note.md",
			expected:  "obsidian://open?vault=my+vault&file=note",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := buildObsidianURI(c.vaultPath, c.notePath)
			if err != nil {
				t.Fatalf("buildObsidianURI failed: %v", err)
			}
			if got != c.expected {
				t.Errorf("expected %q, got %q", c.expected, got)
			}
		})
	}
}

func TestBuildObsidianURINoteOutsideVault(t *testing.T) {
	if _, err := buildObsidianURI("/home/user/vault", "relative/path.md"); err == nil {
		t.Error("expected an error when the note path can't be made relative to the vault")
	}
}
