package cmd

import (
	"context"
	"fmt"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/session"
	"github.com/sabhz/trani/pkg/errlog"
	"github.com/sabhz/trani/pkg/notify"
	"github.com/spf13/cobra"
)

var recordWorkerPrompt string

// recordWorkerCmd is an internal command spawned as a detached process by
// session.Launch when Obsidian is configured (no editor process to block
// on); it is not meant to be invoked directly.
var recordWorkerCmd = &cobra.Command{
	Use:    "__record-worker",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.ExpandPaths()
		cfg.ApplyDefaults()

		sess, err := session.New(recordWorkerPrompt, cfg)
		if err != nil {
			notify.New().Error("⚠️ Trani", fmt.Sprintf("Error al iniciar la sesión: %v", err))
			errlog.Error("session_init", "", err)
			return err
		}

		if err := sess.Start(context.Background()); err != nil {
			// stdio is redirected to /dev/null in this detached process, so a
			// returned error would otherwise be invisible to the user.
			notify.New().Error("⚠️ Trani", fmt.Sprintf("Error en la sesión: %v", err))
			errlog.Error("session_start", sess.Title(), err)
			return err
		}

		return nil
	},
}

func init() {
	recordWorkerCmd.Flags().StringVar(&recordWorkerPrompt, "prompt", "default", "Prompt template name")
	rootCmd.AddCommand(recordWorkerCmd)
}
