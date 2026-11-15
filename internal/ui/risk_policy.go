package ui

import (
	"context"
	"sync"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/internal/riskpolicy"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func (a App) riskPolicyDiagnostic() string {
	if err := sdk.RiskSettingsError(a.cfg.Plugins["risk"].Options); err != nil {
		return err.Error()
	}
	current := a.holder.current
	selection, err := config.ResolveRiskPolicy(a.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return err.Error()
	}
	// The evaluator owns embedded modules. Built-in selection needs no disk
	// acquisition; custom selection must never recover through that policy.
	if selection.Path == "" {
		return ""
	}
	if _, err := riskpolicy.LoadSource(selection, nil); err != nil {
		return err.Error()
	}
	return ""
}

type riskAssessmentKey struct {
	current   *sdk.Context
	planID    string
	selection config.RiskPolicySelection
	digest    string
}

// App copies share one bounded cache; concurrent superseded evaluations may finish
// in either order, but their captured Context identities never share a cache key.
type riskAssessmentCache struct {
	mu     sync.Mutex
	key    riskAssessmentKey
	result sdk.PolicyAssessment
}

// assessRisk uses only captured Context values while work runs off the UI thread.
func (a App) assessRisk(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary) sdk.PolicyAssessment {
	return a.assessRiskWith(ctx, current, summary, riskpolicy.Assess)
}

func (a App) assessRiskWith(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, assess func(context.Context, riskpolicy.Snapshot, sdk.PolicyEvidence) sdk.PolicyAssessment) sdk.PolicyAssessment {
	unavailable := func(code string) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: "unavailable", ErrorCode: code}
	}
	if err := sdk.RiskSettingsError(a.cfg.Plugins["risk"].Options); err != nil {
		return unavailable("configuration_invalid")
	}
	selection, err := config.ResolveRiskPolicy(a.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return unavailable(err.Error())
	}
	snapshot, err := riskpolicy.LoadSource(selection, riskpolicy.BuiltinModules())
	if err != nil {
		return unavailable(err.Error())
	}
	evidence := sdk.PolicyEvidence{ErrorCode: "input_incomplete"}
	if summary != nil && summary.PolicyEvidence != nil {
		evidence = *summary.PolicyEvidence
	}
	key := riskAssessmentKey{current: current, planID: evidence.PlanID, selection: selection, digest: snapshot.Digest()}
	if ctx.Err() == nil {
		a.riskCache.mu.Lock()
		cached := a.riskCache.key == key
		result := a.riskCache.result
		a.riskCache.mu.Unlock()
		if cached {
			return result
		}
	}
	result := assess(ctx, snapshot, evidence)
	if ctx.Err() != nil {
		return result
	}
	next, err := config.ResolveRiskPolicy(a.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return unavailable("source_changed")
	}
	verified, err := riskpolicy.LoadSource(next, riskpolicy.BuiltinModules())
	if err != nil || next != selection || verified.Digest() != snapshot.Digest() {
		return unavailable("source_changed")
	}
	a.riskCache.mu.Lock()
	a.riskCache.key, a.riskCache.result = key, result
	a.riskCache.mu.Unlock()
	return result
}
