package session

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sabhz/trani/internal/audio"
	"github.com/sabhz/trani/internal/config"
)

// stubTranscriber returns canned text and records every path it was asked
// to transcribe, so tests can assert on call order without hitting a real
// transcription API.
type stubTranscriber struct {
	calls []string
	texts []string // one per call, cycling through if exhausted
}

func (s *stubTranscriber) Transcribe(ctx context.Context, audioPath string) (string, error) {
	s.calls = append(s.calls, audioPath)
	if len(s.texts) == 0 {
		return "", nil
	}
	return s.texts[(len(s.calls)-1)%len(s.texts)], nil
}

// writeTestChunk generates a short, real WAV file via sox so postProcessAudio
// (which shells out to sox) has valid input to work with.
func writeTestChunk(t *testing.T, path string) {
	t.Helper()
	if _, err := exec.LookPath("sox"); err != nil {
		t.Skip("sox not available")
	}
	cmd := exec.Command("sox", "-n", path, "synth", "0.3", "sine", "440")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to generate test chunk: %v", err)
	}
}

func appendSegmentListLine(t *testing.T, listPath, entry string) {
	t.Helper()
	f, err := os.OpenFile(listPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open segment list: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(entry + "\n"); err != nil {
		t.Fatalf("failed to write segment list entry: %v", err)
	}
}

func TestChunkerProcessesNewSegmentsProgressively(t *testing.T) {
	tempDir := t.TempDir()
	sessionsDir := t.TempDir()

	cfg := &config.Config{
		Paths: config.PathsConfig{SessionsDir: sessionsDir, TempDir: tempDir},
		Audio: config.AudioConfig{Mode: config.AudioModeMic},
	}

	recorder := audio.New(cfg.Audio, tempDir)
	transcriber := &stubTranscriber{texts: []string{"hola", "mundo"}}

	c, err := newChunker(cfg, "20260101-1200", recorder, transcriber)
	if err != nil {
		t.Fatalf("newChunker failed: %v", err)
	}

	// First chunk arrives.
	chunk0 := filepath.Join(tempDir, "chunk-mic-000.wav")
	writeTestChunk(t, chunk0)
	appendSegmentListLine(t, recorder.MicSegmentList(), chunk0)

	if err := c.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce (1st chunk) failed: %v", err)
	}

	if len(transcriber.calls) != 1 {
		t.Fatalf("expected 1 transcribe call after first poll, got %d", len(transcriber.calls))
	}
	if _, err := os.Stat(chunk0); !os.IsNotExist(err) {
		t.Error("processed chunk file should have been removed")
	}

	text, err := os.ReadFile(c.txtPath)
	if err != nil {
		t.Fatalf("failed to read transcript: %v", err)
	}
	if string(text) != "hola\n" {
		t.Errorf("transcript after 1st chunk: expected %q, got %q", "hola\n", string(text))
	}

	// A second poll with no new segments should not call the transcriber again.
	if err := c.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce (no new chunks) failed: %v", err)
	}
	if len(transcriber.calls) != 1 {
		t.Fatalf("expected still 1 transcribe call with no new chunks, got %d", len(transcriber.calls))
	}

	// Second chunk arrives later (simulating the next 5-minute boundary).
	chunk1 := filepath.Join(tempDir, "chunk-mic-001.wav")
	writeTestChunk(t, chunk1)
	appendSegmentListLine(t, recorder.MicSegmentList(), chunk1)

	if err := c.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce (2nd chunk) failed: %v", err)
	}

	if len(transcriber.calls) != 2 {
		t.Fatalf("expected 2 transcribe calls after second poll, got %d", len(transcriber.calls))
	}

	text, err = os.ReadFile(c.txtPath)
	if err != nil {
		t.Fatalf("failed to read transcript: %v", err)
	}
	if string(text) != "hola\nmundo\n" {
		t.Errorf("transcript after 2nd chunk: expected %q, got %q", "hola\nmundo\n", string(text))
	}

	if _, err := os.Stat(c.wavPath); err != nil {
		t.Errorf("expected accumulated audio file to exist: %v", err)
	}
}

