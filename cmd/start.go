package cmd

import (
	"context"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/session"
	"github.com/spf13/cobra"
)

var (
	startPrompt       string
	startPreserveAudio bool
)

var startCmd = &cobra.Command{
	Use:   "start [title]",
	Short: "Start a new recording session",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.ExpandPaths()
		cfg.ApplyDefaults()

		title := ""
		if len(args) > 0 {
			title = args[0]
		}

		sess, err := session.New(title, startPrompt, startPreserveAudio, cfg)
		if err != nil {
			return err
		}

		return sess.Start(context.Background())
	},
}

func init() {
	startCmd.Flags().StringVar(&startPrompt, "prompt", "default", "Prompt template name")
	startCmd.Flags().BoolVar(&startPreserveAudio, "preserve-audio", false, "Keep audio file after processing")
	rootCmd.AddCommand(startCmd)
}
