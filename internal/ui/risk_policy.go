package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

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
		return string(errorCode(err, sdk.ErrorConfigurationInvalid))
	}
	// The evaluator owns embedded modules. Built-in selection needs no disk
	// acquisition; custom selection must never recover through that policy.
	if selection.Path == "" {
		return ""
	}
	if _, err := riskpolicy.LoadSource(selection, nil); err != nil {
		return string(errorCode(err, sdk.ErrorSourceUnreadable))
	}
	return ""
}

// errorCode keeps a failure visible as a code even when err carries none.
func errorCode(err error, fallback sdk.ErrorCode) sdk.ErrorCode {
	var code sdk.ErrorCode
	if errors.As(err, &code) && code != "" {
		return code
	}
	return fallback
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

type riskEvaluator func(context.Context, riskpolicy.Snapshot, sdk.PolicyEvidence) sdk.PolicyAssessment

// riskRequests is shared by App copies and touched only on the Update loop. Every
// request made since the last Context change or plan invalidation shares one
// generation and one cancellable context.
type riskRequests struct {
	generation uint64
	ctx        context.Context
	cancel     context.CancelFunc
}

func (r *riskRequests) begin() (uint64, context.Context) {
	if r.cancel == nil {
		r.ctx, r.cancel = context.WithCancel(context.Background())
	}
	return r.generation, r.ctx
}

func (r *riskRequests) advance() {
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.generation++
}

type riskAssessedMsg struct {
	generation uint64
	event      sdk.RiskAssessedEvent
}

func (a App) requestAssessment(summary *sdk.PlanSummary) tea.Cmd {
	return a.requestAssessmentWith(summary, riskpolicy.Assess)
}

// requestAssessmentWith pairs the generation with the Context of the request, so a
// result for a superseded Context or plan never reaches plugins.
func (a App) requestAssessmentWith(summary *sdk.PlanSummary, assess riskEvaluator) tea.Cmd {
	current := a.holder.current
	generation, ctx := a.riskRequests.begin()
	return func() tea.Msg {
		return riskAssessedMsg{generation: generation, event: a.runAssessment(ctx, current, summary, assess)}
	}
}

// runAssessment loads the policy before any plan, so a broken policy never runs Terraform.
func (a App) runAssessment(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, assess riskEvaluator) sdk.RiskAssessedEvent {
	event := sdk.RiskAssessedEvent{Summary: summary}
	selection, snapshot, failure := a.loadRiskPolicy(current)
	if failure != "" {
		event.Assessment = unavailableAssessment(failure)
		event.Hint = sdk.DiagnosticHint(sdk.RiskSettingsError(a.cfg.Plugins["risk"].Options))
		return event
	}
	if summary == nil {
		event.Summary, event.Err = planForAssessment(ctx, current)
		if event.Err != nil {
			return event
		}
	}
	event.Assessment = a.evaluateRisk(ctx, current, event.Summary, selection, snapshot, assess)
	return event
}

func planForAssessment(ctx context.Context, current *sdk.Context) (*sdk.PlanSummary, error) {
	dir, err := os.MkdirTemp("", "tfui-risk-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	opts := current.PlanOptions()
	opts.PlanFile = filepath.Join(dir, "plan.tfplan")
	summary, err := current.Service.Plan(ctx, opts)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return summary, err
}

func unavailableAssessment(code sdk.ErrorCode) sdk.PolicyAssessment {
	return sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: code}
}

// assessRisk uses only captured Context values while work runs off the UI thread.
func (a App) assessRisk(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary) sdk.PolicyAssessment {
	return a.assessRiskWith(ctx, current, summary, riskpolicy.Assess)
}

func (a App) assessRiskWith(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, assess riskEvaluator) sdk.PolicyAssessment {
	selection, snapshot, failure := a.loadRiskPolicy(current)
	if failure != "" {
		return unavailableAssessment(failure)
	}
	return a.evaluateRisk(ctx, current, summary, selection, snapshot, assess)
}

func (a App) loadRiskPolicy(current *sdk.Context) (config.RiskPolicySelection, riskpolicy.Snapshot, sdk.ErrorCode) {
	if err := sdk.RiskSettingsError(a.cfg.Plugins["risk"].Options); err != nil {
		return config.RiskPolicySelection{}, riskpolicy.Snapshot{}, sdk.ErrorConfigurationInvalid
	}
	selection, err := config.ResolveRiskPolicy(a.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return config.RiskPolicySelection{}, riskpolicy.Snapshot{}, errorCode(err, sdk.ErrorConfigurationInvalid)
	}
	snapshot, err := riskpolicy.LoadSource(selection, riskpolicy.BuiltinModules())
	if err != nil {
		return config.RiskPolicySelection{}, riskpolicy.Snapshot{}, errorCode(err, sdk.ErrorSourceUnreadable)
	}
	return selection, snapshot, ""
}

func (a App) evaluateRisk(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, selection config.RiskPolicySelection, snapshot riskpolicy.Snapshot, assess riskEvaluator) sdk.PolicyAssessment {
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
		return unavailableAssessment(sdk.ErrorSourceChanged)
	}
	verified, err := riskpolicy.LoadSource(next, riskpolicy.BuiltinModules())
	if err != nil || next != selection || verified.Digest() != snapshot.Digest() {
		return unavailableAssessment(sdk.ErrorSourceChanged)
	}
	a.riskCache.mu.Lock()
	a.riskCache.key, a.riskCache.result = key, result
	a.riskCache.mu.Unlock()
	return result
}
