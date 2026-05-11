package macro

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CommandType identifies a tape command.
type CommandType int

const (
	CmdKey CommandType = iota
	CmdWaitReady
	CmdWaitView
	CmdAssertView
	CmdScreenshot
	CmdResize
	CmdSleep
)

// Command represents a single instruction in a tape file.
type Command struct {
	Type CommandType
	Args []string
	Line int
}

// ParseTape parses tape DSL from bytes.
// Format: one command per line, or semicolons separate inline commands.
// Empty lines and lines starting with # are ignored.
func ParseTape(data []byte) ([]Command, error) {
	input := string(data)

	// If input contains semicolons but no newlines, split on semicolons (inline mode)
	if !strings.Contains(input, "\n") && strings.Contains(input, ";") {
		input = strings.ReplaceAll(input, ";", "\n")
	}

	lines := strings.Split(input, "\n")
	commands := make([]Command, 0, len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		cmd, err := parseLine(line, i+1)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

func parseLine(line string, lineNum int) (Command, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return Command{}, fmt.Errorf("empty command")
	}

	verb := parts[0]
	args := parts[1:]

	switch verb {
	case "key":
		if len(args) != 1 {
			return Command{}, fmt.Errorf("key requires exactly 1 argument, got %d", len(args))
		}
		return Command{Type: CmdKey, Args: args, Line: lineNum}, nil

	case "wait":
		if len(args) == 0 {
			return Command{}, fmt.Errorf("wait requires an argument (ready, or view <substring>)")
		}
		switch args[0] {
		case "ready":
			if len(args) != 1 {
				return Command{}, fmt.Errorf("wait ready takes no additional arguments")
			}
			return Command{Type: CmdWaitReady, Line: lineNum}, nil
		case "view":
			if len(args) < 2 {
				return Command{}, fmt.Errorf("wait view requires a substring argument")
			}
			substr := strings.Join(args[1:], " ")
			return Command{Type: CmdWaitView, Args: []string{substr}, Line: lineNum}, nil
		default:
			return Command{}, fmt.Errorf("unknown wait target %q (use: ready, view)", args[0])
		}

	case "assert":
		if len(args) < 2 {
			return Command{}, fmt.Errorf("assert requires at least 2 arguments (e.g., assert view <substring>)")
		}
		switch args[0] {
		case "view":
			substr := strings.Join(args[1:], " ")
			return Command{Type: CmdAssertView, Args: []string{substr}, Line: lineNum}, nil
		default:
			return Command{}, fmt.Errorf("unknown assert target %q (use: view)", args[0])
		}

	case "screenshot":
		if len(args) != 1 {
			return Command{}, fmt.Errorf("screenshot requires exactly 1 argument (file path)")
		}
		return Command{Type: CmdScreenshot, Args: args, Line: lineNum}, nil

	case "resize":
		if len(args) != 2 {
			return Command{}, fmt.Errorf("resize requires exactly 2 arguments (width height)")
		}
		if _, err := strconv.Atoi(args[0]); err != nil {
			return Command{}, fmt.Errorf("resize width must be integer: %w", err)
		}
		if _, err := strconv.Atoi(args[1]); err != nil {
			return Command{}, fmt.Errorf("resize height must be integer: %w", err)
		}
		return Command{Type: CmdResize, Args: args, Line: lineNum}, nil

	case "sleep":
		if len(args) != 1 {
			return Command{}, fmt.Errorf("sleep requires exactly 1 argument (duration, e.g., 100ms, 2s)")
		}
		if _, err := time.ParseDuration(args[0]); err != nil {
			return Command{}, fmt.Errorf("sleep duration invalid: %w", err)
		}
		return Command{Type: CmdSleep, Args: args, Line: lineNum}, nil

	default:
		return Command{}, fmt.Errorf("unknown command %q (available: key, wait, assert, screenshot, resize, sleep)", verb)
	}
}
