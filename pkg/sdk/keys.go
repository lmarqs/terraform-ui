package sdk

// Well-known session keys for inter-plugin communication.
const (
	// SessionKeyPlanSummary holds the *PlanSummary from the last successful plan.
	SessionKeyPlanSummary = "plan.summary"
	// SessionKeyPlanFile holds the path (string) to the saved tfplan.out file.
	SessionKeyPlanFile = "plan.file"
	// SessionKeyResourceCount holds the total resource count (int) from the plan.
	SessionKeyResourceCount = "plan.resource_count"
)
