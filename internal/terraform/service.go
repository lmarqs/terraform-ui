package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/internal/logging"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const planFileName = "tfplan.out"

// Service is a type alias for the SDK Service interface. Internal packages and
// existing code can continue to reference terraform.Service. New code should
// prefer importing pkg/sdk directly.
type Service = sdk.Service

// TerraformService implements the Service interface using hashicorp/terraform-exec
// to shell out to the terraform (or tofu) binary.
type TerraformService struct {
	workingDir string
	binaryPath string
	statePath  string
	stateCache *tfjson.State
}

// NewService creates a new TerraformService configured with the given working
// directory and path to the terraform/tofu binary.
func NewService(workingDir, binaryPath string) *TerraformService {
	return &TerraformService{
		workingDir: workingDir,
		binaryPath: binaryPath,
	}
}

func NewServiceWithState(workingDir, binaryPath, statePath string) *TerraformService {
	return &TerraformService{
		workingDir: workingDir,
		binaryPath: binaryPath,
		statePath:  statePath,
	}
}

// WithDir returns a new TerraformService scoped to the given working directory.
func (s *TerraformService) WithDir(dir string) Service {
	return &TerraformService{
		workingDir: dir,
		binaryPath: s.binaryPath,
		statePath:  s.statePath,
	}
}

func (s *TerraformService) loadState(ctx context.Context) (*tfjson.State, error) {
	if s.stateCache != nil {
		return s.stateCache, nil
	}
	tf, err := s.newTerraform()
	if err != nil {
		return nil, err
	}
	state, err := tf.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading terraform state: %w", err)
	}
	s.stateCache = state
	return state, nil
}

func (s *TerraformService) newTerraform() (*tfexec.Terraform, error) {
	tf, err := tfexec.NewTerraform(s.workingDir, s.binaryPath)
	if err != nil {
		return nil, fmt.Errorf("creating terraform instance: %w", err)
	}
	return tf, nil
}

// Plan runs terraform plan and returns the parsed changes.
func (s *TerraformService) Plan(ctx context.Context, opts sdk.PlanOptions) (*PlanSummary, error) {
	logging.Logger().Debug("terraform.exec", "cmd", "plan", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "plan", "error", err.Error(), "duration", time.Since(start).String())
		return nil, err
	}

	planFilePath := filepath.Join(s.workingDir, planFileName)

	planOpts := []tfexec.PlanOption{
		tfexec.Out(planFilePath),
	}
	for _, t := range opts.Targets {
		planOpts = append(planOpts, tfexec.Target(t))
	}
	for _, f := range opts.VarFiles {
		planOpts = append(planOpts, tfexec.VarFile(f))
	}
	for k, v := range opts.Vars {
		planOpts = append(planOpts, tfexec.Var(k+"="+v))
	}
	for _, r := range opts.Replace {
		planOpts = append(planOpts, tfexec.Replace(r))
	}
	if opts.Destroy {
		planOpts = append(planOpts, tfexec.Destroy(true))
	}
	if opts.RefreshOnly {
		planOpts = append(planOpts, tfexec.RefreshOnly(true))
	}
	if opts.Refresh != nil {
		planOpts = append(planOpts, tfexec.Refresh(*opts.Refresh))
	}
	if opts.Parallelism > 0 {
		planOpts = append(planOpts, tfexec.Parallelism(opts.Parallelism))
	}
	if opts.Lock != nil {
		planOpts = append(planOpts, tfexec.Lock(*opts.Lock))
	}
	if opts.LockTimeout != "" {
		planOpts = append(planOpts, tfexec.LockTimeout(opts.LockTimeout))
	}

	_, err = tf.Plan(ctx, planOpts...)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "plan", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("running terraform plan: %w", err)
	}

	plan, err := tf.ShowPlanFile(ctx, planFilePath)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "plan", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("reading plan file: %w", err)
	}

	summary := ParsePlan(plan)

	for i := range summary.Changes {
		summary.Changes[i].Risk = ClassifyRisk(&summary.Changes[i])
	}

	DetectPhantomChanges(summary.Changes)

	logging.Logger().Debug("terraform.result", "cmd", "plan", "changes", len(summary.Changes), "duration", time.Since(start).String())
	return summary, nil
}

// Apply runs terraform apply on the saved plan file.
func (s *TerraformService) Apply(ctx context.Context, opts sdk.ApplyOptions) error {
	logging.Logger().Debug("terraform.exec", "cmd", "apply", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "apply", "error", err.Error(), "duration", time.Since(start).String())
		return err
	}

	planFilePath := filepath.Join(s.workingDir, planFileName)

	err = tf.Apply(ctx, tfexec.DirOrPlan(planFilePath))
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "apply", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("running terraform apply: %w", err)
	}

	_ = os.Remove(planFilePath)
	logging.Logger().Debug("terraform.result", "cmd", "apply", "duration", time.Since(start).String())
	return nil
}

