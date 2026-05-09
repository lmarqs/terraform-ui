package terraform

import (
	"context"
)

// Service defines the interface for terraform operations.
type Service interface {
	// Plan runs terraform plan and returns the parsed changes.
	Plan(ctx context.Context, targets []string) (*PlanSummary, error)

	// Apply runs terraform apply and streams progress.
	Apply(ctx context.Context, targets []string) error

	// StateList returns all resources in the current state.
	StateList(ctx context.Context) ([]Resource, error)

	// Show returns detailed information about a specific resource.
	Show(ctx context.Context, address string) (string, error)

	// Workspace returns the current workspace name.
	Workspace(ctx context.Context) (string, error)
}

// TerraformService implements Service using terraform-exec.
type TerraformService struct {
	workingDir string
	binaryPath string
}

// NewService creates a new TerraformService.
func NewService(workingDir, binaryPath string) *TerraformService {
	return &TerraformService{
		workingDir: workingDir,
		binaryPath: binaryPath,
	}
}

// Plan runs terraform plan and returns the parsed changes.
func (s *TerraformService) Plan(ctx context.Context, targets []string) (*PlanSummary, error) {
	// TODO: implement using terraform-exec
	return &PlanSummary{}, nil
}

// Apply runs terraform apply and streams progress.
func (s *TerraformService) Apply(ctx context.Context, targets []string) error {
	// TODO: implement using terraform-exec
	return nil
}

// StateList returns all resources in the current state.
func (s *TerraformService) StateList(ctx context.Context) ([]Resource, error) {
	// TODO: implement using terraform-exec
	return []Resource{}, nil
}

// Show returns detailed information about a specific resource.
func (s *TerraformService) Show(ctx context.Context, address string) (string, error) {
	// TODO: implement using terraform-exec
	return "", nil
}

// Workspace returns the current workspace name.
func (s *TerraformService) Workspace(ctx context.Context) (string, error) {
	// TODO: implement using terraform-exec
	return "default", nil
}
