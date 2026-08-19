package errlog

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

var once sync.Once
var logger *slog.Logger

func get() *slog.Logger {
	once.Do(func() {
		path := filepath.Join(os.Getenv("HOME"), ".config", "trani", "logs.jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
			return
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
			return
		}
		logger = slog.New(slog.NewJSONHandler(f, nil))
	})
	return logger
}

// Error appends one JSON line to ~/.config/trani/logs.jsonl. Best-effort:
// if the log file can't be opened, the line is silently discarded — this
// is a debugging aid, not something that should ever break the app.
func Error(event, session string, err error) {
	get().Error(err.Error(), "event", event, "session", session)
}