// StateList returns all resources in the current state.
func (s *TerraformService) StateList(ctx context.Context) ([]Resource, error) {
	s.stateCache = nil
	state, err := s.loadState(ctx)
	if err != nil {
		return nil, err
	}

	if state == nil || state.Values == nil {
		return []Resource{}, nil
	}

	resources := ParseStateResources(state.Values.RootModule)
	return resources, nil
}

// Show returns detailed information about a specific resource.
// Sensitive attribute values are redacted before returning.
func (s *TerraformService) Show(ctx context.Context, address string) (string, error) {
	state, err := s.loadState(ctx)
	if err != nil {
		return "", err
	}

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

	output, err := json.MarshalIndent(display, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling resource: %w", err)
	}

	return string(output), nil
}

// Workspace returns the current workspace name.
func (s *TerraformService) Workspace(ctx context.Context) (string, error) {
	tf, err := s.newTerraform()
	if err != nil {
		return "", err
	}

	workspace, err := tf.WorkspaceShow(ctx)
	if err != nil {
		return "", fmt.Errorf("getting current workspace: %w", err)
	}

	return workspace, nil
}

// WorkspaceList returns all workspace names.
func (s *TerraformService) WorkspaceList(ctx context.Context) ([]string, error) {
	tf, err := s.newTerraform()
	if err != nil {
		return nil, err
	}

	workspaces, current, err := tf.WorkspaceList(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}

	// Ensure the current workspace is included in the list
	found := false
	for _, ws := range workspaces {
		if ws == current {
			found = true
			break
		}
	}
	if !found {
		workspaces = append(workspaces, current)
	}

	return workspaces, nil
}

// WorkspaceSelect switches to the specified workspace.
func (s *TerraformService) WorkspaceSelect(ctx context.Context, name string) error {
	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	if err := tf.WorkspaceSelect(ctx, name); err != nil {
		return fmt.Errorf("selecting workspace %q: %w", name, err)
	}
	return nil
}

// WorkspaceNew creates a new workspace and switches to it.
func (s *TerraformService) WorkspaceNew(ctx context.Context, name string) error {
	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	if err := tf.WorkspaceNew(ctx, name); err != nil {
		return fmt.Errorf("creating workspace %q: %w", name, err)
	}
	return nil
}

// WorkspaceDelete deletes the specified workspace.
func (s *TerraformService) WorkspaceDelete(ctx context.Context, name string) error {
	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	if err := tf.WorkspaceDelete(ctx, name); err != nil {
		return fmt.Errorf("deleting workspace %q: %w", name, err)
	}
	return nil
}

// StateRm removes a resource from state.
func (s *TerraformService) StateRm(ctx context.Context, address string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "state rm", "dir", s.workingDir, "address", address)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	var rmOpts []tfexec.StateRmCmdOption
	if s.statePath != "" {
		rmOpts = append(rmOpts, tfexec.State(s.statePath))
	}
	if err := tf.StateRm(ctx, address, rmOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "state rm", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("removing %q from state: %w", address, err)
	}

	s.stateCache = nil
	logging.Logger().Debug("terraform.result", "cmd", "state rm", "duration", time.Since(start).String())
	return nil
}

// StateMove moves a resource in state.
func (s *TerraformService) StateMove(ctx context.Context, source, dest string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "state mv", "dir", s.workingDir, "source", source, "dest", dest)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	var mvOpts []tfexec.StateMvCmdOption
	if s.statePath != "" {
		mvOpts = append(mvOpts, tfexec.State(s.statePath))
	}
	if err := tf.StateMv(ctx, source, dest, mvOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "state mv", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("moving %q to %q: %w", source, dest, err)
	}

	s.stateCache = nil
	logging.Logger().Debug("terraform.result", "cmd", "state mv", "duration", time.Since(start).String())
	return nil
}

// Import imports an existing resource into state.
func (s *TerraformService) Import(ctx context.Context, address, id string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "import", "dir", s.workingDir, "address", address, "id", id)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	var importOpts []tfexec.ImportOption
	if s.statePath != "" {
		importOpts = append(importOpts, tfexec.State(s.statePath))
	}
	if err := tf.Import(ctx, address, id, importOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "import", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("importing %q with id %q: %w", address, id, err)
	}

	s.stateCache = nil
	logging.Logger().Debug("terraform.result", "cmd", "import", "duration", time.Since(start).String())
	return nil
}

