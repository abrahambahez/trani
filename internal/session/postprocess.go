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

	llmClient, err := llm.New(cfg.LLM)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	transcription, finalAudioPaths, err := transcribeSession(ctx, sessionPath, cfg)
	if err != nil {
		return err
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
		for _, audioPath := range finalAudioPaths {
			if err := os.Remove(audioPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove audio file: %w", err)
			}
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

// transcribeSession processes and transcribes a session's recorded audio,
// handling both single-stream (mic) and dual-stream (mic_system) sessions.
// It returns the transcription text and the list of final audio files (so
// the caller can clean them up unless preserveAudio is set).
func transcribeSession(ctx context.Context, sessionPath string, cfg *config.Config) (string, []string, error) {
	transcriber, err := transcribe.New(cfg.Transcription)
	if err != nil {
		return "", nil, fmt.Errorf("failed to initialize transcriber: %w", err)
	}

	if cfg.Audio.Mode != config.AudioModeMicSystem {
		audioPath := filepath.Join(sessionPath, "audio.wav")
		if err := postProcessAudio(audioPath); err != nil {
			return "", nil, fmt.Errorf("failed to process audio: %w", err)
		}

		transcription, err := transcriber.Transcribe(ctx, audioPath)
		if err != nil {
			return "", nil, fmt.Errorf("transcription failed: %w", err)
		}

		return transcription, []string{audioPath}, nil
	}

	micPath := filepath.Join(sessionPath, "audio-mic.wav")
	systemPath := filepath.Join(sessionPath, "audio-system.wav")

	if err := postProcessAudio(micPath); err != nil {
		return "", nil, fmt.Errorf("failed to process microphone audio: %w", err)
	}
	if err := postProcessAudio(systemPath); err != nil {
		return "", nil, fmt.Errorf("failed to process system audio: %w", err)
	}

	if cfg.Audio.MixStrategy == config.MixStrategySeparateTranscribe {
		micText, err := transcriber.Transcribe(ctx, micPath)
		if err != nil {
			return "", nil, fmt.Errorf("microphone transcription failed: %w", err)
		}

		systemText, err := transcriber.Transcribe(ctx, systemPath)
		if err != nil {
			return "", nil, fmt.Errorf("system audio transcription failed: %w", err)
		}

		transcription := strings.TrimSpace(strings.TrimSpace(micText) + "\n" + strings.TrimSpace(systemText))
		return transcription, []string{micPath, systemPath}, nil
	}

	mixedPath := filepath.Join(sessionPath, "audio.wav")
	if err := mixAudio(micPath, systemPath, mixedPath); err != nil {
		return "", nil, fmt.Errorf("failed to mix audio: %w", err)
	}
	os.Remove(micPath)
	os.Remove(systemPath)

	transcription, err := transcriber.Transcribe(ctx, mixedPath)
	if err != nil {
		return "", nil, fmt.Errorf("transcription failed: %w", err)
	}

	return transcription, []string{mixedPath}, nil
}

// mixAudio sums two already normalized mono streams into one, then
// normalizes the result again to guard against the combined signal
// clipping or drifting too quiet.
func mixAudio(a, b, out string) error {
	tempMixed := out + ".tmp.wav"

	if err := runSox("-m", a, b, tempMixed); err != nil {
		return fmt.Errorf("failed to combine streams: %w", err)
	}

	if err := runSox(tempMixed, out, "norm"); err != nil {
		os.Remove(tempMixed)
		return fmt.Errorf("failed to normalize mixed audio: %w", err)
	}

	os.Remove(tempMixed)
	return nil
}
