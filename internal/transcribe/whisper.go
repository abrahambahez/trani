package transcribe

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sabhz/trani/internal/config"
)

// WhisperLocal implements Transcriber using local whisper.cpp.
type WhisperLocal struct {
	modelPath  string
	binaryPath string
	threads    int
	language   string
}

// NewWhisperLocal creates a new WhisperLocal transcriber.
// Returns error if binary or model paths are not configured.
func NewWhisperLocal(cfg config.LocalWhisperConfig) (*WhisperLocal, error) {
	if cfg.BinaryPath == "" {
		return nil, fmt.Errorf("whisper binary path not configured")
	}
	if cfg.ModelPath == "" {
		return nil, fmt.Errorf("whisper model path not configured")
	}

	return &WhisperLocal{
		modelPath:  cfg.ModelPath,
		binaryPath: cfg.BinaryPath,
		threads:    cfg.Threads,
		language:   cfg.Language,
	}, nil
}

// Transcribe converts audio to text using local whisper.cpp.
func (w *WhisperLocal) Transcribe(ctx context.Context, audioPath string) (string, error) {
	if _, err := os.Stat(w.binaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("whisper binary not found at %s", w.binaryPath)
	}

	if _, err := os.Stat(w.modelPath); os.IsNotExist(err) {
		return "", fmt.Errorf("whisper model not found at %s", w.modelPath)
	}

	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return "", fmt.Errorf("audio file not found at %s", audioPath)
	}

	outputDir := filepath.Dir(audioPath)
	outputBase := filepath.Join(outputDir, "transcription")

	args := []string{
		"-m", w.modelPath,
		"-f", audioPath,
		"-l", w.language,
		"-t", strconv.Itoa(w.threads),
		"-otxt",
		"-of", outputBase,
	}

	cmd := exec.CommandContext(ctx, w.binaryPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper transcription failed: %w\nOutput: %s", err, string(output))
	}

	transcriptionPath := outputBase + ".txt"
	content, err := os.ReadFile(transcriptionPath)
	if err != nil {
		return "", fmt.Errorf("failed to read transcription file: %w", err)
	}

	os.Remove(transcriptionPath)

	return strings.TrimSpace(string(content)), nil
}
