package terraform

import (
	"encoding/json"
	"fmt"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// LoadPlan parses terraform plan JSON (output of `terraform show -json <planfile>`)
// and returns a PlanSummary with risk classification and phantom detection applied.
func LoadPlan(data []byte) (*PlanSummary, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parsing plan JSON: %w (hint: use `terraform show -json <planfile>` to convert binary plans)", err)
	}

	summary := ParsePlan(&plan)

	for i := range summary.Changes {
		summary.Changes[i].Risk = ClassifyRisk(&summary.Changes[i])
	}
	DetectPhantomChanges(summary.Changes)

	return summary, nil
}

// LoadState parses terraform state and returns resources and the parsed state.
// Supports two formats:
//   - Show JSON (output of `terraform show -json`): {"format_version": "1.0", "values": {...}}
//   - Raw tfstate (output of `terraform state pull`): {"version": 4, "resources": [...]}
func LoadState(data []byte) ([]sdk.Resource, *tfjson.State, error) {
	if isShowJSON(data) {
		return parseShowState(data)
	}
	return parseRawState(data)
}

func isShowJSON(data []byte) bool {
	var probe struct {
		FormatVersion string `json:"format_version"`
	}
	_ = json.Unmarshal(data, &probe)
	return probe.FormatVersion != ""
}

func parseShowState(data []byte) ([]sdk.Resource, *tfjson.State, error) {
	var state tfjson.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, nil, fmt.Errorf("parsing state JSON: %w", err)
	}

	if state.Values == nil {
		return []sdk.Resource{}, &state, nil
	}

	resources := ParseStateResources(state.Values.RootModule)
	return resources, &state, nil
}

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

func parseRawState(data []byte) ([]sdk.Resource, *tfjson.State, error) {
	var raw rawState
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("parsing state: %w", err)
	}

	if raw.Version == 0 {
		return nil, nil, fmt.Errorf("parsing state: not a valid terraform state file (no version field)")
	}

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
				Module:       ExtractModule(address),
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
			addr = fmt.Sprintf(`%s["%s"]`, addr, k)
		}
	}

	return addr
}

func cleanProvider(provider string) string {
	p := strings.TrimPrefix(provider, `provider["`)
	p = strings.TrimSuffix(p, `"]`)
	return p
}
