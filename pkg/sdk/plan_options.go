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
