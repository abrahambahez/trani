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
// write it into the session's note. It runs in a detached process spawned
// by SpawnPostprocess, decoupled from the recording lock.
func RunPostprocessWorker(ctx context.Context, sessionPath, sourcesTitle, promptTemplate string, preserveAudio bool, notifyID string, cfg *config.Config) error {
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

	notesPath := filepath.Join(sessionPath, "notas.md")
	notesContent, _ := os.ReadFile(notesPath)
	notes := strings.TrimSpace(string(notesContent))
	hasNotes := len(notes) > 0

	template := loadPromptTemplateStandalone(cfg.Paths.PromptsDir, promptTemplate, hasNotes)
	prompt := fillPromptTemplate(template, transcription, notes)

	resumen, err := llmClient.Generate(ctx, prompt)
	resumenPath := filepath.Join(sessionPath, "resumen.md")

	if err != nil {
		errorMsg := fmt.Sprintf("Error al generar resumen: %v", err)
		os.WriteFile(resumenPath, []byte(errorMsg), 0644)
	} else {
		os.WriteFile(resumenPath, []byte(resumen), 0644)
	}

	if !preserveAudio {
		if err := os.Remove(wavPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove archived audio: %w", err)
		}
	}

	doneMessage := fmt.Sprintf("Sesión completada - %s", filepath.Base(sessionPath))
	if notifyID != "" {
		notifier.Update(notifyID, "✅ Trani", doneMessage)
	} else {
		notifier.Info("✅ Trani", doneMessage)
	}

	return nil
}
