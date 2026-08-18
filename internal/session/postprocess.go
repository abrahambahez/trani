package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/llm"
	"github.com/sabhz/trani/pkg/notify"
)

// RunPostprocessWorker finalizes a session whose recording has already
// stopped and whose audio was already transcribed progressively, chunk by
// chunk, by the chunker while the session was live. It only needs to clean
// up the accumulated transcript, generate the structured summary, and
// overwrite the session note with it (the user's raw notes and the final
// note are the same file). It runs in a detached process spawned by
// SpawnPostprocess, decoupled from the recording lock.
func RunPostprocessWorker(ctx context.Context, notePath, sourcesTitle, promptTemplate string, preserveAudio bool, notifyID string, cfg *config.Config) error {
	notifier := notify.New()

	llmClient, err := llm.New(cfg.LLM)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	sourcesDir := filepath.Join(cfg.Paths.SessionsDir, ".sources")
	txtPath := filepath.Join(sourcesDir, sourcesTitle+".txt")
	wavPath := filepath.Join(sourcesDir, sourcesTitle+".wav")

	rawTranscription, err := os.ReadFile(txtPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read transcription: %w", err)
	}

	transcription := removeConsecutiveDuplicateLines(strings.TrimSpace(string(rawTranscription)))

	notesContent, _ := os.ReadFile(notePath)
	notes := strings.TrimSpace(string(notesContent))
	hasNotes := len(notes) > 0

	sessionTitle := strings.TrimSuffix(filepath.Base(notePath), filepath.Ext(notePath))

	template, err := loadPromptTemplateStandalone(cfg.Paths.PromptsDir, promptTemplate, hasNotes)
	if err != nil {
		notifier.Error("⚠️ Trani", fmt.Sprintf("Error al cargar plantilla de prompt (%s): %v", sessionTitle, err))
		return nil
	}
	prompt := fillPromptTemplate(template, transcription, notes)

	resumen, err := llmClient.Generate(ctx, prompt)

	if err == nil && strings.TrimSpace(resumen) == "" {
		err = fmt.Errorf("the model returned an empty summary")
	}

	if err != nil {
		// Leave the user's raw notes untouched on failure; only the
		// notification reports the problem. Handled here (not returned as
		// an error) so the caller doesn't also fire its generic failure
		// notification on top of this specific one.
		notifier.Error("⚠️ Trani", fmt.Sprintf("Error al generar resumen (%s): %v", sessionTitle, err))
		return nil
	}

	if err := os.WriteFile(notePath, []byte(resumen), 0644); err != nil {
		return fmt.Errorf("failed to update session note: %w", err)
	}

	if !preserveAudio {
		if err := os.Remove(wavPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove archived audio: %w", err)
		}
	}

	doneMessage := fmt.Sprintf("Sesión completada - %s", sessionTitle)
	if notifyID != "" {
		notifier.Update(notifyID, "✅ Trani", doneMessage)
	} else {
		notifier.Info("✅ Trani", doneMessage)
	}

	return nil
}
