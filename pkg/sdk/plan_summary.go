package sdk

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
