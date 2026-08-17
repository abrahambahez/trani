package cmd

import (
	"fmt"

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

		lock, err := session.ReadLock(cfg)
		if err != nil {
			return err
		}
		if lock == nil {
			return fmt.Errorf("no active session found")
		}

		return session.SignalStop(lock)
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
