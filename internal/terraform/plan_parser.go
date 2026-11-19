package terraform

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// ParsePlan converts a tfjson.Plan into a PlanSummary.
func ParsePlan(plan *tfjson.Plan) *PlanSummary {
	summary := &PlanSummary{
		Changes: make([]PlanChange, 0),
	}

	if plan == nil || plan.ResourceChanges == nil {
		return summary
	}

	for _, rc := range plan.ResourceChanges {
		if rc == nil || rc.Change == nil {
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
				Deposed:      rc.DeposedKey,
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

// mapActions delegates to the SDK so the plan-file path and the `-json`
// passthrough path agree on which actions count as a change.
func mapActions(actions tfjson.Actions) Action {
	strs := make([]string, len(actions))
	for i, a := range actions {
		strs[i] = string(a)
	}
	return sdk.ActionFromStrings(strs)
}

func parseAttributeDiffs(change *tfjson.Change) []AttributeDiff {
	diffs := make([]AttributeDiff, 0)

	if change.Before == nil && change.After == nil {
		return diffs
	}

	beforeMap := jsonToMap(change.Before)
	afterMap := jsonToMap(change.After)

	keys := make(map[string]bool)
	for k := range beforeMap {
		keys[k] = true
	}
	for k := range afterMap {
		keys[k] = true
	}

	unknownMap := jsonToMap(change.AfterUnknown)
	for k := range unknownMap {
		keys[k] = true
	}
	sortedKeys := make([]string, 0, len(keys))
	for key := range keys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)
	for _, key := range sortedKeys {
		oldVal := marshalValue(beforeMap[key])
		newVal := marshalValue(afterMap[key])
		if markedValue(unknownMap[key]) {
			newVal = marshalWithUnknown(afterMap[key], unknownMap[key])
		}

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
			ForcesNew: attributeForcesNew(change.ReplacePaths, key),
		})
	}

	return diffs
}

func jsonToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return make(map[string]interface{})
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return make(map[string]interface{})
}

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

// unknownLeaf marks an unknown leaf until marshalling, after which its quoted form is
// replaced with the bare notation so it cannot be mistaken for a string value.
const unknownLeaf = "\x00known after apply\x00"

func marshalWithUnknown(value, unknown interface{}) string {
	return strings.ReplaceAll(marshalValue(withUnknownLeaves(value, unknown)), `"\u0000known after apply\u0000"`, "(known after apply)")
}

// withUnknownLeaves adds each unknown leaf of the after_unknown mask to the known after
// value: Terraform omits an unknown object key from after and nulls an unknown list element.
func withUnknownLeaves(value, unknown interface{}) interface{} {
	switch mask := unknown.(type) {
	case map[string]interface{}:
		object, _ := value.(map[string]interface{})
		filled := make(map[string]interface{}, len(object))
		for key, child := range object {
			filled[key] = child
		}
		for key, child := range mask {
			if markedValue(child) {
				filled[key] = withUnknownLeaves(object[key], child)
			}
		}
		return filled
	case []interface{}:
		list, _ := value.([]interface{})
		filled := make([]interface{}, max(len(list), len(mask)))
		copy(filled, list)
		for i, child := range mask {
			if markedValue(child) {
				filled[i] = withUnknownLeaves(filled[i], child)
			}
		}
		return filled
	}
	return unknownLeaf
}

func isKeySensitive(sensitive interface{}, key string) bool {
	return maskMarked(sensitive, key)
}

func maskMarked(mask interface{}, key string) bool {
	if mask == nil {
		return false
	}
	switch m := mask.(type) {
	case bool:
		return m
	case map[string]interface{}:
		return markedValue(m[key])
	}
	return false
}

func markedValue(mask interface{}) bool {
	switch value := mask.(type) {
	case bool:
		return value
	case map[string]interface{}:
		for _, child := range value {
			if markedValue(child) {
				return true
			}
		}
	case []interface{}:
		for _, child := range value {
			if markedValue(child) {
				return true
			}
		}
	}
	return false
}

func attributeForcesNew(paths []interface{}, key string) bool {
	for _, value := range paths {
		path, ok := value.([]interface{})
		if !ok {
			continue
		}
		if len(path) > 0 && path[0] == key {
			return true
		}
	}
	return false
}
