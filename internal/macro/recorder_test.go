package macro

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type recorderTestModel struct {
	content string
}

func (m recorderTestModel) Init() tea.Cmd                       { return nil }
func (m recorderTestModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m recorderTestModel) View() string                        { return m.content }

func TestRecorder_captures_frames_on_key(t *testing.T) {
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

func TestRecorder_generates_tape(t *testing.T) {
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

func TestRecorder_writes_manifest(t *testing.T) {
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

func TestRecorder_records_resize(t *testing.T) {
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

func TestRecorder_implements_tea_Model(t *testing.T) {
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

func TestRecorder_skips_unrecognized_keys(t *testing.T) {
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

func TestRecorder_nil_inner_does_not_panic(t *testing.T) {
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

func TestRunner_with_recorder_captures_frames(t *testing.T) {
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
