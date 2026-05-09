package terraform

// RiskLevel classifies the risk of a planned change.
type RiskLevel int

const (
	RiskNone RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "none"
	}
}

// Action represents the type of change terraform will make.
type Action string

const (
	ActionCreate            Action = "create"
	ActionRead              Action = "read"
	ActionUpdate            Action = "update"
	ActionDelete            Action = "delete"
	ActionDeleteThenCreate  Action = "delete-then-create"
	ActionCreateThenDelete  Action = "create-then-delete"
	ActionNoOp              Action = "no-op"
)

// Resource represents a terraform resource in state.
type Resource struct {
	Address      string
	Type         string
	Name         string
	Module       string
	ProviderName string
}

// AttributeDiff represents a change to a single attribute.
type AttributeDiff struct {
	Key      string
	OldValue string
	NewValue string
	Sensitive bool
	ForcesNew bool
}

// PlanChange represents a single resource change in a plan.
type PlanChange struct {
	Resource       Resource
	Action         Action
	AttributeDiffs []AttributeDiff
	Risk           RiskLevel
	IsPhantom      bool
}

// PlanSummary holds aggregate information about a plan.
type PlanSummary struct {
	Changes    []PlanChange
	ToCreate   int
	ToUpdate   int
	ToDelete   int
	ToReplace  int
	ToRead     int
}
