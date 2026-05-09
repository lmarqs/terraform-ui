package sdk

// Well-known session keys for inter-plugin communication.
const (
	// SessionKeyPlanSummary holds the *PlanSummary from the last successful plan.
	SessionKeyPlanSummary = "plan.summary"
	// SessionKeyPlanFile holds the path (string) to the saved tfplan.out file.
	SessionKeyPlanFile = "plan.file"
	// SessionKeyResourceCount holds the total resource count (int) from the plan.
	SessionKeyResourceCount = "plan.resource_count"

	// SessionKeyActiveContext holds the relative path (string) of the active context.
	SessionKeyActiveContext = "context.active"
	// SessionKeyActiveContextAbs holds the absolute path (string) of the active context.
	SessionKeyActiveContextAbs = "context.active_abs"
	// SessionKeyContextCount holds the number of discovered contexts (int).
	// A value > 1 indicates a multi-context (monorepo) environment.
	SessionKeyContextCount = "context.count"
)
