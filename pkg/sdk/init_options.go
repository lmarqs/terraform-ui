package sdk

import "io"

// InitOptions holds all options for a terraform init operation.
//
// Every field maps to a flag that terraform-exec accepts; tfui never forwards
// arbitrary flag strings (the typed terraform-exec API has no passthrough).
// Get is a *bool so the zero value means "terraform default" rather than
// "disabled". Note: terraform init has no -lock/-lock-timeout flags (removed
// in Terraform 0.15), so they are intentionally absent here.
type InitOptions struct {
	Upgrade       bool
	Reconfigure   bool
	Backend       BackendMode
	BackendConfig []string
	ForceCopy     bool
	Get           *bool
	FromModule    string
	PluginDir     []string
	Writer        io.Writer // receives streaming output; nil = discard
}
