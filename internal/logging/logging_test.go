package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetLogger(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { logger = nil })
	logger = nil
}

func TestLogger_WhenInitNotCalled_ShouldReturnDiscardLogger(t *testing.T) {
	resetLogger(t)

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil, want non-nil discard logger")
	}
	if !l.Handler().Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Logger() handler should be enabled for info level")
	}
}

func TestLogger_WhenCalledMultipleTimes_ShouldReturnSameInstance(t *testing.T) {
	resetLogger(t)

	l1 := Logger()
	l2 := Logger()
	if l1 != l2 {
		t.Error("Logger() should return the same instance on subsequent calls")
	}
}

func TestInit_WhenDebugFalse_ShouldCreateDiscardLogger(t *testing.T) {
	resetLogger(t)

	Init(false, "1.0.0", "/tmp", "terraform", "")

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil after Init(debug=false)")
	}
}

func TestInit_WhenDebugTrue_ShouldWriteToLogDir(t *testing.T) {
	resetLogger(t)
	logDir := t.TempDir()

	Init(true, "1.0.0", "/work", "terraform", logDir)

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil after Init(debug=true)")
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("ReadDir(%q) failed: %v", logDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one log file in logDir")
	}

	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "debug-") && strings.HasSuffix(e.Name(), ".log") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no debug-*.log file found in %q", logDir)
	}
}

func TestInit_WhenDebugTrue_ShouldLogSessionStart(t *testing.T) {
	resetLogger(t)
	logDir := t.TempDir()

	Init(true, "2.0.0", "/mydir", "tofu", logDir)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no log file created")
	}

	content, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var record map[string]interface{}
	if err := json.Unmarshal(content, &record); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\ncontent: %s", err, content)
	}

	if record["msg"] != "session.start" {
		t.Errorf("first log message = %q, want %q", record["msg"], "session.start")
	}
	if record["version"] != "2.0.0" {
		t.Errorf("version = %q, want %q", record["version"], "2.0.0")
	}
	if record["dir"] != "/mydir" {
		t.Errorf("dir = %q, want %q", record["dir"], "/mydir")
	}
	if record["binary"] != "tofu" {
		t.Errorf("binary = %q, want %q", record["binary"], "tofu")
	}
}

func TestInit_WhenDebugTrueAndEmptyLogDir_ShouldUseDefaultPath(t *testing.T) {
	resetLogger(t)

	Init(true, "1.0.0", "/work", "terraform", "")

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil")
	}
}

func TestInit_WhenDebugTrueAndHomeDirUnavailable_ShouldFallbackToDiscard(t *testing.T) {
	resetLogger(t)
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")

	Init(true, "1.0.0", "/work", "terraform", "")

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil after UserHomeDir error")
	}
}

func TestInit_WhenOpenFileFails_ShouldFallbackToDiscard(t *testing.T) {
	resetLogger(t)
	dir := t.TempDir()
	logDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	// Make the directory read-only so OpenFile fails but MkdirAll succeeds
	if err := os.Chmod(logDir, 0o500); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}
	t.Cleanup(func() { os.Chmod(logDir, 0o700) })

	Init(true, "1.0.0", "/work", "terraform", logDir)

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil after OpenFile failure")
	}
}

func TestInit_WhenDebugTrueAndInvalidLogDir_ShouldFallbackToDiscard(t *testing.T) {
	resetLogger(t)

	Init(true, "1.0.0", "/work", "terraform", "/proc/nonexistent/impossible/path")

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil after failed Init")
	}
}

func TestInit_WhenLogDirNotWritable_ShouldFallbackToDiscard(t *testing.T) {
	resetLogger(t)
	dir := t.TempDir()
	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0o500); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	nestedDir := filepath.Join(readOnlyDir, "subdir")

	Init(true, "1.0.0", "/work", "terraform", nestedDir)

	l := Logger()
	if l == nil {
		t.Fatal("Logger() returned nil after failed MkdirAll")
	}
}

