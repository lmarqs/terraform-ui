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

const defaultBinary = "terraform"

// StaticService implements sdk.Service with pre-loaded plan and state data.
// All mutating operations return CommandErr and collect the command for later retrieval.
type StaticService struct {
	plan      *sdk.PlanSummary
	resources []sdk.Resource
	state     *tfjson.State
	binary    string
	commands  []sdk.Command
}

// NewStaticService creates a read-only service pre-loaded with the given data.
// Either plan or state (or both) may be nil.
// Binary sets the terraform binary name in emitted commands (defaults to "terraform").
func NewStaticService(plan *sdk.PlanSummary, resources []sdk.Resource, state *tfjson.State, binary string) *StaticService {
	if binary == "" {
		binary = defaultBinary
	}
	return &StaticService{
		plan:      plan,
		resources: resources,
		state:     state,
		binary:    binary,
	}
}

func (s *StaticService) record(verb string, args []string, flags []string) {
	s.commands = append(s.commands, sdk.Command{
		Binary: s.binary,
		Verb:   verb,
		Args:   args,
		Flags:  flags,
	})
}

func (s *StaticService) commandErr(verb string, args []string, flags []string) error {
	cmd := sdk.Command{
		Binary: s.binary,
		Verb:   verb,
		Args:   args,
		Flags:  flags,
	}
	s.commands = append(s.commands, cmd)
	return &sdk.CommandErr{Cmd: cmd}
}

// Commands returns all commands collected during execution, in order.
func (s *StaticService) Commands() []sdk.Command {
	return s.commands
}

func (s *StaticService) Plan(_ context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
	flags := buildPlanFlags(opts)
	s.record("plan", nil, flags)
	if s.plan == nil {
		return &sdk.PlanSummary{Changes: []sdk.PlanChange{}}, nil
	}
	return s.plan, nil
}

func (s *StaticService) Apply(_ context.Context, opts sdk.ApplyOptions) error {
	flags := buildApplyFlags(opts)
	return s.commandErr("apply", nil, flags)
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

func (s *StaticService) StateList(_ context.Context) ([]sdk.Resource, error) {
	s.record("state list", nil, nil)
	if s.resources == nil {
		return []sdk.Resource{}, nil
	}
	return s.resources, nil
}

func (s *StaticService) Show(_ context.Context, address string) (string, error) {
	s.record("state show", []string{address}, nil)
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
	s.record("workspace show", nil, nil)
	return "readonly", nil
}

func (s *StaticService) WorkspaceList(_ context.Context) ([]string, error) {
	s.record("workspace list", nil, nil)
	return []string{"readonly"}, nil
}

func (s *StaticService) WorkspaceSelect(_ context.Context, name string) error {
	return s.commandErr("workspace select", []string{name}, nil)
}

func (s *StaticService) WorkspaceNew(_ context.Context, name string) error {
	return s.commandErr("workspace new", []string{name}, nil)
}

func (s *StaticService) WorkspaceDelete(_ context.Context, name string) error {
	return s.commandErr("workspace delete", []string{name}, nil)
}

func (s *StaticService) StateRm(_ context.Context, address string) error {
	return s.commandErr("state rm", []string{address}, nil)
}

func (s *StaticService) StateMove(_ context.Context, src, dst string) error {
	return s.commandErr("state mv", []string{src, dst}, nil)
}

func (s *StaticService) Import(_ context.Context, address, id string) error {
	return s.commandErr("import", []string{address, id}, nil)
}

func (s *StaticService) Taint(_ context.Context, address string) error {
	return s.commandErr("taint", []string{address}, nil)
}

func (s *StaticService) Untaint(_ context.Context, address string) error {
	return s.commandErr("untaint", []string{address}, nil)
}

func (s *StaticService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	return nil, s.commandErr("validate", nil, nil)
}

func (s *StaticService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return nil, s.commandErr("output", nil, nil)
}

func (s *StaticService) Refresh(_ context.Context) error {
	return s.commandErr("refresh", nil, nil)
}

func (s *StaticService) Init(_ context.Context) error {
	return s.commandErr("init", nil, nil)
}

func (s *StaticService) ForceUnlock(_ context.Context, lockID string) error {
	return s.commandErr("force-unlock", nil, []string{"-force", lockID})
}

func (s *StaticService) WithDir(_ string) sdk.Service {
	return s
}
