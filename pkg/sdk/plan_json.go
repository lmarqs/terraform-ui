package sdk

import (
	"encoding/json"
	"fmt"
)

// ActionFromStrings maps terraform's `change.actions` array onto the single
// Action tfui models. Combinations terraform does not define collapse to
// ActionNoOp: an action we cannot name is not a change we can describe.
func ActionFromStrings(actions []string) Action {
	switch len(actions) {
	case 1:
		switch Action(actions[0]) {
		case ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionNoOp:
			return Action(actions[0])
		}
	case 2:
		switch {
		case actions[0] == string(ActionDelete) && actions[1] == string(ActionCreate):
			return ActionDeleteThenCreate
		case actions[0] == string(ActionCreate) && actions[1] == string(ActionDelete):
			return ActionCreateThenDelete
		}
	}
	return ActionNoOp
}

// planJSONDocument is the minimal shape of `terraform show -json <planfile>`
// needed to count changes. The full document is terraform's schema and is
// passed through to callers verbatim (ADR-0006) — only the action of each
// resource change is read here.
type planJSONDocument struct {
	ResourceChanges []struct {
		Change *struct {
			Actions []string `json:"actions"`
		} `json:"change"`
	} `json:"resource_changes"`
}

// PlanJSONChangeCount counts the resource changes a plan summary would list in
// raw `terraform show -json` bytes: no-ops and data reads are not changes.
//
// The error must not be treated as "no changes" — an unreadable document means
// the count is unknown, which is a different answer from zero.
func PlanJSONChangeCount(data []byte) (int, error) {
	var doc planJSONDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, fmt.Errorf("reading plan JSON: %w", err)
	}

	count := 0
	for _, rc := range doc.ResourceChanges {
		if rc.Change == nil {
			continue
		}
		switch ActionFromStrings(rc.Change.Actions) {
		case ActionNoOp, ActionRead:
		default:
			count++
		}
	}
	return count, nil
}
