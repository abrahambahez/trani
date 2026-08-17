package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sabhz/trani/internal/config"
)

// obsidianOpenTimeout bounds the "obsidian open" call so a stuck or
// unreachable Obsidian instance can't block the whole session from ever
// reaching the point where it listens for a stop signal.
const obsidianOpenTimeout = 15 * time.Second

// openNote opens the session note for the user. With no Obsidian vault
// configured, it falls back to nvim in the foreground and returns the
// running *exec.Cmd for the caller to wait on. With a vault configured, it
// opens the note in Obsidian (non-blocking, since Obsidian is a separate
// GUI app) and returns a nil *exec.Cmd, signaling the caller should wait
// for an explicit stop signal instead of an editor process.
//
// A failure to launch Obsidian (app not running, CLI missing) degrades to
// a warning instead of failing the session: the note file already exists
// on disk and trani can read/write it directly regardless of whether the
// GUI ever opened it.
func openNote(ctx context.Context, notePath string, cfg *config.Config) (*exec.Cmd, error) {
	if cfg.Obsidian.VaultPath == "" {
		cmd := exec.CommandContext(ctx, "nvim", notePath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start editor: %w", err)
		}

		return cmd, nil
	}

	relPath, err := filepath.Rel(cfg.Obsidian.VaultPath, notePath)
	if err != nil {
		return nil, fmt.Errorf("note path is not inside the configured vault: %w", err)
	}

	openCtx, cancel := context.WithTimeout(ctx, obsidianOpenTimeout)
	defer cancel()

	cliCmd := exec.CommandContext(openCtx, cfg.Obsidian.CLIPath, "open", "path="+relPath)
	if err := cliCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "trani: warning: failed to open note in Obsidian: %v\n", err)
	}

	return nil, nil
}
