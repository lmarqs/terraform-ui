package output

// Input is the typed DTO carrying parsed CLI flags into the output plugin's
// lifecycle. cmd/tfui/output_command.go binds the root-persistent --json
// value into Input.JSON at RunE time and hands the value to Plugin.Activate.
type Input struct {
	// JSON signals the caller wants JSON-shaped stdout. Plugin.Stdout reads
	// this directly from p.input.JSON when rendering the outputs map.
	JSON bool
}
