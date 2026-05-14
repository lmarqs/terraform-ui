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
// lines to the specified logDir (or ~/.tfui/logs if empty). If debug is false,
// all logs are discarded.
func Init(debug bool, version, dir, binary, logDir string) {
	if !debug {
		logger = InitWithWriter(io.Discard, false, version, dir, binary)
		return
	}

	if logDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			logger = InitWithWriter(io.Discard, false, version, dir, binary)
			return
		}
		logDir = filepath.Join(home, ".tfui", "logs")
	}

	if err := os.MkdirAll(logDir, 0o700); err != nil {
		logger = InitWithWriter(io.Discard, false, version, dir, binary)
		return
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("debug-%s.log", time.Now().Format("20060102-150405")))
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		logger = InitWithWriter(io.Discard, false, version, dir, binary)
		return
	}

	logger = InitWithWriter(f, true, version, dir, binary)
}

// InitWithWriter creates a configured logger writing to the given writer.
// If debug is true, the handler is set to LevelDebug and session metadata is
// logged. Returns the configured logger without modifying global state.
func InitWithWriter(w io.Writer, debug bool, version, dir, binary string) *slog.Logger {
	var opts *slog.HandlerOptions
	if debug {
		opts = &slog.HandlerOptions{Level: slog.LevelDebug}
	}

	handler := slog.NewJSONHandler(w, opts)
	l := slog.New(handler)

	if debug {
		l.Info("session.start",
			"version", version,
			"os", runtime.GOOS,
			"dir", dir,
			"binary", binary,
		)
	}

	return l
}

// Logger returns the global logger instance. If Init has not been called, it
// returns a discard logger to avoid nil panics.
func Logger() *slog.Logger {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return logger
}
