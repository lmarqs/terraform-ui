package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type CompositeService struct {
	live       sdk.Service
	planFile   string
	stateFile  string
	stdinPlan  []byte
	stdinState []byte
}

func NewCompositeService(live sdk.Service, planFile, stateFile string, stdinPlan, stdinState []byte) *CompositeService {
	return &CompositeService{
		live:       live,
		planFile:   planFile,
		stateFile:  stateFile,
		stdinPlan:  stdinPlan,
		stdinState: stdinState,
	}
}

func (c *CompositeService) Plan(_ context.Context, opts sdk.PlanOptions) (*PlanSummary, error) {
	if c.planFile != "" {
		data, err := os.ReadFile(c.planFile)
		if err != nil {
			return nil, fmt.Errorf("reading plan file %s: %w", c.planFile, err)
		}
		return LoadPlan(data)
	}
	if c.stdinPlan != nil {
		return LoadPlan(c.stdinPlan)
	}
	return c.live.Plan(context.Background(), opts)
}

func (c *CompositeService) Apply(ctx context.Context, opts sdk.ApplyOptions) error {
	return c.live.Apply(ctx, opts)
}

func (c *CompositeService) StateList(_ context.Context) ([]sdk.Resource, error) {
	if c.stateFile != "" {
		resources, _, err := c.loadStateFromFile()
		return resources, err
	}
	if c.stdinState != nil {
		resources, _, err := LoadState(c.stdinState)
		return resources, err
	}
	return c.live.StateList(context.Background())
}

func (c *CompositeService) Show(_ context.Context, address string) (string, error) {
	if c.stateFile != "" {
		_, state, err := c.loadStateFromFile()
		if err != nil {
			return "", err
		}
		return c.showFromState(state, address)
	}
	if c.stdinState != nil {
		_, state, err := LoadState(c.stdinState)
		if err != nil {
			return "", err
		}
		return c.showFromState(state, address)
	}
	return c.live.Show(context.Background(), address)
}

func (c *CompositeService) showFromState(state *tfjson.State, address string) (string, error) {
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

func (c *CompositeService) Workspace(ctx context.Context) (string, error) {
	return c.live.Workspace(ctx)
}

func (c *CompositeService) WorkspaceList(ctx context.Context) ([]string, error) {
	return c.live.WorkspaceList(ctx)
}

func (c *CompositeService) WorkspaceSelect(ctx context.Context, name string) error {
	return c.live.WorkspaceSelect(ctx, name)
}

func (c *CompositeService) WorkspaceNew(ctx context.Context, name string) error {
	return c.live.WorkspaceNew(ctx, name)
}

func (c *CompositeService) WorkspaceDelete(ctx context.Context, name string) error {
	return c.live.WorkspaceDelete(ctx, name)
}

func (c *CompositeService) StateRm(ctx context.Context, address string) error {
	return c.live.StateRm(ctx, address)
}

func (c *CompositeService) StateMove(ctx context.Context, source, dest string) error {
	return c.live.StateMove(ctx, source, dest)
}

func (c *CompositeService) Import(ctx context.Context, address, id string) error {
	return c.live.Import(ctx, address, id)
}

func (c *CompositeService) Taint(ctx context.Context, address string) error {
	return c.live.Taint(ctx, address)
}

func (c *CompositeService) Untaint(ctx context.Context, address string) error {
	return c.live.Untaint(ctx, address)
}

func (c *CompositeService) Validate(ctx context.Context) ([]sdk.Diagnostic, error) {
	return c.live.Validate(ctx)
}

func (c *CompositeService) Output(ctx context.Context) (map[string]sdk.OutputValue, error) {
	return c.live.Output(ctx)
}

func (c *CompositeService) Refresh(_ context.Context) error {
	if c.planFile != "" || c.stdinPlan != nil {
		return nil
	}
	if c.stateFile != "" || c.stdinState != nil {
		return nil
	}
	return c.live.Refresh(context.Background())
}

func (c *CompositeService) Init(ctx context.Context) error {
	return c.live.Init(ctx)
}

func (c *CompositeService) ForceUnlock(ctx context.Context, lockID string) error {
	return c.live.ForceUnlock(ctx, lockID)
}

func (c *CompositeService) WithDir(dir string) sdk.Service {
	return &CompositeService{
		live:       c.live.WithDir(dir),
		planFile:   c.planFile,
		stateFile:  c.stateFile,
		stdinPlan:  c.stdinPlan,
		stdinState: c.stdinState,
	}
}

func (c *CompositeService) loadStateFromFile() ([]sdk.Resource, *tfjson.State, error) {
	data, err := os.ReadFile(c.stateFile)
	if err != nil {
		return nil, nil, fmt.Errorf("reading state file %s: %w", c.stateFile, err)
	}
	return LoadState(data)
}
