package untaint

// Input is the typed DTO carrying parsed CLI flags into the untaint plugin's
// lifecycle. cmd/tfui/untaint_command.go binds positional args directly into
// Addrs and hands the value to Plugin.Activate.
type Input struct {
	// Addrs lists the resource addresses to untaint. Each address is passed
	// individually to terraform's `untaint` command.
	Addrs []string
	// JSON signals the caller wants JSON-shaped stdout. Untaint has no stdout
	// content today; the field exists for symmetry with other plugins.
	JSON bool
}
