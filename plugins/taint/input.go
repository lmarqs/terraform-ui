package taint

// Input is the typed DTO carrying parsed CLI flags into the taint plugin's
// lifecycle. cmd/tfui/taint_command.go binds positional args directly into
// Addrs and hands the value to Plugin.Activate.
type Input struct {
	// Addrs lists the resource addresses to taint. Each address is passed
	// individually to terraform's `taint` command.
	Addrs []string
	// JSON signals the caller wants JSON-shaped stdout. Taint has no stdout
	// content today; the field exists for symmetry with other plugins.
	JSON bool
}