// Taint marks a resource for recreation.
// Note: terraform taint is deprecated in newer versions of Terraform in favor of
// using -replace with plan/apply. terraform-exec still supports the command.
func (s *TerraformService) Taint(ctx context.Context, address string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "taint", "dir", s.workingDir, "address", address)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	var taintOpts []tfexec.TaintOption
	if s.statePath != "" {
		taintOpts = append(taintOpts, tfexec.State(s.statePath))
	}
	if err := tf.Taint(ctx, address, taintOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "taint", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("tainting %q: %w", address, err)
	}

	logging.Logger().Debug("terraform.result", "cmd", "taint", "duration", time.Since(start).String())
	return nil
}

// Untaint removes taint from a resource.
// Note: terraform untaint is deprecated in newer versions of Terraform.
func (s *TerraformService) Untaint(ctx context.Context, address string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "untaint", "dir", s.workingDir, "address", address)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	var untaintOpts []tfexec.UntaintOption
	if s.statePath != "" {
		untaintOpts = append(untaintOpts, tfexec.State(s.statePath))
	}
	if err := tf.Untaint(ctx, address, untaintOpts...); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "untaint", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("untainting %q: %w", address, err)
	}

	logging.Logger().Debug("terraform.result", "cmd", "untaint", "duration", time.Since(start).String())
	return nil
}

// Validate runs terraform validate.
func (s *TerraformService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	logging.Logger().Debug("terraform.exec", "cmd", "validate", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return nil, err
	}

	validateOutput, err := tf.Validate(ctx)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "validate", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("running terraform validate: %w", err)
	}

	result := make([]sdk.Diagnostic, 0)
	if validateOutput != nil {
		for _, d := range validateOutput.Diagnostics {
			diag := sdk.Diagnostic{
				Severity: string(d.Severity),
				Summary:  d.Summary,
				Detail:   d.Detail,
			}
			if d.Range != nil {
				diag.File = d.Range.Filename
				diag.Line = d.Range.Start.Line
			}
			result = append(result, diag)
		}
	}

	logging.Logger().Debug("terraform.result", "cmd", "validate", "diagnostics", len(result), "duration", time.Since(start).String())
	return result, nil
}

// Output returns all terraform outputs.
func (s *TerraformService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	logging.Logger().Debug("terraform.exec", "cmd", "output", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return nil, err
	}

	outputs, err := tf.Output(ctx)
	if err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "output", "error", err.Error(), "duration", time.Since(start).String())
		return nil, fmt.Errorf("getting outputs: %w", err)
	}

	result := make(map[string]sdk.OutputValue)
	for name, meta := range outputs {
		var value interface{}
		if len(meta.Value) > 0 {
			_ = json.Unmarshal(meta.Value, &value)
		}
		result[name] = sdk.OutputValue{
			Name:      name,
			Value:     value,
			Type:      string(meta.Type),
			Sensitive: meta.Sensitive,
		}
	}

	logging.Logger().Debug("terraform.result", "cmd", "output", "count", len(result), "duration", time.Since(start).String())
	return result, nil
}

// Refresh refreshes terraform state.
func (s *TerraformService) Refresh(ctx context.Context) error {
	logging.Logger().Debug("terraform.exec", "cmd", "refresh", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	if err := tf.Refresh(ctx); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "refresh", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("refreshing state: %w", err)
	}

	s.stateCache = nil
	logging.Logger().Debug("terraform.result", "cmd", "refresh", "duration", time.Since(start).String())
	return nil
}

// Init runs terraform init.
func (s *TerraformService) Init(ctx context.Context) error {
	logging.Logger().Debug("terraform.exec", "cmd", "init", "dir", s.workingDir)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	if err := tf.Init(ctx); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "init", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("running terraform init: %w", err)
	}

	logging.Logger().Debug("terraform.result", "cmd", "init", "duration", time.Since(start).String())
	return nil
}

// ForceUnlock removes a state lock by ID.
func (s *TerraformService) ForceUnlock(ctx context.Context, lockID string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "force-unlock", "dir", s.workingDir, "lockID", lockID)
	start := time.Now()

	tf, err := s.newTerraform()
	if err != nil {
		return err
	}

	if err := tf.ForceUnlock(ctx, lockID); err != nil {
		logging.Logger().Debug("terraform.result", "cmd", "force-unlock", "error", err.Error(), "duration", time.Since(start).String())
		return fmt.Errorf("force-unlocking state (ID %s): %w", lockID, err)
	}

	logging.Logger().Debug("terraform.result", "cmd", "force-unlock", "duration", time.Since(start).String())
	return nil
}

