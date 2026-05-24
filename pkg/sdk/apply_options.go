package sdk

import "io"

// ApplyOptions holds all options for a terraform apply operation.
// All fields are passed through to terraform as-is.
type ApplyOptions struct {
	PlanFile    string
	Targets     []string
	VarFiles    []string
	Vars        map[string]string
	Parallelism int
	Lock        LockMode
	LockTimeout LockTimeout
	AutoApprove bool
	ExtraArgs   []string
	Writer      io.Writer
}
