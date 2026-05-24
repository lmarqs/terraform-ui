package sdk

// WorkspaceNewOptions holds options for terraform workspace new.
type WorkspaceNewOptions struct {
	Lock        LockMode
	LockTimeout LockTimeout
}

// WorkspaceDeleteOptions holds options for terraform workspace delete.
type WorkspaceDeleteOptions struct {
	Force       bool
	Lock        LockMode
	LockTimeout LockTimeout
}
