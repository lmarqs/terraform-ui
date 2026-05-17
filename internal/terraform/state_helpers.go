package terraform

import (
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
