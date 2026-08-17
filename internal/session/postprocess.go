package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/llm"
	"github.com/sabhz/trani/internal/transcribe"
	"github.com/sabhz/trani/pkg/notify"
)

// RunPostprocessWorker transcribes, summarizes, and finalizes a session
// whose recording has already stopped. It is meant to run in a detached
// process spawned by SpawnPostprocess, decoupled from the recording lock.
func RunPostprocessWorker(ctx context.Context, sessionPath, promptTemplate string, preserveAudio bool, notifyID string, cfg *config.Config) error {
	notifier := notify.New()

	transcriber, err := transcribe.New(cfg.Transcription)
	if err != nil {
		return fmt.Errorf("failed to initialize transcriber: %w", err)
	}

	llmClient, err := llm.New(cfg.LLM)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	audioPath := filepath.Join(sessionPath, "audio.wav")

	if err := postProcessAudio(audioPath); err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	transcription, err := transcriber.Transcribe(ctx, audioPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	transcriptionPath := filepath.Join(sessionPath, "transcripcion.txt")
	if err := os.WriteFile(transcriptionPath, []byte(transcription), 0644); err != nil {
		return fmt.Errorf("failed to save transcription: %w", err)
	}

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
		if err := os.Remove(audioPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove audio file: %w", err)
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
