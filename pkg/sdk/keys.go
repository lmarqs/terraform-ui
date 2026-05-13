package sdk

// Well-known session keys for inter-plugin communication.
const (
	// SessionKeyPlanSummary holds the *PlanSummary from the last successful plan.
	SessionKeyPlanSummary = "plan.summary"
	// SessionKeyPlanFile holds the path (string) to the saved tfplan.out file.
	SessionKeyPlanFile = "plan.file"
	// SessionKeyResourceCount holds the total resource count (int) from the plan.
	SessionKeyResourceCount = "plan.resource_count"

	// SessionKeyActiveChdir holds the relative path (string) of the active chdir member.
	SessionKeyActiveChdir = "chdir.active"
	// SessionKeyActiveChdirAbs holds the absolute path (string) of the active chdir member.
	SessionKeyActiveChdirAbs = "chdir.active_abs"
	// SessionKeyChdirCount holds the number of configured chdir members (int).
	SessionKeyChdirCount = "chdir.count"

	// SessionKeyWorkspace holds the current terraform workspace name (string).
	SessionKeyWorkspace = "workspace.active"
)
