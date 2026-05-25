package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lmarqs/terraform-ui/internal/macro"
	"github.com/lmarqs/terraform-ui/internal/source"
)

// MacroSpec holds macro tape and recording configuration. Active() reports
// whether the session is macro-driven (tape provides input + backend swaps
// to MacroService).
type MacroSpec struct {
	TapeURI   string
	RecordDir string
}

// Active reports whether macro mode is enabled.
func (m MacroSpec) Active() bool {
	return m.TapeURI != ""
}

// LoadTape resolves and parses the macro tape file. Returns nil commands
// (not an error) when no tape is configured.
func (m MacroSpec) LoadTape() ([]macro.Command, error) {
	if m.TapeURI == "" {
		return nil, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	resolver := source.NewResolver(
		&source.LocalProvider{BaseDir: cwd},
		&source.StdinProvider{},
	)
	tapeData, err := resolver.Resolve(context.Background(), m.TapeURI)
	if err != nil {
		return nil, fmt.Errorf("loading macro tape: %w", err)
	}
	commands, err := macro.ParseTape(tapeData)
	if err != nil {
		return nil, &macro.RunError{Code: macro.ExitSyntaxError, Message: err.Error()}
	}
	if commands == nil {
		commands = []macro.Command{}
	}
	return commands, nil
}
