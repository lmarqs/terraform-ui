package terraform

import (
	"encoding/json"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
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
