package macro

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type recorderTestModel struct {
	content string
}

func (m recorderTestModel) Init() tea.Cmd                       { return nil }
func (m recorderTestModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m recorderTestModel) View() string                        { return m.content }

func TestRecorder_WhenKeyPressed_ShouldCaptureFrame(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "hello world"}
	rec := NewRecorder(inner, dir, 80, 24)

	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "frame_0001.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("frame content = %q, want %q", string(data), "hello world")
	}
}

func TestRecorder_WhenSessionEnds_ShouldGenerateTape(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "view"}
	rec := NewRecorder(inner, dir, 80, 24)

	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	rec.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "recording.tape"))
	if err != nil {
		t.Fatal(err)
	}
	commands, err := ParseTape(data)
	if err != nil {
		t.Fatalf("generated tape is not parseable: %v", err)
	}
	hasP := false
	hasEnter := false
	for _, cmd := range commands {
		if cmd.Type == CmdKey && len(cmd.Args) > 0 {
			if cmd.Args[0] == "p" {
				hasP = true
			}
			if cmd.Args[0] == "enter" {
				hasEnter = true
			}
		}
	}
	if !hasP || !hasEnter {
		t.Errorf("tape missing expected keys: hasP=%v, hasEnter=%v\ntape:\n%s", hasP, hasEnter, string(data))
	}
}

func TestRecorder_WhenRecordingComplete_ShouldWriteManifest(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "x"}
	rec := NewRecorder(inner, dir, 120, 35)

	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("invalid manifest JSON: %v", err)
	}
	if manifest.Width != 120 || manifest.Height != 35 {
		t.Errorf("manifest dimensions = %dx%d, want 120x35", manifest.Width, manifest.Height)
	}
	if len(manifest.Frames) != 2 {
		t.Errorf("manifest has %d frames, want 2", len(manifest.Frames))
	}
}

func TestRecorder_WhenResized_ShouldRecordDimensions(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "x"}
	rec := NewRecorder(inner, dir, 80, 24)

	rec.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "recording.tape"))
	if err != nil {
		t.Fatal(err)
	}
	commands, err := ParseTape(data)
	if err != nil {
		t.Fatalf("tape parse error: %v", err)
	}
	hasResize := false
	for _, cmd := range commands {
		if cmd.Type == CmdResize {
			hasResize = true
		}
	}
	if !hasResize {
		t.Errorf("tape missing resize command:\n%s", string(data))
	}
}

func TestRecorder_WhenUsedAsTeaModel_ShouldSatisfyInterface(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "test"}
	rec := NewRecorder(inner, dir, 80, 24)

	var _ tea.Model = rec

	cmd := rec.Init()
	if cmd != nil {
		t.Error("Init should return nil for mock model")
	}

	if rec.View() != "test" {
		t.Errorf("View() = %q, want %q", rec.View(), "test")
	}
}

func TestRecorder_WhenUnrecognizedKey_ShouldSkip(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "x"}
	rec := NewRecorder(inner, dir, 80, 24)

	rec.Update(tea.KeyMsg{Type: tea.KeyF1})
	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "recording.tape"))
	if err != nil {
		t.Fatal(err)
	}
	commands, err := ParseTape(data)
	if err != nil {
		t.Fatalf("tape parse error: %v", err)
	}
	for _, cmd := range commands {
		if cmd.Type == CmdKey && len(cmd.Args) > 0 && cmd.Args[0] == "" {
			t.Error("tape should not contain empty key command")
		}
	}
}

func TestRecorder_WhenInnerIsNil_ShouldNotPanic(t *testing.T) {
	dir := t.TempDir()
	rec := NewRecorder(nil, dir, 80, 24)

	if rec.Init() != nil {
		t.Error("Init with nil inner should return nil")
	}
	if rec.View() != "" {
		t.Errorf("View with nil inner should be empty, got %q", rec.View())
	}

	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	rec.CaptureView("external view")

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}
	if len(rec.frames) != 1 {
		t.Errorf("expected 1 frame from CaptureView, got %d", len(rec.frames))
	}
}

func TestRecorder_Inner_ShouldReturnWrappedModel(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "hello"}
	rec := NewRecorder(inner, dir, 80, 24)

	got := rec.Inner()
	if got == nil {
		t.Fatal("Inner() should return the wrapped model")
	}
	model, ok := got.(recorderTestModel)
	if !ok {
		t.Fatal("Inner() should return the same type as the wrapped model")
	}
	if model.content != "hello" {
		t.Errorf("Inner().content = %q, want %q", model.content, "hello")
	}
}

func TestRecorder_Inner_WhenNilInner_ShouldReturnNil(t *testing.T) {
	dir := t.TempDir()
	rec := NewRecorder(nil, dir, 80, 24)

	if rec.Inner() != nil {
		t.Error("Inner() should return nil when inner model is nil")
	}
}

func TestRoundDuration_WhenLessThan100ms_ShouldReturn100ms(t *testing.T) {
	result := roundDuration(50 * time.Millisecond)
	if result != "100ms" {
		t.Errorf("roundDuration(50ms) = %q, want %q", result, "100ms")
	}
}

