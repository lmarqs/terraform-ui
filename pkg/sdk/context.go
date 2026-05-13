package sdk

import "log/slog"

// Context provides shared state passed to plugins during initialization.
// It contains the working directory, active workspace, the terraform
// service instance, and a structured logger for debug output.
type Context struct {
	// WorkingDir is the working directory for terraform operations.
	WorkingDir string
	// Workspace is the name of the currently active terraform workspace.
	Workspace string
	// Service is the terraform service used to run plan, apply, and state operations.
	Service Service
	// Logger is the structured logger for debug output. Plugins should use this
	// instead of the global slog to enable testability (inject a discard or buffer logger).
	Logger *slog.Logger
	// Pins is the shared pin service for resource targeting across plugins.
	Pins *PinService
	// Options holds resolved CLI/config options (var-files, vars, extra-args).
	Options *ResolvedOptions
}
