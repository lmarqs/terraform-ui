package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ErrReadOnly is returned by mutating operations on a StaticService.
var ErrReadOnly = errors.New("operation not available in read-only mode")

// StaticService implements sdk.Service with pre-loaded plan and state data.
// All mutating operations return ErrReadOnly.
type StaticService struct {
	plan      *sdk.PlanSummary
	resources []sdk.Resource
	state     *tfjson.State
}

// NewStaticService creates a read-only service pre-loaded with the given data.
// Either plan or state (or both) may be nil.
func NewStaticService(plan *sdk.PlanSummary, resources []sdk.Resource, state *tfjson.State) *StaticService {
	return &StaticService{
		plan:      plan,
		resources: resources,
		state:     state,
	}
}

func (s *StaticService) Plan(_ context.Context, _ []string) (*sdk.PlanSummary, error) {
	if s.plan == nil {
		return &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, nil
	}
	return s.plan, nil
}

func (s *StaticService) Apply(_ context.Context, _ []string) error {
	return fmt.Errorf("apply: %w", ErrReadOnly)
}

func (s *StaticService) StateList(_ context.Context) ([]sdk.Resource, error) {
	if s.resources == nil {
		return []sdk.Resource{}, nil
	}
	return s.resources, nil
}

func (s *StaticService) Show(_ context.Context, address string) (string, error) {
	if s.state == nil || s.state.Values == nil {
		return "", fmt.Errorf("no state available")
	}

	resource := FindResourceInState(s.state.Values.RootModule, address)
	if resource == nil {
		return "", fmt.Errorf("resource %q not found in state", address)
	}

	redacted := RedactSensitiveValues(resource.AttributeValues, resource.SensitiveValues)

	display := struct {
		Address      string                 `json:"address"`
		Type         string                 `json:"type"`
		Name         string                 `json:"name"`
		ProviderName string                 `json:"provider_name"`
		Values       map[string]interface{} `json:"values"`
	}{
		Address:      resource.Address,
		Type:         resource.Type,
		Name:         resource.Name,
		ProviderName: resource.ProviderName,
		Values:       redacted,
	}

	output, err := json.MarshalIndent(display, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling resource: %w", err)
	}
	return string(output), nil
}

func (s *StaticService) Workspace(_ context.Context) (string, error) {
	return "readonly", nil
}

func (s *StaticService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"readonly"}, nil
}

func (s *StaticService) WorkspaceSelect(_ context.Context, _ string) error {
	return fmt.Errorf("workspace select: %w", ErrReadOnly)
}

func (s *StaticService) WorkspaceNew(_ context.Context, _ string) error {
	return fmt.Errorf("workspace new: %w", ErrReadOnly)
}

func (s *StaticService) WorkspaceDelete(_ context.Context, _ string) error {
	return fmt.Errorf("workspace delete: %w", ErrReadOnly)
}

func (s *StaticService) StateRm(_ context.Context, _ string) error {
	return fmt.Errorf("state rm: %w", ErrReadOnly)
}

func (s *StaticService) StateMove(_ context.Context, _, _ string) error {
	return fmt.Errorf("state mv: %w", ErrReadOnly)
}

func (s *StaticService) Import(_ context.Context, _, _ string) error {
	return fmt.Errorf("import: %w", ErrReadOnly)
}

func (s *StaticService) Taint(_ context.Context, _ string) error {
	return fmt.Errorf("taint: %w", ErrReadOnly)
}

func (s *StaticService) Untaint(_ context.Context, _ string) error {
	return fmt.Errorf("untaint: %w", ErrReadOnly)
}

func (s *StaticService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	return nil, fmt.Errorf("validate: %w", ErrReadOnly)
}

func (s *StaticService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, fmt.Errorf("output: %w", ErrReadOnly)
}

func (s *StaticService) Refresh(_ context.Context) error {
	return fmt.Errorf("refresh: %w", ErrReadOnly)
}

func (s *StaticService) Init(_ context.Context) error {
	return fmt.Errorf("init: %w", ErrReadOnly)
}

func (s *StaticService) ForceUnlock(_ context.Context, _ string) error {
	return fmt.Errorf("force-unlock: %w", ErrReadOnly)
}

func (s *StaticService) WithDir(_ string) sdk.Service {
	return s
}
