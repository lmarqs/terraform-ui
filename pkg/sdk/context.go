package sdk

// Context provides shared state passed to plugins during initialization.
// It contains the working directory, active workspace, and the terraform
// service instance that plugins use to execute operations.
type Context struct {
	// Dir is the working directory for terraform operations.
	Dir string
	// Workspace is the name of the currently active terraform workspace.
	Workspace string
	// Service is the terraform service used to run plan, apply, and state operations.
	Service Service
}
