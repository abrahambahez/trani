package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/llm"
	"github.com/sabhz/trani/internal/transcribe"
	"github.com/sabhz/trani/pkg/notify"
)

func ProcessFile(ctx context.Context, audioPath, notesPath, title, promptTemplate string, cfg *config.Config) error {
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", audioPath)
	}

	if title == "" {
		title = strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	}

	timestamp := time.Now().Format("2006-01-02 1504")
	sessionPath := filepath.Join(cfg.Paths.SessionsDir, timestamp)

	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
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

	processedAudioPath := filepath.Join(sessionPath, "audio.wav")
	if err := copyFile(audioPath, processedAudioPath); err != nil {
		return fmt.Errorf("failed to copy audio file: %w", err)
	}

	if err := postProcessAudio(processedAudioPath); err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	transcription, err := transcriber.Transcribe(ctx, processedAudioPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	transcriptionPath := filepath.Join(sessionPath, "transcripcion.txt")
	if err := os.WriteFile(transcriptionPath, []byte(transcription), 0644); err != nil {
		return fmt.Errorf("failed to save transcription: %w", err)
	}

	notes := ""
	if notesPath != "" {
		notesContent, err := os.ReadFile(notesPath)
		if err != nil {
			return fmt.Errorf("failed to read notes file: %w", err)
		}
		notes = strings.TrimSpace(string(notesContent))

		notesDestPath := filepath.Join(sessionPath, "notas.md")
		if err := os.WriteFile(notesDestPath, notesContent, 0644); err != nil {
			return fmt.Errorf("failed to save notes: %w", err)
		}
	}

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

	if err := os.Remove(processedAudioPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove audio file: %w", err)
	}

	notifier.Info("✅ Trani", fmt.Sprintf("Procesamiento completado - %s", sessionPath))
	return nil
}

func loadPromptTemplateStandalone(promptsDir, templateName string, hasNotes bool) string {
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
			return "Error: No se pudo cargar la plantilla de prompt. Verifica la configuración."
		}
	}

	return string(content)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
