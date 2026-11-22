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
	Dir    string   // working directory, serialized as -chdir=<Dir> (omitted if empty)
}

// String serializes the command to a string a POSIX shell runs as written.
// Each token that needs it is single-quoted, plain tokens stay bare, and a
// set Dir becomes -chdir=<Dir> right after the binary. Verb is emitted as is:
// it is a fixed terraform subcommand, possibly two words ("state rm").
func (c Command) String() string {
	parts := []string{shellQuote(c.Binary)}
	if c.Dir != "" {
		parts = append(parts, shellQuote("-chdir="+c.Dir))
	}
	parts = append(parts, c.Verb)
	for _, tok := range c.Flags {
		parts = append(parts, shellQuote(tok))
	}
	for _, tok := range c.Args {
		parts = append(parts, shellQuote(tok))
	}
	return strings.Join(parts, " ")
}

// shellQuote leaves s bare when every byte is shell-safe and otherwise wraps
// it in single quotes, escaping embedded single quotes.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsFunc(s, func(r rune) bool { return !isShellSafe(r) }) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func isShellSafe(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
		strings.ContainsRune("_@%+=:,./-", r)
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
