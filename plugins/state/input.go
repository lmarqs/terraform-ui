package state

// Input is the typed input port for the state plugin.
//
// Subcommand and Targets carry direct CLI invocations like
// `tfui state rm local_file.one`. When Subcommand is empty the plugin
// boots its interactive browser; otherwise it executes the subcommand
// against Targets and exits.
type Input struct {
	JSON       bool
	Subcommand string
	Targets    []string
}
