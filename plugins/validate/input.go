package validate

// Input is the typed DTO carrying parsed CLI flags into the validate plugin's
// lifecycle. cmd/tfui/validate_command.go binds the root-persistent --json
// value into Input.JSON at RunE time and hands the value to Plugin.Activate.
type Input struct {
	// JSON signals the caller wants JSON-shaped stdout. Plugin.Stdout reads
	// this directly from p.input.JSON when rendering the validation result.
	JSON bool
}
