package plan

import (
	"context"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestActivate_GivenCachedAssessmentThenInvalidation_ShouldRejectLateClassification(t *testing.T) {
	for _, action := range []string{"cancel", "invalidate", "context", "pins"} {
		t.Run(action, func(t *testing.T) {
			low, critical := sdk.RiskLow, sdk.RiskCritical
			level := &low
			svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
				return &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionDelete}}}, nil
			}}
			h := sdktest.NewDeps(svc)
			h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
				return sdk.PolicyAssessment{Status: "complete", OverallLevel: level, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: level}}}
			}
			p := New(svc).(*Plugin)
			p.Init(h.Deps)
			defer p.Shutdown()
			drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
			level = &critical
			pending := p.Activate(Input{})
			if pending == nil {
				t.Fatal("cached activation should reassess selected source")
			}
			late := pending()
			switch action {
			case "cancel":
				p.Cancel()
			case "invalidate":
				p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})
			case "pins":
				p.HandleContextChanged(sdk.ContextChangedEvent{Next: h.Ctx, Reason: sdk.ContextPinsChanged})
			case "context":
				h.Ctx = &sdk.Context{WorkingDir: "/different", Service: svc}
				p.HandleContextChanged(sdk.ContextChangedEvent{Next: h.Ctx})
			}
			p.Update(late)
			if view := p.View(100, 30); strings.Contains(view, "CRITICAL") {
				t.Fatalf("late classification replaced invalidated review: %q", view)
			}
		})
	}
}

func TestActivate_GivenRecoveredPolicy_ShouldClearFailureAndUpdateOpenInspection(t *testing.T) {
	high := sdk.RiskHigh
	assessment := sdk.PolicyAssessment{Status: "unavailable", ErrorCode: "compilation_failed"}
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionUpdate}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment { return assessment }
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	p.inspectSelected()
	if view := p.View(120, 35); !strings.Contains(view, "compilation_failed") {
		t.Fatalf("initial inspection=%q; want compilation failure", view)
	}
	assessment = sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high, Reasons: []sdk.PolicyReason{{Code: "recovered", Message: "Current policy evidence"}}}}}
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	view := p.View(120, 35)
	if !strings.Contains(view, "HIGH") || !strings.Contains(view, "Current policy evidence") || strings.Contains(view, "unavailable") || len(p.Stderr()) != 0 {
		t.Fatalf("recovered inspection=%q stderr=%q; want current classification with cleared failure", view, p.Stderr())
	}
	if len(svc.PlanCalls) != 1 {
		t.Fatalf("source recovery planned %d times; want retained plan evidence", len(svc.PlanCalls))
	}
}

func TestActivate_GivenMissingAssessmentRow_ShouldShowInputIncompleteOnInspection(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.omitted"}, Action: sdk.ActionUpdate, Risk: sdk.RiskCritical}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "partial"}
	}
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	p.inspectSelected()
	if view := p.View(100, 30); !strings.Contains(view, "input_incomplete") || strings.Contains(view, "CRITICAL") {
		t.Fatalf("omitted resource inspection=%q; want unavailable input", view)
	}
}

func TestCancel_GivenRunningPlan_ShouldTerminateAndRejectLateSuccess(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.cancelled"}, Action: sdk.ActionUpdate}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	pending := p.Activate(Input{})
	p.Cancel()
	if p.Busy() {
		t.Fatal("cancelled Plan remains busy; headless session cannot terminate")
	}
	drainAssessmentPlan(t, p, pending, nil)
	if view := p.View(100, 30); strings.Contains(view, "aws_instance.cancelled") {
		t.Fatalf("cancelled Plan accepted late success: %q", view)
	}
	if p.Status() != sdk.StatusError || p.ExitCode() != 1 {
		t.Fatalf("cancelled Plan status=%v exit=%d; want terminal failure", p.Status(), p.ExitCode())
	}
}

func TestActivate_GivenCachedPlanReassessment_ShouldHideOldRiskUntilCurrentResult(t *testing.T) {
	high := sdk.RiskHigh
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionUpdate}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high}}}
	}
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	pending := p.Activate(Input{})
	if !p.Busy() || p.Ready() {
		t.Fatalf("reassessment status=%v ready=%v; must wait for current assessment", p.Status(), p.Ready())
	}
	if view := p.View(100, 30); strings.Contains(view, "HIGH") {
		t.Fatalf("reassessment exposes obsolete high classification: %q", view)
	}
	drainAssessmentPlan(t, p, pending, nil)
	if !p.Ready() || p.Busy() {
		t.Fatalf("completed reassessment status=%v; want ready", p.Status())
	}
}

func TestActivate_GivenPartialAssessment_ShouldReportFirstFailureByAddress(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToUpdate: 2, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.z"}, Action: sdk.ActionUpdate}, {Resource: sdk.Resource{Address: "aws_instance.a"}, Action: sdk.ActionUpdate}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "partial", Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.z", Status: "unavailable", ErrorCode: "evaluation_failed"}, {Address: "aws_instance.a", Status: "unavailable", ErrorCode: "result_invalid"}}}
	}
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	if got := string(p.Stderr()); got != "risk unavailable: result_invalid\n" {
		t.Fatalf("stderr=%q; want fixed diagnostic of first failed address", got)
	}
}

func TestActivate_GivenPinsBeforeFirstPlan_ShouldAcquireReviewedPlan(t *testing.T) {
	high := sdk.RiskHigh
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToUpdate: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_instance.reviewed"}, Action: sdk.ActionUpdate}}}, nil
	}}
	h := sdktest.NewDeps(svc)
	h.Deps.AssessRisk = func(context.Context, *sdk.Context, *sdk.PlanSummary) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "complete", OverallLevel: &high, Resources: []sdk.PolicyResourceAssessment{{Address: "aws_instance.reviewed", Status: "classified", Level: &high}}}
	}
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	defer p.Shutdown()
	p.HandleContextChanged(sdk.ContextChangedEvent{Next: h.Ctx, Reason: sdk.ContextPinsChanged})
	drainAssessmentPlan(t, p, p.Activate(Input{}), nil)
	if len(svc.PlanCalls) != 1 || !strings.Contains(p.View(100, 30), "aws_instance.reviewed") {
		t.Fatalf("initial selected plan calls=%d view=%q; want acquisition before assessment", len(svc.PlanCalls), p.View(100, 30))
	}
}
