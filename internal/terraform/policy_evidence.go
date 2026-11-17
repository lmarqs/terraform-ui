package terraform

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func policyEvidence(data []byte, doc map[string]any) *sdk.PolicyEvidence {
	evidence := &sdk.PolicyEvidence{
		PlanID:    fmt.Sprintf("sha256:%x", sha256.Sum256(data)),
		Resources: []sdk.PolicyResourceEvidence{},
	}
	fail := func(code string) *sdk.PolicyEvidence {
		evidence.ErrorCode = code
		evidence.Resources = []sdk.PolicyResourceEvidence{}
		return evidence
	}
	if value := doc["deferred_changes"]; value != nil {
		deferred, ok := value.([]any)
		if !ok {
			return fail("input_invalid")
		}
		if len(deferred) > 0 {
			return fail("input_incomplete")
		}
	}
	resources, _ := doc["resource_changes"].([]any)
	for _, value := range resources {
		resource, _ := value.(map[string]any)
		row, code := policyResourceEvidence(resource)
		if code != "" {
			return fail(code)
		}
		evidence.Resources = append(evidence.Resources, row)
	}
	outputs, _ := doc["output_changes"].(map[string]any)
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		change, _ := outputs[name].(map[string]any)
		actions, code := policyActions(change)
		if code != "" {
			return fail(code)
		}
		evidence.Resources = append(evidence.Resources, sdk.PolicyResourceEvidence{
			Address: "output." + name, Actions: actions, Exclusion: "output_only",
		})
	}
	return evidence
}

func policyActions(change map[string]any) ([]string, string) {
	values, _ := change["actions"].([]any)
	if len(values) == 0 {
		return nil, "input_incomplete"
	}
	actions := make([]string, len(values))
	for i, value := range values {
		actions[i], _ = value.(string)
	}
	if sdk.ActionFromStrings(actions) == sdk.ActionNoOp && (len(actions) != 1 || (actions[0] != "no-op" && !isForget(actions))) {
		return nil, "input_invalid"
	}
	return actions, ""
}

// ActionFromStrings maps forget to no-op, so forget is recognised by its raw action string.
func isForget(actions []string) bool {
	return len(actions) == 1 && actions[0] == "forget"
}

func policyResourceEvidence(resource map[string]any) (sdk.PolicyResourceEvidence, string) {
	address, _ := resource["address"].(string)
	row := sdk.PolicyResourceEvidence{Address: address}
	if address == "" {
		return row, "input_incomplete"
	}
	change, _ := resource["change"].(map[string]any)
	actions, code := policyActions(change)
	if code != "" {
		return row, code
	}
	row.Actions = actions
	previous, _ := resource["previous_address"].(string)
	action := sdk.ActionFromStrings(actions)
	if action == sdk.ActionNoOp || action == sdk.ActionRead {
		switch {
		case previous != "":
			row.Exclusion = "pure_move"
		case change["importing"] != nil:
			row.Exclusion = "pure_import"
		case isForget(actions):
			row.Exclusion = "forget"
		case action == sdk.ActionRead:
			row.Exclusion = "read"
		default:
			row.Exclusion = "no_op"
		}
		return row, ""
	}
	identity := sdk.PolicyResource{Address: address, Index: sdk.PolicyValue{Kind: "absent"}}
	if previous != "" {
		identity.PreviousAddress = &previous
	}
	identity.Mode, _ = resource["mode"].(string)
	identity.Type, _ = resource["type"].(string)
	identity.Name, _ = resource["name"].(string)
	identity.ProviderName, _ = resource["provider_name"].(string)
	if identity.Mode == "" || identity.Type == "" || identity.Name == "" || identity.ProviderName == "" {
		return row, "input_incomplete"
	}
	if identity.Mode != "managed" && identity.Mode != "data" {
		return row, "input_invalid"
	}
	if index, exists := resource["index"]; exists {
		if !policyPathElement(index) {
			return row, "input_invalid"
		}
		identity.Index, _ = projectPolicyValue(index, true, false, false)
	}
	input := &sdk.PolicyInput{SchemaVersion: "tfui.risk.input/v1", Resource: identity,
		Change: sdk.PolicyChange{Actions: actions, ReplacePaths: [][]any{}}}
	if importing := change["importing"]; importing != nil {
		metadata, _ := importing.(map[string]any)
		// Terraform supplies no sensitivity mask for a known import identifier.
		// Do not expose it, including when a separate unknown flag is present.
		if metadata["unknown"] != true || metadata["id"] != nil || metadata["identity"] != nil {
			return row, "input_incomplete"
		}
		input.Change.Importing = &sdk.PolicyImport{ID: sdk.PolicyValue{Kind: "unknown"}, Unknown: true}
	}
	if paths, exists := change["replace_paths"]; exists {
		values, ok := paths.([]any)
		if !ok {
			return row, "input_invalid"
		}
		for _, value := range values {
			path, ok := value.([]any)
			if !ok {
				return row, "input_invalid"
			}
			for _, element := range path {
				if !policyPathElement(element) {
					return row, "input_invalid"
				}
			}
			input.Change.ReplacePaths = append(input.Change.ReplacePaths, path)
		}
	}
	for _, side := range []string{"before", "after"} {
		value, exists := change[side]
		if !exists {
			return row, "input_incomplete"
		}
		unknown := any(false)
		if side == "after" {
			unknown = policyMask(change, "after_unknown")
		}
		sensitive := policyMask(change, side+"_sensitive")
		if !validPolicyMask(unknown) || !validPolicyMask(sensitive) {
			return row, "input_invalid"
		}
		node, valid := projectPolicyValue(value, true, unknown, sensitive)
		if !valid {
			return row, "input_invalid"
		}
		if value == nil && node.Kind == "null" && ((side == "before" && action == sdk.ActionCreate) || (side == "after" && action == sdk.ActionDelete)) {
			node.Kind = "absent"
		}
		if side == "before" {
			input.Change.Before = node
		} else {
			input.Change.After = node
		}
	}
	row.Input = input
	return row, ""
}

