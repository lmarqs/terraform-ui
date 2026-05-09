// Package sdk provides the public API contract for tfui plugins. Plugins should
// import only this package to access terraform domain types, the Service interface,
// the Plugin interface, and shared styles.
package sdk

// RiskLevel classifies the risk severity of a planned infrastructure change,
// ranging from RiskNone (no risk) to RiskCritical (potentially destructive).
type RiskLevel int

const (
	RiskNone RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

// String returns the lowercase string representation of the risk level.
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

// Resource represents a terraform-managed resource identified by its address,
// type, logical name, module path, and provider.
type Resource struct {
	Address      string
	Type         string
	Name         string
	Module       string
	ProviderName string
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

// PlanChange represents a single resource change in a terraform plan, including
// the action to be taken, attribute-level diffs, computed risk, and phantom status.
type PlanChange struct {
	Resource       Resource
	Action         Action
	AttributeDiffs []AttributeDiff
	Risk           RiskLevel
	IsPhantom      bool
}

// PlanSummary holds the full set of resource changes from a terraform plan
// along with aggregate counts by action type.
type PlanSummary struct {
	Changes   []PlanChange
	ToCreate  int
	ToUpdate  int
	ToDelete  int
	ToReplace int
	ToRead    int
}

// ModuleGroup represents a set of plan changes that belong to the same terraform
// module, along with an action summary for quick aggregate display.
type ModuleGroup struct {
	Module  string
	Summary ActionSummary
	Changes []PlanChange
}

// ActionSummary holds counts of changes grouped by action type within a module.
type ActionSummary struct {
	Add     int
	Change  int
	Destroy int
	Replace int
}

// PhantomResult holds the outcome of phantom change detection, including counts
// and the addresses of resources identified as phantom (cosmetic-only) changes.
type PhantomResult struct {
	PhantomCount     int
	RealCount        int
	PhantomAddresses []string
}
