package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMacroSpec_Active_WhenEmpty_ShouldReturnFalse(t *testing.T) {
	m := MacroSpec{}
	if m.Active() {
		t.Error("expected Active() = false for zero value")
	}
}

func TestMacroSpec_Active_WhenTapeSet_ShouldReturnTrue(t *testing.T) {
	m := MacroSpec{TapeURI: "./tape.txt"}
	if !m.Active() {
		t.Error("expected Active() = true when TapeURI is set")
	}
}

func TestMacroSpec_LoadTape_WhenNoURI_ShouldReturnNil(t *testing.T) {
	m := MacroSpec{}
	commands, err := m.LoadTape()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if commands != nil {
		t.Errorf("expected nil commands, got %v", commands)
	}
}

func TestMacroSpec_LoadTape_WhenValidTape_ShouldParseCommands(t *testing.T) {
	dir := t.TempDir()
	tape := filepath.Join(dir, "tape.txt")
	if err := os.WriteFile(tape, []byte("key p\nwait ready\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := MacroSpec{TapeURI: tape}
	commands, err := m.LoadTape()
	if err != nil {
		t.Fatalf("LoadTape error: %v", err)
	}
	if len(commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(commands))
	}
}

func TestMacroSpec_LoadTape_WhenEmptyTape_ShouldReturnEmptySlice(t *testing.T) {
	dir := t.TempDir()
	tape := filepath.Join(dir, "tape.txt")
	if err := os.WriteFile(tape, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	m := MacroSpec{TapeURI: tape}
	commands, err := m.LoadTape()
	if err != nil {
		t.Fatalf("LoadTape error: %v", err)
	}
	if commands == nil {
		t.Error("expected non-nil (empty) slice, got nil")
	}
	if len(commands) != 0 {
		t.Errorf("expected 0 commands, got %d", len(commands))
	}
}

func TestMacroSpec_LoadTape_WhenInvalidSyntax_ShouldReturnRunError(t *testing.T) {
	dir := t.TempDir()
	tape := filepath.Join(dir, "tape.txt")
	if err := os.WriteFile(tape, []byte("invalid_command xyz\n"), 0644); err != nil {
		t.Fatal(err)
	}

	m := MacroSpec{TapeURI: tape}
	_, err := m.LoadTape()
	if err == nil {
		t.Fatal("expected error for invalid tape syntax")
	}
}

func TestMacroSpec_LoadTape_WhenFileMissing_ShouldReturnError(t *testing.T) {
	m := MacroSpec{TapeURI: "/nonexistent/tape.txt"}
	_, err := m.LoadTape()
	if err == nil {
		t.Fatal("expected error for missing tape file")
	}
}
