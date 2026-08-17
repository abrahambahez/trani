package cmd

import (
	"context"
	"fmt"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/session"
	"github.com/sabhz/trani/pkg/notify"
	"github.com/spf13/cobra"
)

var (
	postprocessSessionPath   string
	postprocessSourcesTitle  string
	postprocessPrompt        string
	postprocessPreserveAudio bool
	postprocessNotifyID      string
)

// postprocessWorkerCmd is an internal command spawned as a detached process
// by Session.finishRecording; it is not meant to be invoked directly.
var postprocessWorkerCmd = &cobra.Command{
	Use:    "__postprocess-worker",
	Hidden: true,
	Args:   cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		cfg.ExpandPaths()
		cfg.ApplyDefaults()

		err = session.RunPostprocessWorker(
			context.Background(),
			postprocessSessionPath,
			postprocessSourcesTitle,
			postprocessPrompt,
			postprocessPreserveAudio,
			postprocessNotifyID,
			cfg,
		)
		if err != nil {
			// stdio is redirected to /dev/null in this detached process, so a
			// returned error would otherwise be invisible to the user.
			notify.New().Error("⚠️ Trani", fmt.Sprintf("Error procesando la sesión: %v", err))
		}
		return err
	},
}

func init() {
	postprocessWorkerCmd.Flags().StringVar(&postprocessSessionPath, "session-path", "", "Path to the session directory")
	postprocessWorkerCmd.Flags().StringVar(&postprocessSourcesTitle, "sources-title", "", "Original session timestamp, used to locate .sources/<title>.txt and .wav")
	postprocessWorkerCmd.Flags().StringVar(&postprocessPrompt, "prompt", "default", "Prompt template name")
	postprocessWorkerCmd.Flags().BoolVar(&postprocessPreserveAudio, "preserve-audio", false, "Keep audio file after processing")
	postprocessWorkerCmd.Flags().StringVar(&postprocessNotifyID, "notify-id", "", "Notification ID to update on completion")
	postprocessWorkerCmd.MarkFlagRequired("session-path")
	postprocessWorkerCmd.MarkFlagRequired("sources-title")
	rootCmd.AddCommand(postprocessWorkerCmd)
}
