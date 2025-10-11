package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trani",
	Short: "Audio recording with AI transcription and notes",
	Long:  `Trani records audio sessions, transcribes them using Whisper, and generates structured summaries using Claude AI.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
