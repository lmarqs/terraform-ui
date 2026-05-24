package sdk

// Action represents the type of change terraform will make to a resource
// (create, update, delete, or replace variants).
type Action string

const (
	ActionCreate           Action = "create"
	ActionRead             Action = "read"
	ActionUpdate           Action = "update"
	ActionDelete           Action = "delete"
	ActionDeleteThenCreate Action = "delete-then-create"
	ActionCreateThenDelete Action = "create-then-delete"
	ActionNoOp             Action = "no-op"
)

// PlanChange represents a single resource change in a terraform plan, including
// the action to be taken, attribute-level diffs, computed risk, and phantom status.
type PlanChange struct {
	Resource       Resource
	Action         Action
	AttributeDiffs []AttributeDiff
	Risk           RiskLevel
	IsPhantom      bool
}

// AttributeDiff represents a change to a single resource attribute,
// capturing the old and new values along with sensitivity and force-new flags.
type AttributeDiff struct {
	Key       string
	OldValue  string
	NewValue  string
	Sensitive bool
	ForcesNew bool
}
