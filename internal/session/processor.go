package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/llm"
	"github.com/sabhz/trani/internal/transcribe"
	"github.com/sabhz/trani/pkg/notify"
)

func ProcessFile(ctx context.Context, audioPath, notesPath, promptTemplate string, cfg *config.Config) error {
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", audioPath)
	}

	sourcesTitle := time.Now().Format("2006-01-02 1504")
	notePath := filepath.Join(cfg.Paths.SessionsDir, sourcesTitle+".md")
	sourcesDir := filepath.Join(cfg.Paths.SessionsDir, ".sources")

	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return fmt.Errorf("failed to create sessions directory: %w", err)
	}
	if err := os.MkdirAll(cfg.Paths.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	transcriber, err := transcribe.New(cfg.Transcription)
	if err != nil {
		return fmt.Errorf("failed to initialize transcriber: %w", err)
	}

	llmClient, err := llm.New(cfg.LLM)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	if err := ensureDefaultPrompts(cfg.Paths.PromptsDir); err != nil {
		return fmt.Errorf("failed to initialize prompts: %w", err)
	}

	notifier := notify.New()
	notifier.Info("🎙️ Trani", "Procesando audio...")

	processedAudioPath := filepath.Join(cfg.Paths.TempDir, sourcesTitle+".wav")
	if err := copyFile(audioPath, processedAudioPath); err != nil {
		return fmt.Errorf("failed to copy audio file: %w", err)
	}
	defer os.Remove(processedAudioPath)

	if err := postProcessAudio(processedAudioPath); err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	var prompt string
	if notesPath != "" {
		notesContent, err := os.ReadFile(notesPath)
		if err != nil {
			return fmt.Errorf("failed to read notes file: %w", err)
		}
		if err := os.WriteFile(notePath, notesContent, 0644); err != nil {
			return fmt.Errorf("failed to seed note: %w", err)
		}
		prompt = buildTranscriptionPrompt(string(notesContent))
	}

	transcription, err := transcriber.Transcribe(ctx, processedAudioPath, prompt)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	transcriptionPath := filepath.Join(sourcesDir, sourcesTitle+".txt")
	if err := os.WriteFile(transcriptionPath, []byte(transcription), 0644); err != nil {
		return fmt.Errorf("failed to save transcription: %w", err)
	}

	if err := writeSummary(ctx, llmClient, notePath, transcription, cfg.Paths.PromptsDir, promptTemplate, sourcesTitle, notifier); err != nil {
		return err
	}

	notifier.Info("✅ Trani", fmt.Sprintf("Procesamiento completado - %s", sourcesTitle))
	return nil
}

func loadPromptTemplateStandalone(promptsDir, templateName string, hasNotes bool) (string, error) {
	suffix := ".txt"
	if !hasNotes {
		suffix = "_no_notes.txt"
	}

	filename := templateName + suffix
	promptPath := filepath.Join(promptsDir, filename)

	content, err := os.ReadFile(promptPath)
	if err != nil {
		defaultFilename := "default" + suffix
		defaultPath := filepath.Join(promptsDir, defaultFilename)
		content, err = os.ReadFile(defaultPath)
		if err != nil {
			return "", fmt.Errorf("neither %q nor %q exist in %s", filename, defaultFilename, promptsDir)
		}
	}

	return string(content), nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
