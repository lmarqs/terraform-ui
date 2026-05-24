package sdk

import "io"

// InitOptions holds all options for a terraform init operation.
type InitOptions struct {
	Upgrade       bool
	Reconfigure   bool
	Backend       BackendMode
	BackendConfig []string
	ExtraArgs     []string
	Writer        io.Writer // receives streaming output; nil = discard
}
