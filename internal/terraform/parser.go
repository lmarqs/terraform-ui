package terraform

import "github.com/lmarqs/terraform-ui/pkg/sdk"

// Service is a type alias for the SDK Service interface. Internal packages and
// existing code can continue to reference terraform.Service. New code should
// prefer importing pkg/sdk directly.
type Service = sdk.Service

// Type aliases re-export SDK types so that internal packages can continue using
// the terraform package without breaking existing code. New code should prefer
// importing pkg/sdk directly.
type (
	RiskLevel     = sdk.RiskLevel
	Action        = sdk.Action
	Resource      = sdk.Resource
	AttributeDiff = sdk.AttributeDiff
	PlanChange    = sdk.PlanChange
	PlanSummary   = sdk.PlanSummary
	ModuleGroup   = sdk.ModuleGroup
	ActionSummary = sdk.ActionSummary
	PhantomResult = sdk.PhantomResult
)

// Re-export SDK constants.
const (
	RiskNone     = sdk.RiskNone
	RiskLow      = sdk.RiskLow
	RiskMedium   = sdk.RiskMedium
	RiskHigh     = sdk.RiskHigh
	RiskCritical = sdk.RiskCritical

	ActionCreate           = sdk.ActionCreate
	ActionRead             = sdk.ActionRead
	ActionUpdate           = sdk.ActionUpdate
	ActionDelete           = sdk.ActionDelete
	ActionDeleteThenCreate = sdk.ActionDeleteThenCreate
	ActionCreateThenDelete = sdk.ActionCreateThenDelete
	ActionNoOp             = sdk.ActionNoOp
)
