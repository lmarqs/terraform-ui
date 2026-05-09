package terraform

import (
	"context"
	"encoding/json"
	"fmt"
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
}

// NewService creates a new TerraformService configured with the given working
// directory and path to the terraform/tofu binary.
func NewService(workingDir, binaryPath string) *TerraformService {
	return &TerraformService{
		workingDir: workingDir,
		binaryPath: binaryPath,
	}
}

func (s *TerraformService) newTerraform() (*tfexec.Terraform, error) {
	tf, err := tfexec.NewTerraform(s.workingDir, s.binaryPath)
	if err != nil {
		return nil, fmt.Errorf("creating terraform instance: %w", err)
	}
	return tf, nil
}

// Plan runs terraform plan and returns the parsed changes.
func (s *TerraformService) Plan(ctx context.Context, targets []string) (*PlanSummary, error) {
	logging.Logger().Debug("terraform.exec", "cmd", "plan", "dir", s.workingDir, "targets", targets)
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
	for _, t := range targets {
		planOpts = append(planOpts, tfexec.Target(t))
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

	summary := parsePlan(plan)

	for i := range summary.Changes {
		summary.Changes[i].Risk = ClassifyRisk(&summary.Changes[i])
	}

	DetectPhantomChanges(summary.Changes)

	logging.Logger().Debug("terraform.result", "cmd", "plan", "changes", len(summary.Changes), "duration", time.Since(start).String())
	return summary, nil
}

// Apply runs terraform apply on the saved plan file.
func (s *TerraformService) Apply(ctx context.Context, targets []string) error {
	logging.Logger().Debug("terraform.exec", "cmd", "apply", "dir", s.workingDir, "targets", targets)
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

	logging.Logger().Debug("terraform.result", "cmd", "apply", "duration", time.Since(start).String())
	return nil
}

// StateList returns all resources in the current state.
func (s *TerraformService) StateList(ctx context.Context) ([]Resource, error) {
	tf, err := s.newTerraform()
	if err != nil {
		return nil, err
	}

	state, err := tf.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading terraform state: %w", err)
	}

	if state == nil || state.Values == nil {
		return []Resource{}, nil
	}

	resources := parseStateResources(state.Values.RootModule)
	return resources, nil
}

// Show returns detailed information about a specific resource.
func (s *TerraformService) Show(ctx context.Context, address string) (string, error) {
	tf, err := s.newTerraform()
	if err != nil {
		return "", err
	}

	state, err := tf.Show(ctx)
	if err != nil {
		return "", fmt.Errorf("reading terraform state: %w", err)
	}

	if state == nil || state.Values == nil {
		return "", fmt.Errorf("no state available")
	}

	resource := findResourceInState(state.Values.RootModule, address)
	if resource == nil {
		return "", fmt.Errorf("resource %q not found in state", address)
	}

	output, err := json.MarshalIndent(resource, "", "  ")
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

// parsePlan converts a tfjson.Plan into a PlanSummary.
func parsePlan(plan *tfjson.Plan) *PlanSummary {
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

// parseStateResources recursively extracts resources from a state module.
func parseStateResources(module *tfjson.StateModule) []Resource {
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
		resources = append(resources, parseStateResources(child)...)
	}

	return resources
}

// findResourceInState searches for a resource by address in the state module tree.
func findResourceInState(module *tfjson.StateModule, address string) *tfjson.StateResource {
	if module == nil {
		return nil
	}

	for _, r := range module.Resources {
		if r.Address == address {
			return r
		}
	}

	for _, child := range module.ChildModules {
		if r := findResourceInState(child, address); r != nil {
			return r
		}
	}

	return nil
}
