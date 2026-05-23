package sdk

// Context is the immutable snapshot of the complete terraform operating
// environment: project + chdir + workspace + resolved execution parameters
// (var-files, vars, parallelism, lock, lock-timeout, extra-args, pins,
// scoped Service). The app owns one Context at a time and replaces
// it atomically on chdir, workspace, or pin changes — never patches it.
//
// Plugins read Context fresh at the top of every terraform-affecting
// operation. Captured Context values are safe to share across goroutines
// because they are never mutated after construction.
//
// See ADR-0018 (atomic context replacement) and ADR-0019 (unidirectional data
// flow) for rationale.
type Context struct {
	// Chdir is the relative member path (e.g., "modules/vpc").
	Chdir string
	// WorkingDir is the absolute path of the active chdir.
	WorkingDir string
	// Workspace is the active terraform workspace.
	Workspace string
	// Service is the terraform service scoped to WorkingDir.
	Service Service

	// Pins are the user-selected resource addresses scoped to this Context.
	// They become terraform -target flags when PlanOptions is built.
	// Pins die on context switch — they are part of the snapshot, not a sidecar.
	Pins []string

	// VarFiles are the resolved -var-file paths for this Context.
	VarFiles []string
	// Vars are the resolved -var key=value entries for this Context.
	Vars map[string]string
	// ExtraArgs is the CLI passthrough (-- args) carried into terraform.
	ExtraArgs []string

	// Parallelism is the resolved -parallelism value (0 = terraform default).
	Parallelism int
	// Lock is the resolved -lock value (nil = terraform default).
	Lock *bool
	// LockTimeout is the resolved -lock-timeout duration string.
	LockTimeout string
}

// PlanOptions builds a PlanOptions from the Context's resolved fields. The
// caller may override individual fields (Writer, PlanFile, Replace, Destroy,
// RefreshOnly, Refresh) on the returned struct before passing to Service.
func (c *Context) PlanOptions() PlanOptions {
	if c == nil {
		return PlanOptions{}
	}
	return PlanOptions{
		Targets:     c.Pins,
		VarFiles:    c.VarFiles,
		Vars:        c.Vars,
		ExtraArgs:   c.ExtraArgs,
		Parallelism: c.Parallelism,
		Lock:        c.Lock,
		LockTimeout: c.LockTimeout,
	}
}

// ApplyOptions builds an ApplyOptions from the Context's resolved fields.
// Apply receives the plan file separately from the plan plugin — it is not
// a Context concern (ADR-0019). Targets do not appear here: apply runs
// `terraform apply <planfile>` and any targeting was already baked into
// the plan that produced the file.
func (c *Context) ApplyOptions() ApplyOptions {
	if c == nil {
		return ApplyOptions{}
	}
	return ApplyOptions{
		VarFiles:    c.VarFiles,
		Vars:        c.Vars,
		ExtraArgs:   c.ExtraArgs,
		Parallelism: c.Parallelism,
		Lock:        c.Lock,
		LockTimeout: c.LockTimeout,
	}
}

// WithPins returns a fresh Context snapshot with the supplied pins,
// leaving every other field — and the receiver — untouched. Returns a new
// pointer; the receiver is never mutated.
func (c *Context) WithPins(pins []string) *Context {
	next := *c
	next.Pins = append([]string(nil), pins...)
	return &next
}

// TogglePin returns a fresh Context with the address added to Pins if absent,
// or removed if present. The receiver is never mutated.
func (c *Context) TogglePin(address string) *Context {
	for i, pin := range c.Pins {
		if pin == address {
			pins := make([]string, 0, len(c.Pins)-1)
			pins = append(pins, c.Pins[:i]...)
			pins = append(pins, c.Pins[i+1:]...)
			return c.WithPins(pins)
		}
	}
	pins := make([]string, 0, len(c.Pins)+1)
	pins = append(pins, c.Pins...)
	pins = append(pins, address)
	return c.WithPins(pins)
}

// PinnedAddresses returns the Pins from the Context returned by getCtx.
func PinnedAddresses(getCtx func() *Context) []string {
	return getCtx().Pins
}
