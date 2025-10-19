package cmd

import (
	"context"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/session"
	"github.com/spf13/cobra"
)

var (
	processNotes  string
	processTitle  string
	processPrompt string
)

var processCmd = &cobra.Command{
	Use:   "process <audio-file>",
	Short: "Process an existing audio file with transcription and summary",
	Long:  `Process an existing audio file by transcribing it and generating a structured summary. Optionally provide notes to enhance the summary.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		audioPath := args[0]

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.ExpandPaths()
		cfg.ApplyDefaults()

		return session.ProcessFile(
			context.Background(),
			audioPath,
			processNotes,
			processTitle,
			processPrompt,
			cfg,
		)
	},
}

func init() {
	processCmd.Flags().StringVar(&processNotes, "notes", "", "Path to notes file")
	processCmd.Flags().StringVar(&processTitle, "title", "", "Output directory title (defaults to audio filename)")
	processCmd.Flags().StringVar(&processPrompt, "prompt", "default", "Prompt template name")
	rootCmd.AddCommand(processCmd)
}
