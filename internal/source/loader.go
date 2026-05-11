package source

import (
	"context"
	"encoding/json"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// LoadPlan loads a terraform plan from the given URI and returns a parsed PlanSummary.
// Supports JSON plan format (output of `terraform show -json <planfile>`).
func LoadPlan(ctx context.Context, resolver *Resolver, uri string) (*sdk.PlanSummary, error) {
	data, err := resolver.Resolve(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("loading plan: %w", err)
	}

	var plan tfjson.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parsing plan JSON from %q: %w (hint: use `terraform show -json <planfile>` to convert binary plans)", uri, err)
	}

	summary := terraform.ParsePlan(&plan)

	for i := range summary.Changes {
		summary.Changes[i].Risk = terraform.ClassifyRisk(&summary.Changes[i])
	}
	terraform.DetectPhantomChanges(summary.Changes)

	return summary, nil
}

// LoadState loads a terraform state from the given URI and returns parsed resources.
// Supports JSON state format (terraform.tfstate or output of `terraform show -json`).
func LoadState(ctx context.Context, resolver *Resolver, uri string) ([]sdk.Resource, *tfjson.State, error) {
	data, err := resolver.Resolve(ctx, uri)
	if err != nil {
		return nil, nil, fmt.Errorf("loading state: %w", err)
	}

	var state tfjson.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, nil, fmt.Errorf("parsing state JSON from %q: %w", uri, err)
	}

	if state.Values == nil {
		return []sdk.Resource{}, &state, nil
	}

	resources := terraform.ParseStateResources(state.Values.RootModule)
	return resources, &state, nil
}
