package terraform

import (
	"encoding/json"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
)

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
			Tainted:      r.Tainted,
		})
	}

	for _, child := range module.ChildModules {
		resources = append(resources, ParseStateResources(child)...)
	}

	return resources
}

// ShowResourceJSON renders a single resource from parsed state as indented,
// sensitive-redacted JSON. Both Service adapters route Show through this one
// function so the exec path and the recorded-command (macro) path can never
// diverge — previously each hand-rolled the display struct and the macro copy
// silently dropped the Tainted field (#46).
func ShowResourceJSON(state *tfjson.State, address string) (string, error) {
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
		Tainted      bool                   `json:"tainted,omitempty"`
		Values       map[string]interface{} `json:"values"`
	}{
		Address:      resource.Address,
		Type:         resource.Type,
		Name:         resource.Name,
		ProviderName: resource.ProviderName,
		Tainted:      resource.Tainted,
		Values:       redacted,
	}

	// json.MarshalIndent cannot fail here: the struct holds only strings and a
	// map[string]interface{} produced by json.Unmarshal (JSON-safe types).
	output, _ := json.MarshalIndent(display, "", "  ")
	return string(output), nil
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
