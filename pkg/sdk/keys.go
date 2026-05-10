package sdk

// Well-known session keys for inter-plugin communication.
const (
	// SessionKeyPlanSummary holds the *PlanSummary from the last successful plan.
	SessionKeyPlanSummary = "plan.summary"
	// SessionKeyPlanFile holds the path (string) to the saved tfplan.out file.
	SessionKeyPlanFile = "plan.file"
	// SessionKeyResourceCount holds the total resource count (int) from the plan.
	SessionKeyResourceCount = "plan.resource_count"

	// SessionKeyActiveScope holds the relative path (string) of the active scope.
	SessionKeyActiveScope = "scope.active"
	// SessionKeyActiveScopeAbs holds the absolute path (string) of the active scope.
	SessionKeyActiveScopeAbs = "scope.active_abs"
	// SessionKeyScopeCount holds the number of discovered scopes (int).
	// A value > 1 indicates a multi-scope (monorepo) environment.
	SessionKeyScopeCount = "scope.count"
)
