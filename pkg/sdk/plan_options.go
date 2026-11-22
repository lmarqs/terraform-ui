package sdk

import "io"

// PlanOptions holds all options for a terraform plan operation.
type PlanOptions struct {
	Targets     []string
	VarFiles    []string
	Vars        map[string]string
	Replace     []string
	Destroy     bool
	Refresh     RefreshMode
	Parallelism int
	Lock        LockMode
	LockTimeout LockTimeout
	ExtraArgs   []string
	PlanFile    string    // path to write the plan artifact (-out=<path>); empty = service default
	Writer      io.Writer // receives streaming output; nil = discard
	// Reread asks for a plan seeded from a file to be read from that file again
	// rather than served as it was when seeded. Set by an explicit refresh.
	Reread bool
}

// WithDefaultTargets returns o with defaults as its targets when o selects
// none. The defaults are the project's `plugin "plan" { targets }` list: an
// explicit selection replaces them, because a default that cannot be narrowed
// is the wrong shape for targeting. Every plan tfui runs for the user applies
// this one rule, so each view plans the same resources.
func (o PlanOptions) WithDefaultTargets(defaults []string) PlanOptions {
	if len(o.Targets) == 0 {
		o.Targets = defaults
	}
	return o
}
