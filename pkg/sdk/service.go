package sdk

import "context"

// Service defines the interface for all terraform operations that tfui depends on.
// Implementations wrap terraform-exec or similar backends.
type Service interface {
	// Plan runs terraform plan with optional resource targets and returns
	// the parsed plan summary including changes, risk levels, and phantom detection.
	Plan(ctx context.Context, targets []string) (*PlanSummary, error)

	// Apply runs terraform apply on the previously saved plan file.
	// If targets are provided, they scope the apply to specific resources.
	Apply(ctx context.Context, targets []string) error

	// StateList returns all managed resources in the current terraform state.
	StateList(ctx context.Context) ([]Resource, error)

	// Show returns a JSON representation of a specific resource identified by address.
	Show(ctx context.Context, address string) (string, error)

	// Workspace returns the name of the currently selected terraform workspace.
	Workspace(ctx context.Context) (string, error)

	// WorkspaceList returns the names of all available terraform workspaces.
	WorkspaceList(ctx context.Context) ([]string, error)

	// WorkspaceSelect switches to the specified terraform workspace.
	WorkspaceSelect(ctx context.Context, name string) error

	// WorkspaceNew creates a new terraform workspace and switches to it.
	WorkspaceNew(ctx context.Context, name string) error

	// WorkspaceDelete deletes the specified terraform workspace.
	WorkspaceDelete(ctx context.Context, name string) error

	// WithDir returns a new Service instance scoped to the given working directory.
	WithDir(dir string) Service
}
