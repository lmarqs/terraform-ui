package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const defaultBinary = "terraform"

type commandStore struct {
	mu       sync.Mutex
	commands []sdk.Command
}

func (s *commandStore) append(cmd sdk.Command) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append(s.commands, cmd)
}

func (s *commandStore) all() []sdk.Command {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commands
}

// MacroService records terraform operations as sdk.Command without executing them.
// Reads are served from a ServiceCache (pre-seeded or empty).
type MacroService struct {
	binary   string
	cache    *ServiceCache
	store    *commandStore
	applyErr error
}

// SetApplyError configures an error to be returned by Apply.
func (r *MacroService) SetApplyError(err error) {
	r.applyErr = err
}

// NewMacroService creates a MacroService that records commands and reads from cache.
func NewMacroService(binary string, cache *ServiceCache) *MacroService {
	if binary == "" {
		binary = defaultBinary
	}
	if cache == nil {
		cache = NewServiceCache()
	}
	return &MacroService{
		binary: binary,
		cache:  cache,
		store:  &commandStore{},
	}
}

// Commands returns all recorded commands in order.
// Returns nil when nothing was recorded.
func (r *MacroService) Commands() []sdk.Command {
	return r.store.all()
}

func (r *MacroService) record(verb string, args, flags []string) {
	r.store.append(sdk.Command{
		Binary: r.binary,
		Verb:   verb,
		Args:   args,
		Flags:  flags,
	})
}

func (r *MacroService) Plan(_ context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
	r.record("plan", nil, buildPlanFlags(opts))
	if plan, ok := r.cache.GetPlan(); ok {
		return plan, nil
	}
	return &sdk.PlanSummary{}, nil
}

func (r *MacroService) Apply(_ context.Context, opts sdk.ApplyOptions) error {
	r.record("apply", nil, buildApplyFlags(opts))
	return r.applyErr
}

func (r *MacroService) StateList(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
	if resources, ok := r.cache.GetResources(); ok {
		return resources, nil
	}
	return []sdk.Resource{}, nil
}

func (r *MacroService) Show(_ context.Context, address string) (string, error) {
	if state, ok := r.cache.GetState(); ok {
		return showFromState(state, address)
	}
	return "{}", nil
}

func (r *MacroService) Workspace(_ context.Context) (string, error) {
	return "default", nil
}

func (r *MacroService) WorkspaceList(_ context.Context) ([]string, error) {
	if r.cache != nil {
		if ws, ok := r.cache.GetWorkspaces(); ok {
			return ws, nil
		}
	}
	return []string{"default"}, nil
}

func (r *MacroService) WorkspaceSelect(_ context.Context, name string) error {
	r.record("workspace select", []string{name}, nil)
	return nil
}

func (r *MacroService) WorkspaceNew(_ context.Context, name string, _ sdk.WorkspaceNewOptions) error {
	r.record("workspace new", []string{name}, nil)
	return nil
}

func (r *MacroService) WorkspaceDelete(_ context.Context, name string, _ sdk.WorkspaceDeleteOptions) error {
	r.record("workspace delete", []string{name}, nil)
	return nil
}

func (r *MacroService) StateRm(_ context.Context, address string) error {
	r.record("state rm", []string{address}, nil)
	return nil
}

func (r *MacroService) StateMove(_ context.Context, src, dst string) error {
	r.record("state mv", []string{src, dst}, nil)
	return nil
}

func (r *MacroService) Import(_ context.Context, address, id string) error {
	r.record("import", []string{address, id}, nil)
	return nil
}

func (r *MacroService) Taint(_ context.Context, address string) error {
	r.record("taint", []string{address}, nil)
	return nil
}

func (r *MacroService) Untaint(_ context.Context, address string) error {
	r.record("untaint", []string{address}, nil)
	return nil
}

func (r *MacroService) Validate(_ context.Context) ([]sdk.Diagnostic, error) {
	if r.cache != nil {
		if diags, ok := r.cache.GetDiagnostics(); ok {
			return diags, nil
		}
	}
	return []sdk.Diagnostic{}, nil
}

func (r *MacroService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	if r.cache != nil {
		if outputs, ok := r.cache.GetOutputs(); ok {
			return outputs, nil
		}
	}
	return map[string]sdk.OutputValue{}, nil
}

func (r *MacroService) Refresh(_ context.Context) error {
	r.record("refresh", nil, nil)
	r.cache.InvalidateAll()
	return nil
}

func (r *MacroService) Init(_ context.Context, opts sdk.InitOptions) error {
	r.record("init", nil, buildInitFlags(opts))
	return nil
}

func buildInitFlags(opts sdk.InitOptions) []string {
	var flags []string
	if opts.Upgrade {
		flags = append(flags, "-upgrade")
	}
	if opts.Reconfigure {
		flags = append(flags, "-reconfigure")
	}
	if opts.Backend != nil && !*opts.Backend {
		flags = append(flags, "-backend=false")
	}
	for _, bc := range opts.BackendConfig {
		flags = append(flags, "-backend-config="+bc)
	}
	flags = append(flags, opts.ExtraArgs...)
	return flags
}

func (r *MacroService) Version(_ context.Context) (*sdk.VersionInfo, error) {
	return &sdk.VersionInfo{TerraformVersion: "0.0.0"}, nil
}

func (r *MacroService) ForceUnlock(_ context.Context, lockID string) error {
	r.record("force-unlock", nil, []string{"-force", lockID})
	return nil
}

func (r *MacroService) WithDir(dir string) sdk.Service {
	return &MacroService{
		binary: r.binary,
		cache:  NewServiceCache(),
		store:  r.store,
	}
}

// showFromState looks up a resource in tfjson.State and returns JSON.
func showFromState(state *tfjson.State, address string) (string, error) {
	if state == nil || state.Values == nil {
		return "", fmt.Errorf("no state available")
	}
	resource := FindResourceInState(state.Values.RootModule, address)
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
	// json.MarshalIndent cannot fail here: the struct contains only strings and
	// a map[string]interface{} produced by json.Unmarshal (JSON-safe types).
	output, _ := json.MarshalIndent(display, "", "  ")
	return string(output), nil
}

func buildPlanFlags(opts sdk.PlanOptions) []string {
	var flags []string
	if opts.PlanFile != "" {
		flags = append(flags, "-out="+opts.PlanFile)
	}
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
	if opts.PlanFile != "" {
		flags = append(flags, opts.PlanFile)
	}
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
