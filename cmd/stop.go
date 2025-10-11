package cmd

import (
	"context"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/session"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the active recording session",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.ExpandPaths()
		cfg.ApplyDefaults()

		sess, err := session.LoadActive(cfg)
		if err != nil {
			return err
		}

		return sess.Stop(context.Background())
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