func TestInitWithWriter_WhenDebugTrue_ShouldSetDebugLevel(t *testing.T) {
	var buf bytes.Buffer

	l := InitWithWriter(&buf, true, "1.0.0", "/dir", "terraform")

	if !l.Handler().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug=true should enable LevelDebug")
	}
}

func TestInitWithWriter_WhenDebugFalse_ShouldUseDefaultLevel(t *testing.T) {
	var buf bytes.Buffer

	l := InitWithWriter(&buf, false, "1.0.0", "/dir", "terraform")

	if l.Handler().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug=false should not enable LevelDebug")
	}
	if !l.Handler().Enabled(context.Background(), slog.LevelInfo) {
		t.Error("debug=false should enable LevelInfo")
	}
}

func TestInitWithWriter_WhenDebugTrue_ShouldLogSessionMetadata(t *testing.T) {
	var buf bytes.Buffer

	InitWithWriter(&buf, true, "3.0.0", "/project", "tofu")

	var record map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\ncontent: %s", err, buf.String())
	}

	if record["msg"] != "session.start" {
		t.Errorf("msg = %q, want %q", record["msg"], "session.start")
	}
	if record["version"] != "3.0.0" {
		t.Errorf("version = %q, want %q", record["version"], "3.0.0")
	}
	if record["dir"] != "/project" {
		t.Errorf("dir = %q, want %q", record["dir"], "/project")
	}
	if record["binary"] != "tofu" {
		t.Errorf("binary = %q, want %q", record["binary"], "tofu")
	}
	if _, ok := record["os"]; !ok {
		t.Error("expected 'os' key in session.start log")
	}
}

func TestInitWithWriter_WhenDebugFalse_ShouldNotLogSessionMetadata(t *testing.T) {
	var buf bytes.Buffer

	InitWithWriter(&buf, false, "1.0.0", "/dir", "terraform")

	if buf.Len() != 0 {
		t.Errorf("debug=false should not write anything, got: %s", buf.String())
	}
}

func TestInitWithWriter_WhenDiscardWriter_ShouldReturnValidLogger(t *testing.T) {
	l := InitWithWriter(io.Discard, true, "1.0.0", "/dir", "terraform")

	if l == nil {
		t.Fatal("InitWithWriter returned nil")
	}
	l.Info("test message")
}

func TestInit_WhenDebugTrueWithValidDir_ShouldCreateFileWithCorrectPermissions(t *testing.T) {
	resetLogger(t)
	logDir := t.TempDir()

	Init(true, "1.0.0", "/work", "terraform", logDir)

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no log file created")
	}

	info, err := os.Stat(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("log file permissions = %o, want 0600", perm)
	}
}

func TestInit_WhenDebugTrueWithValidDir_ShouldCreateDirWithCorrectPermissions(t *testing.T) {
	resetLogger(t)
	baseDir := t.TempDir()
	logDir := filepath.Join(baseDir, "nested", "logs")

	Init(true, "1.0.0", "/work", "terraform", logDir)

	info, err := os.Stat(logDir)
	if err != nil {
		t.Fatalf("Stat(%q) failed: %v", logDir, err)
	}
	perm := info.Mode().Perm()
	if perm != 0o700 {
		t.Errorf("log dir permissions = %o, want 0700", perm)
	}
}

func TestLogger_WhenInitCalledWithDebug_ShouldReturnConfiguredLogger(t *testing.T) {
	resetLogger(t)
	logDir := t.TempDir()

	Init(true, "1.0.0", "/work", "terraform", logDir)

	l := Logger()
	if !l.Handler().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("after Init(debug=true), Logger() should support debug level")
	}
}

func TestLogger_WhenInitCalledWithoutDebug_ShouldReturnDiscardLogger(t *testing.T) {
	resetLogger(t)

	Init(false, "1.0.0", "/work", "terraform", "")

	l := Logger()
	var buf bytes.Buffer
	l.Info("test")
	if buf.Len() != 0 {
		t.Error("after Init(debug=false), Logger() should discard")
	}
}
