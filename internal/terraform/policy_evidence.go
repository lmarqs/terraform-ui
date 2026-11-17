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
	fail := func(code sdk.ErrorCode) *sdk.PolicyEvidence {
		evidence.ErrorCode = code
		evidence.Resources = []sdk.PolicyResourceEvidence{}
		return evidence
	}
	if value := doc["deferred_changes"]; value != nil {
		deferred, ok := value.([]any)
		if !ok {
			return fail(sdk.ErrorInputInvalid)
		}
		if len(deferred) > 0 {
			return fail(sdk.ErrorInputIncomplete)
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
			Address: "output." + name, Actions: actions, Exclusion: sdk.ExclusionOutputOnly,
		})
	}
	return evidence
}

func policyActions(change map[string]any) ([]string, sdk.ErrorCode) {
	values, _ := change["actions"].([]any)
	if len(values) == 0 {
		return nil, sdk.ErrorInputIncomplete
	}
	actions := make([]string, len(values))
	for i, value := range values {
		actions[i], _ = value.(string)
	}
	if sdk.ActionFromStrings(actions) == sdk.ActionNoOp && (len(actions) != 1 || (actions[0] != "no-op" && !isForget(actions))) {
		return nil, sdk.ErrorInputInvalid
	}
	return actions, ""
}

// ActionFromStrings maps forget to no-op, so forget is recognised by its raw action string.
func isForget(actions []string) bool {
	return len(actions) == 1 && actions[0] == "forget"
}

func policyResourceEvidence(resource map[string]any) (sdk.PolicyResourceEvidence, sdk.ErrorCode) {
	address, _ := resource["address"].(string)
	deposed, _ := resource["deposed"].(string)
	row := sdk.PolicyResourceEvidence{Address: address, Deposed: deposed}
	if address == "" {
		return row, sdk.ErrorInputIncomplete
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
			row.Exclusion = sdk.ExclusionPureMove
		case change["importing"] != nil:
			row.Exclusion = sdk.ExclusionPureImport
		case isForget(actions):
			row.Exclusion = sdk.ExclusionForget
		case action == sdk.ActionRead:
			row.Exclusion = sdk.ExclusionRead
		default:
			row.Exclusion = sdk.ExclusionNoOp
		}
		return row, ""
	}
	identity := sdk.PolicyResource{Address: address, Index: sdk.PolicyValue{Kind: sdk.KindAbsent}}
	if previous != "" {
		identity.PreviousAddress = &previous
	}
	if deposed != "" {
		identity.Deposed = &deposed
	}
	identity.Mode, _ = resource["mode"].(string)
	identity.Type, _ = resource["type"].(string)
	identity.Name, _ = resource["name"].(string)
	identity.ProviderName, _ = resource["provider_name"].(string)
	if identity.Mode == "" || identity.Type == "" || identity.Name == "" || identity.ProviderName == "" {
		return row, sdk.ErrorInputIncomplete
	}
	if identity.Mode != "managed" && identity.Mode != "data" {
		return row, sdk.ErrorInputInvalid
	}
	if index, exists := resource["index"]; exists {
		if !policyPathElement(index) {
			return row, sdk.ErrorInputInvalid
		}
		identity.Index, _ = projectPolicyValue(index, true, false, false)
	}
	input := &sdk.PolicyInput{SchemaVersion: "tfui.risk.input/v1", Resource: identity,
		Change: sdk.PolicyChange{Actions: actions, ReplacePaths: [][]any{}}}
	if importing := change["importing"]; importing != nil {
		metadata, _ := importing.(map[string]any)
		known := metadata["id"] != nil || metadata["identity"] != nil
		if known == (metadata["unknown"] == true) {
			return row, sdk.ErrorInputIncomplete
		}
		// Terraform supplies no sensitivity mask for a known import ID or identity, so neither reaches the policy.
		id := sdk.PolicyValue{Kind: sdk.KindUnknown}
		if known {
			id = sdk.PolicyValue{Kind: sdk.KindSensitive}
		}
		input.Change.Importing = &sdk.PolicyImport{ID: id, Unknown: !known}
	}
	if paths, exists := change["replace_paths"]; exists {
		values, ok := paths.([]any)
		if !ok {
			return row, sdk.ErrorInputInvalid
		}
		for _, value := range values {
			path, ok := value.([]any)
			if !ok {
				return row, sdk.ErrorInputInvalid
			}
			for _, element := range path {
				if !policyPathElement(element) {
					return row, sdk.ErrorInputInvalid
				}
			}
			input.Change.ReplacePaths = append(input.Change.ReplacePaths, path)
		}
	}
	for _, side := range []string{"before", "after"} {
		value, exists := change[side]
		if !exists {
			return row, sdk.ErrorInputIncomplete
		}
		unknown := any(false)
		if side == "after" {
			unknown = policyMask(change, "after_unknown")
		}
		sensitive := policyMask(change, side+"_sensitive")
		if !validPolicyMask(unknown) || !validPolicyMask(sensitive) {
			return row, sdk.ErrorInputInvalid
		}
		node, valid := projectPolicyValue(value, true, unknown, sensitive)
		if !valid {
			return row, sdk.ErrorInputInvalid
		}
		if value == nil && node.Kind == sdk.KindNull && ((side == "before" && action == sdk.ActionCreate) || (side == "after" && action == sdk.ActionDelete)) {
			node.Kind = sdk.KindAbsent
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

func projectPolicyValue(value any, exists bool, unknown, sensitive any) (sdk.PolicyValue, bool) {
	if unknown == true {
		if !validOpaquePolicyMask(value, sensitive) {
			return sdk.PolicyValue{}, false
		}
		kind := sdk.KindUnknown
		if markedValue(sensitive) {
			kind = sdk.KindSensitiveUnknown
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
		node = sdk.PolicyValue{Kind: sdk.KindObject, Entries: &entries}
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
		node = sdk.PolicyValue{Kind: sdk.KindArray, Items: &items}
	default:
		if !scalarPolicyMask(unknown) || !scalarPolicyMask(sensitive) {
			return node, false
		}
		switch v := value.(type) {
		case nil:
			node.Kind = sdk.KindNull
		case bool:
			node.Kind, node.Value = sdk.KindBoolean, v
		case string:
			node.Kind, node.Value = sdk.KindString, v
		case json.Number:
			node.Kind, node.Value = sdk.KindNumber, string(v)
		}
	}
	if sensitive == true {
		return sdk.PolicyValue{Kind: sdk.KindSensitive}, true
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
