package source

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
// Supports two formats:
//   - Raw tfstate (output of `terraform state pull`): {"version": 4, "resources": [...]}
//   - Show JSON (output of `terraform show -json`): {"format_version": "1.0", "values": {...}}
func LoadState(ctx context.Context, resolver *Resolver, uri string) ([]sdk.Resource, *tfjson.State, error) {
	data, err := resolver.Resolve(ctx, uri)
	if err != nil {
		return nil, nil, fmt.Errorf("loading state: %w", err)
	}

	// Detect format by checking for "format_version" (show -json) vs "version" (raw tfstate)
	if isShowJSON(data) {
		return parseShowState(data, uri)
	}
	return parseRawState(data, uri)
}

func isShowJSON(data []byte) bool {
	var probe struct {
		FormatVersion string `json:"format_version"`
	}
	_ = json.Unmarshal(data, &probe)
	return probe.FormatVersion != ""
}

func parseShowState(data []byte, uri string) ([]sdk.Resource, *tfjson.State, error) {
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

// rawState represents the raw terraform.tfstate format (output of `terraform state pull`).
type rawState struct {
	Version   int           `json:"version"`
	Resources []rawResource `json:"resources"`
}

type rawResource struct {
	Module    string        `json:"module"`
	Mode      string        `json:"mode"`
	Type      string        `json:"type"`
	Name      string        `json:"name"`
	Provider  string        `json:"provider"`
	Instances []rawInstance `json:"instances"`
}

type rawInstance struct {
	IndexKey       interface{}            `json:"index_key"`
	Attributes     map[string]interface{} `json:"attributes"`
	SensitiveAttrs json.RawMessage        `json:"sensitive_attributes"`
}

func parseRawState(data []byte, uri string) ([]sdk.Resource, *tfjson.State, error) {
	var raw rawState
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("parsing state from %q: %w", uri, err)
	}

	if raw.Version == 0 {
		return nil, nil, fmt.Errorf("parsing state from %q: not a valid terraform state file (no version field)", uri)
	}

	// Convert raw state to tfjson.State for StaticService.Show() compatibility
	stateResources := make([]*tfjson.StateResource, 0)
	sdkResources := make([]sdk.Resource, 0)

	for _, r := range raw.Resources {
		if r.Mode == "data" {
			continue
		}

		for _, inst := range r.Instances {
			address := buildAddress(r.Module, r.Type, r.Name, inst.IndexKey)
			providerName := cleanProvider(r.Provider)

			stateResources = append(stateResources, &tfjson.StateResource{
				Address:         address,
				Type:            r.Type,
				Name:            r.Name,
				ProviderName:    providerName,
				AttributeValues: inst.Attributes,
			})

			sdkResources = append(sdkResources, sdk.Resource{
				Address:      address,
				Type:         r.Type,
				Name:         r.Name,
				Module:       terraform.ExtractModule(address),
				ProviderName: providerName,
			})
		}
	}

	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: stateResources,
			},
		},
	}

	return sdkResources, state, nil
}

func buildAddress(module, resourceType, name string, indexKey interface{}) string {
	var addr string
	if module != "" {
		addr = module + "." + resourceType + "." + name
	} else {
		addr = resourceType + "." + name
	}

	if indexKey != nil {
		switch k := indexKey.(type) {
		case float64:
			addr = fmt.Sprintf("%s[%d]", addr, int(k))
		case string:
			addr = fmt.Sprintf("%s[%q]", addr, k)
		}
	}

	return addr
}

func cleanProvider(provider string) string {
	// Raw format: provider["registry.terraform.io/hashicorp/aws"]
	// Clean to: registry.terraform.io/hashicorp/aws
	p := strings.TrimPrefix(provider, "provider[\"")
	p = strings.TrimSuffix(p, "\"]")
	return p
}
