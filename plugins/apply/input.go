package apply

import "github.com/lmarqs/terraform-ui/pkg/sdk"

// Input is the typed DTO carrying parsed CLI flags into the apply plugin's
// lifecycle. cmd/tfui/apply_command.go binds cobra flags directly into this
// struct and hands it to Plugin.Activate.
type Input struct {
	// AutoApprove skips the confirm step and starts the apply immediately.
	AutoApprove bool
	// Targets are passed to terraform's `-target=` flag. Used in the standalone
	// CLI path (no plan file). Ignored when a staged plan file is present, per
	// terraform's own constraint (ADR-0019).
	Targets []string
	// JSON signals the caller wants JSON-shaped stdout. Apply has no stdout
	// content today; the field exists for symmetry with other plugins.
	JSON bool
	// Lock overrides the state-locking mode resolved from config for this run,
	// on the preview plan and the apply alike. LockDefault — the zero value —
	// means the caller passed no --lock flag.
	Lock sdk.LockMode
	// LockTimeout overrides the resolved -lock-timeout for this run. Empty means
	// the caller passed no --lock-timeout flag.
	LockTimeout sdk.LockTimeout
}
