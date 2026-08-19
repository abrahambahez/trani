package cmd

import (
	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/session"
	"github.com/spf13/cobra"
)

var startPrompt string

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a new recording session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.ExpandPaths()
		cfg.ApplyDefaults()

		return session.Launch(startPrompt, cfg)
	},
}

func init() {
	startCmd.Flags().StringVar(&startPrompt, "prompt", "default", "Prompt template name")
	rootCmd.AddCommand(startCmd)
}
