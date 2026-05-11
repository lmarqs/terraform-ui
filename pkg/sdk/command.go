package sdk

import "strings"

// Command represents a terraform CLI invocation that can be serialized
// to a runnable shell command. Every service operation maps to exactly
// one Command.
type Command struct {
	Binary string   // "terraform" or "tofu"
	Verb   string   // "plan", "apply", "state rm", "taint", etc.
	Args   []string // positional arguments (addresses, IDs)
	Flags  []string // flags like "-target=X"
	Dir    string   // working directory (omitted from String() if empty)
}

// String serializes the command to a runnable shell string.
func (c Command) String() string {
	parts := []string{c.Binary, c.Verb}
	parts = append(parts, c.Flags...)
	parts = append(parts, c.Args...)
	return strings.Join(parts, " ")
}

// WithDir returns a copy of the command with the directory set.
func (c Command) WithDir(dir string) Command {
	c.Dir = dir
	return c
}

// CommandErr wraps a Command as an error, allowing services to return
// the equivalent CLI command instead of executing it.
type CommandErr struct {
	Cmd Command
}

func (e *CommandErr) Error() string {
	return e.Cmd.String()
}

// IsCommandErr extracts the Command from an error if it wraps a CommandErr.
func IsCommandErr(err error) (Command, bool) {
	if err == nil {
		return Command{}, false
	}
	if ce, ok := err.(*CommandErr); ok {
		return ce.Cmd, true
	}
	return Command{}, false
}
