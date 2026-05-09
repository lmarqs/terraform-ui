package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var logger *slog.Logger

// Init configures the global logger. If debug is true, logs are written as JSON
// lines to ~/.tfui/debug.log (truncated on each session). If false, all logs are
// discarded.
func Init(debug bool, version, dir, binary string) {
	if !debug {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: discard if we can't determine home
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
		return
	}

	logDir := filepath.Join(home, ".tfui", "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
		return
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("debug-%s.log", time.Now().Format("20060102-150405")))
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
		return
	}

	handler := slog.NewJSONHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger = slog.New(handler)

	// Log session start metadata
	logger.Info("session.start",
		"version", version,
		"os", runtime.GOOS,
		"dir", dir,
		"binary", binary,
	)
}

// Logger returns the global logger instance. If Init has not been called, it
// returns a discard logger to avoid nil panics.
func Logger() *slog.Logger {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return logger
}