func policyPathElement(value any) bool {
	switch v := value.(type) {
	case string:
		return true
	case json.Number:
		number, ok := new(big.Rat).SetString(string(v))
		return ok && number.Sign() >= 0 && number.IsInt()
	default:
		return false
	}
}

func policyMask(object map[string]any, key string) any {
	if value, exists := object[key]; exists {
		return value
	}
	return false
}

func validPolicyMask(mask any) bool {
	switch m := mask.(type) {
	case bool:
		return true
	case map[string]any:
		for _, child := range m {
			if !validPolicyMask(child) {
				return false
			}
		}
		return true
	case []any:
		for _, child := range m {
			if !validPolicyMask(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func markedPolicyMask(mask any) bool {
	switch m := mask.(type) {
	case bool:
		return m
	case map[string]any:
		for _, child := range m {
			if markedPolicyMask(child) {
				return true
			}
		}
	case []any:
		for _, child := range m {
			if markedPolicyMask(child) {
				return true
			}
		}
	}
	return false
}

func projectPolicyValue(value any, exists bool, unknown, sensitive any) (sdk.PolicyValue, bool) {
	if unknown == true {
		if !validOpaquePolicyMask(value, sensitive) {
			return sdk.PolicyValue{}, false
		}
		kind := "unknown"
		if markedPolicyMask(sensitive) {
			kind = "sensitive_unknown"
		}
		return sdk.PolicyValue{Kind: kind}, true
	}
	if !exists {
		return sdk.PolicyValue{}, false
	}
	node := sdk.PolicyValue{}
	switch v := value.(type) {
	case map[string]any:
		u, uok := objectPolicyMask(unknown)
		s, sok := objectPolicyMask(sensitive)
		if !uok || !sok {
			return node, false
		}
		keys := make(map[string]bool)
		for _, object := range []map[string]any{v, u, s} {
			for key := range object {
				keys[key] = true
			}
		}
		entries := make(map[string]sdk.PolicyValue)
		for key := range keys {
			child, present := v[key]
			entry, valid := projectPolicyValue(child, present, policyMask(u, key), policyMask(s, key))
			if !valid {
				return node, false
			}
			entries[key] = entry
		}
		node = sdk.PolicyValue{Kind: "object", Entries: &entries}
	case []any:
		u, uok := arrayPolicyMask(unknown, len(v))
		s, sok := arrayPolicyMask(sensitive, len(v))
		if !uok || !sok {
			return node, false
		}
		items := make([]sdk.PolicyValue, len(v))
		for i, child := range v {
			entry, valid := projectPolicyValue(child, true, u[i], s[i])
			if !valid {
				return node, false
			}
			items[i] = entry
		}
		node = sdk.PolicyValue{Kind: "array", Items: &items}
	default:
		if !scalarPolicyMask(unknown) || !scalarPolicyMask(sensitive) {
			return node, false
		}
		switch v := value.(type) {
		case nil:
			node.Kind = "null"
		case bool:
			node.Kind, node.Value = "boolean", v
		case string:
			node.Kind, node.Value = "string", v
		case json.Number:
			node.Kind, node.Value = "number", string(v)
		}
	}
	if sensitive == true {
		return sdk.PolicyValue{Kind: "sensitive"}, true
	}
	return node, true
}

// Terraform marks a scalar with a bare bool, and writes an empty object or array
// where nothing under a null or scalar value is marked. Callers have already
// validated that every mask is a bool, object or array.
func scalarPolicyMask(mask any) bool {
	switch m := mask.(type) {
	case map[string]any:
		return len(m) == 0
	case []any:
		return len(m) == 0
	}
	return true
}

func objectPolicyMask(mask any) (map[string]any, bool) {
	if _, ok := mask.(bool); ok {
		return nil, true
	}
	object, ok := mask.(map[string]any)
	return object, ok
}

func arrayPolicyMask(mask any, length int) ([]any, bool) {
	if _, ok := mask.(bool); ok {
		values := make([]any, length)
		for i := range values {
			values[i] = false
		}
		return values, true
	}
	values, ok := mask.([]any)
	return values, ok && len(values) == length
}

// An unknown ancestor can still have concrete source container shapes. Validate
// those before redaction, while allowing masks where the source shape is unknown.
func validOpaquePolicyMask(value, mask any) bool {
	if value == nil {
		return true
	}
	switch m := mask.(type) {
	case map[string]any:
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		for key, child := range m {
			if !validOpaquePolicyMask(object[key], child) {
				return false
			}
		}
	case []any:
		array, ok := value.([]any)
		if !ok || len(array) != len(m) {
			return false
		}
		for i, child := range m {
			if !validOpaquePolicyMask(array[i], child) {
				return false
			}
		}
	}
	return true
}
