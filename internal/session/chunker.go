package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sabhz/trani/internal/audio"
	"github.com/sabhz/trani/internal/config"
	"github.com/sabhz/trani/internal/transcribe"
)

// chunkPollInterval is how often the chunker checks for newly closed
// chunks. It only needs to be shorter than audio.chunk_seconds.
const chunkPollInterval = 20 * time.Second

// chunker watches a recorder's segmented chunk files as they close and
// transcribes them progressively, appending results to the session's
// .sources/<title>.txt and .wav files as it goes. This lets most of the
// transcription work happen while the recording is still in progress
// instead of all at once when the session stops.
type chunker struct {
	cfg         *config.Config
	recorder    *audio.Recorder
	transcriber transcribe.Transcriber

	txtPath string
	wavPath string

	processed int
}

func newChunker(cfg *config.Config, sourcesTitle string, recorder *audio.Recorder, transcriber transcribe.Transcriber) (*chunker, error) {
	sourcesDir := filepath.Join(cfg.Paths.SessionsDir, ".sources")
	if err := os.MkdirAll(sourcesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sources directory: %w", err)
	}

	return &chunker{
		cfg:         cfg,
		recorder:    recorder,
		transcriber: transcriber,
		txtPath:     filepath.Join(sourcesDir, sourcesTitle+".txt"),
		wavPath:     filepath.Join(sourcesDir, sourcesTitle+".wav"),
	}, nil
}

// run polls for newly closed chunks until stop is closed. Per-chunk errors
// are logged, not returned, so a transient transcription failure doesn't
// take down an in-progress recording.
func (c *chunker) run(ctx context.Context, stop <-chan struct{}) {
	ticker := time.NewTicker(chunkPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.pollOnce(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "trani: chunk processing error: %v\n", err)
			}
		case <-stop:
			return
		}
	}
}

// pollOnce processes any chunks that have closed (appeared in the segment
// list) since the last call. Safe to call once more after the recorder has
// stopped, to pick up the final partial chunk.
func (c *chunker) pollOnce(ctx context.Context) error {
	if !c.recorder.HasSystemAudio() {
		return c.pollMicOnly(ctx)
	}
	return c.pollMicSystem(ctx)
}

func (c *chunker) pollMicOnly(ctx context.Context) error {
	segments, err := readSegmentList(c.recorder.MicSegmentList())
	if err != nil {
		return err
	}

	for c.processed < len(segments) {
		chunkPath := segments[c.processed]
		if err := c.processMicOnlyChunk(ctx, chunkPath); err != nil {
			return fmt.Errorf("chunk %s: %w", filepath.Base(chunkPath), err)
		}
		c.processed++
	}

	return nil
}

func (c *chunker) pollMicSystem(ctx context.Context) error {
	micSegments, err := readSegmentList(c.recorder.MicSegmentList())
	if err != nil {
		return err
	}

	systemSegments, err := readSegmentList(c.recorder.SystemSegmentList())
	if err != nil {
		return err
	}

	ready := len(micSegments)
	if len(systemSegments) < ready {
		ready = len(systemSegments)
	}

	for c.processed < ready {
		micPath := micSegments[c.processed]
		systemPath := systemSegments[c.processed]
		if err := c.processMicSystemChunk(ctx, micPath, systemPath); err != nil {
			return fmt.Errorf("chunk %s: %w", filepath.Base(micPath), err)
		}
		c.processed++
	}

	return nil
}

func (c *chunker) processMicOnlyChunk(ctx context.Context, chunkPath string) error {
	if err := postProcessAudio(chunkPath); err != nil {
		return fmt.Errorf("failed to process audio: %w", err)
	}

	text, err := c.transcriber.Transcribe(ctx, chunkPath)
	if err != nil {
		return fmt.Errorf("transcription failed: %w", err)
	}

	if err := c.appendText(text); err != nil {
		return err
	}
	if err := c.appendAudio(chunkPath); err != nil {
		return err
	}

	return os.Remove(chunkPath)
}

func (c *chunker) processMicSystemChunk(ctx context.Context, micPath, systemPath string) error {
	if err := postProcessAudio(micPath); err != nil {
		return fmt.Errorf("failed to process microphone audio: %w", err)
	}
	if err := postProcessAudio(systemPath); err != nil {
		return fmt.Errorf("failed to process system audio: %w", err)
	}

	// Always mix a combined chunk for the archived .sources/*.wav,
	// regardless of mix_strategy, so the archive stays one coherent track.
	combinedPath := systemPath + ".combined.wav"
	if err := mixAudio(micPath, systemPath, combinedPath); err != nil {
		return fmt.Errorf("failed to combine audio: %w", err)
	}
	defer os.Remove(combinedPath)

	var text string
	if c.cfg.Audio.MixStrategy == config.MixStrategySeparateTranscribe {
		micText, err := c.transcriber.Transcribe(ctx, micPath)
		if err != nil {
			return fmt.Errorf("microphone transcription failed: %w", err)
		}
		systemText, err := c.transcriber.Transcribe(ctx, systemPath)
		if err != nil {
			return fmt.Errorf("system audio transcription failed: %w", err)
		}
		text = strings.TrimSpace(strings.TrimSpace(micText) + "\n" + strings.TrimSpace(systemText))
	} else {
		transcription, err := c.transcriber.Transcribe(ctx, combinedPath)
		if err != nil {
			return fmt.Errorf("transcription failed: %w", err)
		}
		text = transcription
	}

	if err := c.appendText(text); err != nil {
		return err
	}
	if err := c.appendAudio(combinedPath); err != nil {
		return err
	}

	os.Remove(micPath)
	os.Remove(systemPath)
	return nil
}

func (c *chunker) appendText(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	f, err := os.OpenFile(c.txtPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open transcript file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(text + "\n"); err != nil {
		return fmt.Errorf("failed to append transcript: %w", err)
	}

	return nil
}

func (c *chunker) appendAudio(chunkPath string) error {
	if _, err := os.Stat(c.wavPath); os.IsNotExist(err) {
		return copyFile(chunkPath, c.wavPath)
	}

	tempOut := c.wavPath + ".tmp.wav"
	if err := runSox(c.wavPath, chunkPath, tempOut); err != nil {
		return fmt.Errorf("failed to append audio chunk: %w", err)
	}

	return os.Rename(tempOut, c.wavPath)
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

// readSegmentList returns the chunk paths ffmpeg has appended to the given
// segment list file. ffmpeg writes entries relative to its own working
// directory (typically just the basename), not relative to the list file's
// location, so relative entries are resolved against the list file's
// directory, which is always where the chunks actually live.
func readSegmentList(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read segment list: %w", err)
	}

	dir := filepath.Dir(path)

	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(dir, line)
		}
		out = append(out, line)
	}

	return out, nil
}
