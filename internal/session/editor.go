package session

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/pkg/errlog"
)

// obsidianOpenTimeout bounds the call opening Obsidian's URI so a stuck
// desktop URI handler can't block the whole session from ever reaching the
// point where it listens for a stop signal.
const obsidianOpenTimeout = 10 * time.Second

// buildObsidianURI builds the "obsidian://open" URI for a note, given the
// vault root and the note's absolute path (which must be inside the vault).
// The vault name is the vault directory's basename, and the file parameter
// is the note's path relative to the vault root, without its extension —
// both are how Obsidian's own URI handler expects them.
func buildObsidianURI(vaultPath, notePath string) (string, error) {
	relPath, err := filepath.Rel(vaultPath, notePath)
	if err != nil {
		return "", fmt.Errorf("note path is not inside the configured vault: %w", err)
	}
	relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))

	vaultName := filepath.Base(vaultPath)

	return fmt.Sprintf(
		"obsidian://open?vault=%s&file=%s",
		url.QueryEscape(vaultName),
		url.QueryEscape(relPath),
	), nil
}

// openNote opens the session note in Obsidian, non-blocking since Obsidian
// is a separate GUI app; the caller waits for an explicit stop signal
// instead of an editor process.
//
// Opening goes through Obsidian's own "obsidian://open" URI, dispatched via
// xdg-open, not the obsidian-cli's "open" subcommand: on this setup (a
// flatpak install) the CLI's own IPC to a running instance was unreliable —
// it failed outright while Obsidian was open, and silently ignored the
// requested note on a cold start, just restoring whatever was last open.
// The URI handler correctly jumped to the requested note while Obsidian was
// already running (not separately verified on a cold start).
//
// A failure to open the URI degrades to a warning instead of failing the
// session: the note file already exists on disk and trani can read/write
// it directly regardless of whether the GUI ever opened it.
func openNote(ctx context.Context, notePath string, cfg *config.Config) error {
	uri, err := buildObsidianURI(cfg.Obsidian.VaultPath, notePath)
	if err != nil {
		return err
	}

	openCtx, cancel := context.WithTimeout(ctx, obsidianOpenTimeout)
	defer cancel()

	if err := exec.CommandContext(openCtx, "xdg-open", uri).Run(); err != nil {
		sessionTitle := strings.TrimSuffix(filepath.Base(notePath), filepath.Ext(notePath))
		errlog.Error("obsidian_open", sessionTitle, err)
	}

	return nil
}
