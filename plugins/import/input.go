package tfimport

// Input is the typed DTO carrying parsed CLI flags into the import plugin's
// lifecycle. cmd/tfui/import_command.go binds positional args directly into
// Addr / ID and hands the value to Plugin.Activate.
type Input struct {
	// Addr is the resource address to import (terraform's first positional
	// argument).
	Addr string
	// ID is the cloud provider identifier the resource maps to (terraform's
	// second positional argument).
	ID string
	// JSON signals the caller wants JSON-shaped stdout. Import has no stdout
	// content today; the field exists for symmetry with other plugins.
	JSON bool
}