func TestChunkerMicSystemWaitsForBothStreams(t *testing.T) {
	tempDir := t.TempDir()
	sessionsDir := t.TempDir()

	cfg := &config.Config{
		Paths: config.PathsConfig{SessionsDir: sessionsDir, TempDir: tempDir},
		Audio: config.AudioConfig{Mode: config.AudioModeMicSystem, MixStrategy: config.MixStrategyPostMix},
	}

	recorder := audio.New(cfg.Audio, tempDir)
	transcriber := &stubTranscriber{texts: []string{"mezclado"}}

	c, err := newChunker(cfg, "20260101-1200", recorder, transcriber)
	if err != nil {
		t.Fatalf("newChunker failed: %v", err)
	}

	micChunk := filepath.Join(tempDir, "chunk-mic-000.wav")
	writeTestChunk(t, micChunk)
	appendSegmentListLine(t, recorder.MicSegmentList(), micChunk)

	// Only the mic side has closed its chunk; the pair isn't ready yet.
	if err := c.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce (mic only) failed: %v", err)
	}
	if len(transcriber.calls) != 0 {
		t.Fatalf("expected no transcribe calls before the system chunk arrives, got %d", len(transcriber.calls))
	}

	systemChunk := filepath.Join(tempDir, "chunk-system-000.wav")
	writeTestChunk(t, systemChunk)
	appendSegmentListLine(t, recorder.SystemSegmentList(), systemChunk)

	if err := c.pollOnce(context.Background()); err != nil {
		t.Fatalf("pollOnce (pair ready) failed: %v", err)
	}
	if len(transcriber.calls) != 1 {
		t.Fatalf("expected 1 transcribe call once the pair is ready, got %d", len(transcriber.calls))
	}

	if _, err := os.Stat(micChunk); !os.IsNotExist(err) {
		t.Error("mic chunk should have been removed after mixing")
	}
	if _, err := os.Stat(systemChunk); !os.IsNotExist(err) {
		t.Error("system chunk should have been removed after mixing")
	}
}

func TestReadSegmentListMissingFile(t *testing.T) {
	segments, err := readSegmentList(filepath.Join(t.TempDir(), "does-not-exist.txt"))
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(segments) != 0 {
		t.Errorf("expected no segments, got %d", len(segments))
	}
}

func TestReadSegmentListResolvesRelativeEntriesAgainstListDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "segments.txt")
	// ffmpeg writes entries relative to its own working directory (in
	// practice, just the basename), not as full paths.
	content := "a.wav\n\nb.wav\n  \nc.wav\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write segment list: %v", err)
	}

	segments, err := readSegmentList(path)
	if err != nil {
		t.Fatalf("readSegmentList failed: %v", err)
	}

	expected := []string{
		filepath.Join(dir, "a.wav"),
		filepath.Join(dir, "b.wav"),
		filepath.Join(dir, "c.wav"),
	}
	if len(segments) != len(expected) {
		t.Fatalf("expected %d segments, got %d (%v)", len(expected), len(segments), segments)
	}
	for i, e := range expected {
		if segments[i] != e {
			t.Errorf("segment %d: expected %s, got %s", i, e, segments[i])
		}
	}
}

func TestReadSegmentListKeepsAbsoluteEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "segments.txt")
	absoluteEntry := filepath.Join(dir, "already-absolute.wav")
	if err := os.WriteFile(path, []byte(absoluteEntry+"\n"), 0644); err != nil {
		t.Fatalf("failed to write segment list: %v", err)
	}

	segments, err := readSegmentList(path)
	if err != nil {
		t.Fatalf("readSegmentList failed: %v", err)
	}

	if len(segments) != 1 || segments[0] != absoluteEntry {
		t.Errorf("expected [%s], got %v", absoluteEntry, segments)
	}
}