// ParsePlan converts a tfjson.Plan into a PlanSummary.
func ParsePlan(plan *tfjson.Plan) *PlanSummary {
	summary := &PlanSummary{
		Changes: make([]PlanChange, 0),
	}

	if plan == nil || plan.ResourceChanges == nil {
		return summary
	}

	for _, rc := range plan.ResourceChanges {
		if rc.Change == nil {
			continue
		}

		action := mapActions(rc.Change.Actions)
		if action == ActionNoOp || action == ActionRead {
			if action == ActionRead {
				summary.ToRead++
			}
			continue
		}

		change := PlanChange{
			Resource: Resource{
				Address:      rc.Address,
				Type:         rc.Type,
				Name:         rc.Name,
				Module:       ExtractModule(rc.Address),
				ProviderName: rc.ProviderName,
			},
			Action:         action,
			AttributeDiffs: parseAttributeDiffs(rc.Change),
		}

		summary.Changes = append(summary.Changes, change)

		switch action {
		case ActionCreate:
			summary.ToCreate++
		case ActionUpdate:
			summary.ToUpdate++
		case ActionDelete:
			summary.ToDelete++
		case ActionDeleteThenCreate, ActionCreateThenDelete:
			summary.ToReplace++
		}
	}

	return summary
}

// mapActions converts tfjson.Actions to our Action type.
func mapActions(actions tfjson.Actions) Action {
	switch {
	case actions.NoOp():
		return ActionNoOp
	case actions.Read():
		return ActionRead
	case actions.Create():
		return ActionCreate
	case actions.Update():
		return ActionUpdate
	case actions.Delete():
		return ActionDelete
	case actions.DestroyBeforeCreate():
		return ActionDeleteThenCreate
	case actions.CreateBeforeDestroy():
		return ActionCreateThenDelete
	default:
		return ActionNoOp
	}
}

// parseAttributeDiffs extracts attribute diffs from a resource change.
func parseAttributeDiffs(change *tfjson.Change) []AttributeDiff {
	diffs := make([]AttributeDiff, 0)

	if change.Before == nil && change.After == nil {
		return diffs
	}

	beforeMap := jsonToMap(change.Before)
	afterMap := jsonToMap(change.After)

	// Collect all keys
	keys := make(map[string]bool)
	for k := range beforeMap {
		keys[k] = true
	}
	for k := range afterMap {
		keys[k] = true
	}

	for key := range keys {
		oldVal := marshalValue(beforeMap[key])
		newVal := marshalValue(afterMap[key])

		if oldVal == newVal {
			continue
		}

		sensitive := false
		if change.BeforeSensitive != nil || change.AfterSensitive != nil {
			sensitive = isKeySensitive(change.BeforeSensitive, key) ||
				isKeySensitive(change.AfterSensitive, key)
		}

		diffs = append(diffs, AttributeDiff{
			Key:       key,
			OldValue:  oldVal,
			NewValue:  newVal,
			Sensitive: sensitive,
		})
	}

	return diffs
}

// jsonToMap converts a raw JSON interface{} (from tfjson) to a map of string to interface{}.
func jsonToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return make(map[string]interface{})
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return make(map[string]interface{})
}

// marshalValue converts a value to its JSON string representation.
func marshalValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// isKeySensitive checks if a specific key is marked as sensitive.
func isKeySensitive(sensitive interface{}, key string) bool {
	if sensitive == nil {
		return false
	}
	switch s := sensitive.(type) {
	case bool:
		return s
	case map[string]interface{}:
		if v, ok := s[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
		}
	}
	return false
}

// RedactSensitiveValues replaces sensitive attribute values with "(sensitive)".
func RedactSensitiveValues(values map[string]interface{}, sensitive interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	result := make(map[string]interface{}, len(values))
	for k, v := range values {
		if isSensitiveKey(sensitive, k) {
			result[k] = "(sensitive)"
		} else {
			result[k] = v
		}
	}
	return result
}

func isSensitiveKey(sensitive interface{}, key string) bool {
	if sensitive == nil {
		return false
	}
	switch s := sensitive.(type) {
	case bool:
		return s
	case map[string]interface{}:
		if v, ok := s[key]; ok {
			if b, ok := v.(bool); ok {
				return b
			}
			return v != nil
		}
	}
	return false
}

// ParseStateResources recursively extracts resources from a state module.
func ParseStateResources(module *tfjson.StateModule) []Resource {
	if module == nil {
		return []Resource{}
	}

	resources := make([]Resource, 0)

	for _, r := range module.Resources {
		resources = append(resources, Resource{
			Address:      r.Address,
			Type:         r.Type,
			Name:         r.Name,
			Module:       ExtractModule(r.Address),
			ProviderName: r.ProviderName,
		})
	}

	for _, child := range module.ChildModules {
		resources = append(resources, ParseStateResources(child)...)
	}

	return resources
}

// FindResourceInState searches for a resource by address in the state module tree.
func FindResourceInState(module *tfjson.StateModule, address string) *tfjson.StateResource {
	if module == nil {
		return nil
	}

	for _, r := range module.Resources {
		if r.Address == address {
			return r
		}
	}

	for _, child := range module.ChildModules {
		if r := FindResourceInState(child, address); r != nil {
			return r
		}
	}

	return nil
}
