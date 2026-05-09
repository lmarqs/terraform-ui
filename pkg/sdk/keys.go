package sdk

// Well-known session keys for inter-plugin communication.
const (
	// SessionKeyPlanSummary holds the *PlanSummary from the last successful plan.
	SessionKeyPlanSummary = "plan.summary"
	// SessionKeyPlanFile holds the path (string) to the saved tfplan.out file.
	SessionKeyPlanFile = "plan.file"
	// SessionKeyResourceCount holds the total resource count (int) from the plan.
	SessionKeyResourceCount = "plan.resource_count"

	// SessionKeyActiveProject holds the relative path (string) of the active project.
	SessionKeyActiveProject = "project.active"
	// SessionKeyActiveProjectAbs holds the absolute path (string) of the active project.
	SessionKeyActiveProjectAbs = "project.active_abs"
	// SessionKeyProjectCount holds the number of discovered projects (int).
	// A value > 1 indicates a multi-project (monorepo) environment.
	SessionKeyProjectCount = "project.count"
)