func TestRoundDuration_WhenExactly100ms_ShouldReturn100ms(t *testing.T) {
	result := roundDuration(100 * time.Millisecond)
	if result != "100ms" {
		t.Errorf("roundDuration(100ms) = %q, want %q", result, "100ms")
	}
}

func TestRoundDuration_WhenRoundNumber_ShouldReturnExact(t *testing.T) {
	result := roundDuration(500 * time.Millisecond)
	if result != "500ms" {
		t.Errorf("roundDuration(500ms) = %q, want %q", result, "500ms")
	}
}

func TestRoundDuration_WhenNotRound_ShouldRoundDown(t *testing.T) {
	result := roundDuration(350 * time.Millisecond)
	if result != "300ms" {
		t.Errorf("roundDuration(350ms) = %q, want %q", result, "300ms")
	}
}

func TestRoundDuration_WhenLargerDuration_ShouldRoundToNearest100ms(t *testing.T) {
	result := roundDuration(1250 * time.Millisecond)
	if result != "1.2s" {
		t.Errorf("roundDuration(1250ms) = %q, want %q", result, "1.2s")
	}
}

func TestRecorder_Finalize_WhenNoTapeCommands_ShouldNotWriteTapeFile(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "x"}
	rec := NewRecorder(inner, dir, 80, 24)

	// Only capture a view without any key presses, so no tape commands are generated
	rec.CaptureView("some view")

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	// Manifest should exist
	if _, err := os.ReadFile(filepath.Join(dir, "manifest.json")); err != nil {
		t.Errorf("manifest.json should exist: %v", err)
	}

	// Tape file should NOT exist since no tape commands were recorded
	if _, err := os.ReadFile(filepath.Join(dir, "recording.tape")); err == nil {
		t.Error("recording.tape should not exist when no tape commands are recorded")
	}
}

func TestRecorder_RecordKey_WhenKeyToStringReturnsEmpty_ShouldNotAddToTape(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "x"}
	rec := NewRecorder(inner, dir, 80, 24)

	// Send a key that KeyToString returns "" for (e.g., F1)
	rec.Update(tea.KeyMsg{Type: tea.KeyF1})

	// Send a valid key to ensure tape is non-empty so Finalize writes it
	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "recording.tape"))
	if err != nil {
		t.Fatal(err)
	}
	commands, err := ParseTape(data)
	if err != nil {
		t.Fatalf("tape parse error: %v", err)
	}
	// Should only have the 'x' key, not the F1 key
	for _, cmd := range commands {
		if cmd.Type == CmdKey && len(cmd.Args) > 0 && cmd.Args[0] == "" {
			t.Error("tape should not contain empty key commands")
		}
	}
}

func TestRecorder_Finalize_WhenManifestWriteFails_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	rec := NewRecorder(nil, dir, 80, 24)

	// Make directory read-only so WriteFile for manifest.json fails
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0755) })

	err := rec.Finalize()
	if err == nil {
		t.Fatal("Finalize should return error when manifest write fails")
	}
}

func TestRecorder_Finalize_WhenDirCannotBeCreated_ShouldReturnError(t *testing.T) {
	// Use a path that cannot be created (a file exists where dir is expected)
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	// Try to use blocker/subdir as the recording dir - MkdirAll should fail
	rec := NewRecorder(nil, filepath.Join(blocker, "subdir"), 80, 24)

	err := rec.Finalize()
	if err == nil {
		t.Fatal("Finalize should return error when dir cannot be created")
	}
}

func TestRecorder_RecordKey_WhenElapsedExceeds200ms_ShouldInsertSleepCommand(t *testing.T) {
	dir := t.TempDir()
	inner := recorderTestModel{content: "x"}
	rec := NewRecorder(inner, dir, 80, 24)

	// First key press to establish tape
	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	// Simulate elapsed time by manipulating lastTime
	rec.lastTime = time.Now().Add(-500 * time.Millisecond)

	// Second key after delay
	rec.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	if err := rec.Finalize(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "recording.tape"))
	if err != nil {
		t.Fatal(err)
	}
	commands, err := ParseTape(data)
	if err != nil {
		t.Fatalf("tape parse error: %v", err)
	}
	hasSleep := false
	for _, cmd := range commands {
		if cmd.Type == CmdSleep {
			hasSleep = true
		}
	}
	if !hasSleep {
		t.Errorf("tape should contain a sleep command after >200ms delay:\n%s", string(data))
	}
}

func TestRunner_WhenRecorderAttached_ShouldCaptureFrames(t *testing.T) {
	dir := t.TempDir()
	inner := mockModel{content: "start"}
	driver := NewDriver(inner, 80, 24)
	rec := NewRecorder(nil, dir, 80, 24)
	runner := NewRunner(driver)
	runner.WithRecorder(rec)

	commands := []Command{
		{Type: CmdKey, Args: []string{"p"}},
		{Type: CmdKey, Args: []string{"s"}},
	}

	if err := runner.Execute(commands); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	// Initial frame + 2 command frames = 3
	if len(manifest.Frames) != 3 {
		t.Errorf("expected 3 frames, got %d", len(manifest.Frames))
	}
}
