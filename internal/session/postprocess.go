package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/llm"
	"github.com/sabhz/trani/pkg/errlog"
	"github.com/sabhz/trani/pkg/notify"
)

// RunPostprocessWorker finalizes a session whose recording has already
// stopped and whose audio was already transcribed progressively, chunk by
// chunk, by the chunker while the session was live. It only needs to clean
// up the accumulated transcript, generate the structured summary, and
// append it to the session note (the user's raw notes and the final note
// are the same file). It runs in a detached process spawned by
// SpawnPostprocess, decoupled from the recording lock.
func RunPostprocessWorker(ctx context.Context, notePath, sourcesTitle, promptTemplate, notifyID string, cfg *config.Config) error {
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

	sessionTitle := strings.TrimSuffix(filepath.Base(notePath), filepath.Ext(notePath))

	if err := writeSummary(ctx, llmClient, notePath, transcription, cfg.Paths.PromptsDir, promptTemplate, sessionTitle, notifier); err != nil {
		// writeSummary already notified and logged the failure; the caller
		// (cmd/postprocess_worker.go) would otherwise fire a second, generic
		// failure notification on top of this specific one.
		return nil
	}

	if !cfg.Audio.Preserve {
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

// writeSummary generates the structured summary from transcription + the
// note's existing content, then appends it under a "## Resumen" heading.
// The existing content (frontmatter, a "## Notas" section, whatever the
// user already put in notePath) is always preserved verbatim; a failure at
// any stage leaves notePath untouched. Shared by the live-session worker
// and the standalone `process` command so both postprocess identically.
func writeSummary(ctx context.Context, llmClient llm.Generator, notePath, transcription, promptsDir, promptTemplate, sessionTitle string, notifier *notify.Notifier) error {
	existingContent, _ := os.ReadFile(notePath)
	notes := strings.TrimSpace(string(existingContent))
	hasNotes := len(notes) > 0

	template, err := loadPromptTemplateStandalone(promptsDir, promptTemplate, hasNotes)
	if err != nil {
		notifier.Error("⚠️ Trani", fmt.Sprintf("Error al cargar plantilla de prompt (%s): %v", sessionTitle, err))
		errlog.Error("prompt_template", sessionTitle, err)
		return err
	}
	prompt := fillPromptTemplate(template, transcription, notes)

	resumen, err := llmClient.Generate(ctx, prompt)

	if err == nil && strings.TrimSpace(resumen) == "" {
		err = fmt.Errorf("the model returned an empty summary")
	}

	if err != nil {
		notifier.Error("⚠️ Trani", fmt.Sprintf("Error al generar resumen (%s): %v", sessionTitle, err))
		errlog.Error("summary", sessionTitle, err)
		return err
	}

	finalContent := appendResumenSection(string(existingContent), resumen)
	if err := os.WriteFile(notePath, []byte(finalContent), 0644); err != nil {
		notifier.Error("⚠️ Trani", fmt.Sprintf("Error al guardar la nota (%s): %v", sessionTitle, err))
		errlog.Error("note_write", sessionTitle, err)
		return fmt.Errorf("failed to update note: %w", err)
	}

	return nil
}

// appendResumenSection preserves existingContent verbatim (frontmatter, a
// "## Notas" section, anything else already there) and adds resumen below
// it under a fixed "## Resumen" heading.
func appendResumenSection(existingContent, resumen string) string {
	existing := strings.TrimRight(existingContent, "\n")
	if existing == "" {
		return "## Resumen\n\n" + resumen
	}
	return existing + "\n\n## Resumen\n\n" + resumen
}
