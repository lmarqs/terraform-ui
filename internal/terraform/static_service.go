package terraform

import (
	"context"
	"encoding/json"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const defaultBinary = "terraform"

// StaticService implements sdk.Service with pre-loaded plan and state data.
// It serves data without executing terraform, returning nil from mutating operations.
type StaticService struct {
	plan      *sdk.PlanSummary
	resources []sdk.Resource
	state     *tfjson.State
}

// NewStaticService creates a service pre-loaded with the given data.
// Either plan or state (or both) may be nil.
func NewStaticService(plan *sdk.PlanSummary, resources []sdk.Resource, state *tfjson.State) *StaticService {
	return &StaticService{
		plan:      plan,
		resources: resources,
		state:     state,
	}
}

func buildPlanFlags(opts sdk.PlanOptions) []string {
	var flags []string
	for _, t := range opts.Targets {
		flags = append(flags, "-target="+t)
	}
	for _, f := range opts.VarFiles {
		flags = append(flags, "-var-file="+f)
	}
	for k, v := range opts.Vars {
		flags = append(flags, "-var", k+"="+v)
	}
	for _, r := range opts.Replace {
		flags = append(flags, "-replace="+r)
	}
	if opts.Destroy {
		flags = append(flags, "-destroy")
	}
	if opts.RefreshOnly {
		flags = append(flags, "-refresh-only")
	}
	if opts.Refresh != nil && !*opts.Refresh {
		flags = append(flags, "-refresh=false")
	}
	if opts.Parallelism > 0 {
		flags = append(flags, fmt.Sprintf("-parallelism=%d", opts.Parallelism))
	}
	if opts.Lock != nil && !*opts.Lock {
		flags = append(flags, "-lock=false")
	}
	if opts.LockTimeout != "" {
		flags = append(flags, "-lock-timeout="+opts.LockTimeout)
	}
	flags = append(flags, opts.ExtraArgs...)
	return flags
}

func buildApplyFlags(opts sdk.ApplyOptions) []string {
	var flags []string
	for _, t := range opts.Targets {
		flags = append(flags, "-target="+t)
	}
	for _, f := range opts.VarFiles {
		flags = append(flags, "-var-file="+f)
	}
	for k, v := range opts.Vars {
		flags = append(flags, "-var", k+"="+v)
	}
	if opts.Parallelism > 0 {
		flags = append(flags, fmt.Sprintf("-parallelism=%d", opts.Parallelism))
	}
	if opts.Lock != nil && !*opts.Lock {
		flags = append(flags, "-lock=false")
	}
	if opts.LockTimeout != "" {
		flags = append(flags, "-lock-timeout="+opts.LockTimeout)
	}
	flags = append(flags, opts.ExtraArgs...)
	return flags
}

func (s *StaticService) Plan(_ context.Context, _ sdk.PlanOptions) (*sdk.PlanSummary, error) {
	if s.plan == nil {
		return &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, nil
	}
	return s.plan, nil
}

func (s *StaticService) Apply(_ context.Context, _ sdk.ApplyOptions) error {
	return nil
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
	return nil
}

func (s *StaticService) WorkspaceNew(_ context.Context, _ string) error {
	return nil
}

func (s *StaticService) WorkspaceDelete(_ context.Context, _ string) error {
	return nil
}

func (s *StaticService) StateRm(_ context.Context, _ string) error {
	return nil
}

func (s *StaticService) StateMove(_ context.Context, _, _ string) error {
	return nil
}

func (s *StaticService) Import(_ context.Context, _, _ string) error {
	return nil
}

func (s *StaticService) Taint(_ context.Context, _ string) error {
	return nil
}

func (s *StaticService) Untaint(_ context.Context, _ string) error {
	return nil
}

func (s *StaticService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	return nil, nil
}

func (s *StaticService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, nil
}

func (s *StaticService) Refresh(_ context.Context) error {
	return nil
}

func (s *StaticService) Init(_ context.Context) error {
	return nil
}

func (s *StaticService) ForceUnlock(_ context.Context, _ string) error {
	return nil
}

func (s *StaticService) WithDir(_ string) sdk.Service {
	return s
}
