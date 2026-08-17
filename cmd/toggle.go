package cmd

import (
	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/session"
	"github.com/spf13/cobra"
)

var (
	togglePrompt        string
	togglePreserveAudio bool
)

var toggleCmd = &cobra.Command{
	Use:   "toggle",
	Short: "Toggle recording session (start if inactive, stop if active)",
	Args:  cobra.NoArgs,
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
		if lock != nil {
			return session.SignalStop(lock)
		}

		return session.Launch(togglePrompt, togglePreserveAudio, cfg)
	},
}

func init() {
	toggleCmd.Flags().StringVar(&togglePrompt, "prompt", "default", "Prompt template name")
	toggleCmd.Flags().BoolVar(&togglePreserveAudio, "preserve-audio", false, "Keep audio file after processing")
	rootCmd.AddCommand(toggleCmd)
}
