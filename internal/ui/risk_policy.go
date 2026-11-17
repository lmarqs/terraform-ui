package ui

import (
	"context"
	"errors"
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
		return string(errorCode(err))
	}
	// The evaluator owns embedded modules. Built-in selection needs no disk
	// acquisition; custom selection must never recover through that policy.
	if selection.Path == "" {
		return ""
	}
	if _, err := riskpolicy.LoadSource(selection, nil); err != nil {
		return string(errorCode(err))
	}
	return ""
}

func errorCode(err error) sdk.ErrorCode {
	var code sdk.ErrorCode
	errors.As(err, &code)
	return code
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
	unavailable := func(code sdk.ErrorCode) sdk.PolicyAssessment {
		return sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: code}
	}
	if err := sdk.RiskSettingsError(a.cfg.Plugins["risk"].Options); err != nil {
		return unavailable(sdk.ErrorConfigurationInvalid)
	}
	selection, err := config.ResolveRiskPolicy(a.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return unavailable(errorCode(err))
	}
	snapshot, err := riskpolicy.LoadSource(selection, riskpolicy.BuiltinModules())
	if err != nil {
		return unavailable(errorCode(err))
	}
	evidence := sdk.PolicyEvidence{ErrorCode: sdk.ErrorInputIncomplete}
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
		return unavailable(sdk.ErrorSourceChanged)
	}
	verified, err := riskpolicy.LoadSource(next, riskpolicy.BuiltinModules())
	if err != nil || next != selection || verified.Digest() != snapshot.Digest() {
		return unavailable(sdk.ErrorSourceChanged)
	}
	a.riskCache.mu.Lock()
	a.riskCache.key, a.riskCache.result = key, result
	a.riskCache.mu.Unlock()
	return result
}
