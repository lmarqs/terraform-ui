package sdk

import (
	"context"
)

// OutputValue represents a terraform output.
type OutputValue struct {
	Name      string
	Value     interface{}
	Type      string
	Sensitive bool
}

// VersionInfo holds terraform binary version and provider selections.
type VersionInfo struct {
	TerraformVersion string
	Providers        map[string]string
}

type stateListConfig struct{ skipCache bool }

// StateListOption configures the behavior of StateList.
type StateListOption func(*stateListConfig)

// SkipCache forces StateList to bypass cached data and re-read from the backend.
func SkipCache() StateListOption {
	return func(c *stateListConfig) { c.skipCache = true }
}

// ApplyStateListOptions resolves variadic options into a config struct.
func ApplyStateListOptions(opts []StateListOption) stateListConfig {
	var cfg stateListConfig
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// ShouldSkipCache returns true if the options request a cache bypass.
func (c stateListConfig) ShouldSkipCache() bool {
	return c.skipCache
}

// Service defines the interface for all terraform operations that tfui depends on.
// Implementations wrap terraform-exec or similar backends.
type Service interface {
	// Plan runs terraform plan with the given options and returns
	// the parsed plan summary including changes, risk levels, and phantom detection.
	Plan(ctx context.Context, opts PlanOptions) (*PlanSummary, error)

	// PlanJSON runs terraform plan and then `terraform show -json` against the
	// resulting plan file, returning the raw bytes terraform produced. The
	// schema is terraform's, not tfui's.
	PlanJSON(ctx context.Context, opts PlanOptions) ([]byte, error)

	// Apply runs terraform apply with the given options.
	Apply(ctx context.Context, opts ApplyOptions) error

	// StateList returns all managed resources in the current terraform state.
	StateList(ctx context.Context, opts ...StateListOption) ([]Resource, error)

	// Show returns a JSON representation of a specific resource identified by address.
	Show(ctx context.Context, address string) (string, error)

	// Workspace returns the name of the currently selected terraform workspace.
	Workspace(ctx context.Context) (string, error)

	// WorkspaceList returns the names of all available terraform workspaces.
	WorkspaceList(ctx context.Context) ([]string, error)

	// WorkspaceSelect switches to the specified terraform workspace.
	WorkspaceSelect(ctx context.Context, name string) error

	// WorkspaceNew creates a new terraform workspace and switches to it.
	WorkspaceNew(ctx context.Context, name string, opts WorkspaceNewOptions) error

	// WorkspaceDelete deletes the specified terraform workspace.
	WorkspaceDelete(ctx context.Context, name string, opts WorkspaceDeleteOptions) error

	// StateRm removes a resource from terraform state by address.
	StateRm(ctx context.Context, address string) error

	// StateMove moves a resource from one address to another in state.
	StateMove(ctx context.Context, source, dest string) error

	// Import imports an existing infrastructure resource into terraform state.
	Import(ctx context.Context, address, id string) error

	// Taint marks a resource as tainted, forcing recreation on next apply.
	Taint(ctx context.Context, address string) error

	// Untaint removes the taint from a resource.
	Untaint(ctx context.Context, address string) error

	// Validate runs terraform validate and returns diagnostics.
	Validate(ctx context.Context) ([]Diagnostic, error)

	// ValidateJSON runs terraform validate -json and returns the raw bytes
	// terraform produced. The schema is terraform's, not tfui's.
	ValidateJSON(ctx context.Context) ([]byte, error)

	// Output returns all terraform outputs.
	Output(ctx context.Context) (map[string]OutputValue, error)

	// OutputJSON runs terraform output -json and returns the raw bytes
	// terraform produced. The schema is terraform's, not tfui's.
	OutputJSON(ctx context.Context) ([]byte, error)

	// Refresh refreshes the state to match real infrastructure.
	Refresh(ctx context.Context) error

	// Init runs terraform init in the working directory.
	Init(ctx context.Context, opts InitOptions) error

	// ForceUnlock removes a state lock by ID.
	ForceUnlock(ctx context.Context, lockID string) error

	// Version returns the terraform binary version and provider selections.
	Version(ctx context.Context) (*VersionInfo, error)

	// VersionJSON runs terraform version -json and returns the raw bytes
	// terraform produced. The schema is terraform's, not tfui's.
	VersionJSON(ctx context.Context) ([]byte, error)

	// WithDir returns a new Service instance scoped to the given working directory.
	WithDir(dir string) Service
}
