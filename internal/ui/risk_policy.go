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

type riskEvaluator func(context.Context, riskpolicy.Policy, sdk.PolicyEvidence) sdk.PolicyAssessment

// riskRequests is shared by App copies and touched only on the Update loop. Every
// request made since the last Context change or plan invalidation shares one
// generation and one cancellable context. Plans the host runs for those requests
// share a child context, which cancelling a plugin also ends.
//
// Requests are numbered in issue order. A cancel followed by a new request
// supersedes every request issued before that cancel: their late results would
// otherwise settle the new request, since a requester recognises its answer by
// pending state or Summary pointer.
type riskRequests struct {
	generation uint64
	ctx        context.Context
	cancel     context.CancelFunc
	planCtx    context.Context
	cancelPlan context.CancelFunc

	issued            uint64
	cancelledThrough  uint64
	supersededThrough uint64
}

type riskRequestID struct {
	generation uint64
	number     uint64
}

func (r *riskRequests) begin() (riskRequestID, context.Context) {
	if r.cancel == nil {
		r.ctx, r.cancel = context.WithCancel(context.Background())
	}
	r.supersededThrough = r.cancelledThrough
	r.issued++
	return riskRequestID{generation: r.generation, number: r.issued}, r.ctx
}

func (r *riskRequests) beginPlan() (riskRequestID, context.Context) {
	id, ctx := r.begin()
	if r.cancelPlan == nil {
		r.planCtx, r.cancelPlan = context.WithCancel(ctx)
	}
	return id, r.planCtx
}

func (r *riskRequests) cancelPlans() {
	r.cancelledThrough = r.issued
	if r.cancelPlan != nil {
		r.cancelPlan()
		r.cancelPlan = nil
	}
}

func (r *riskRequests) current(id riskRequestID) bool {
	return id.generation == r.generation && id.number > r.supersededThrough
}

func (r *riskRequests) advance() {
	r.cancelPlans()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	r.generation++
}

type riskAssessedMsg struct {
	request riskRequestID
	event   sdk.RiskAssessedEvent
}

// cancelPlugin cancels p's operation, including any plan the host runs on a
// plugin's behalf, which would otherwise keep Terraform and its state lock busy.
func (a App) cancelPlugin(p sdk.Plugin) {
	if c, ok := p.(sdk.Cancellable); ok {
		c.Cancel()
	}
	a.riskRequests.cancelPlans()
}

func (a App) requestAssessment(summary *sdk.PlanSummary) tea.Cmd {
	return a.requestAssessmentWith(summary, func(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
		return policy.Assess(ctx, evidence)
	})
}

// requestAssessmentWith pairs the request's identity with its Context, so a result
// for a superseded Context, plan or cancelled request never reaches plugins.
func (a App) requestAssessmentWith(summary *sdk.PlanSummary, assess riskEvaluator) tea.Cmd {
	current := a.holder.current
	begin := a.riskRequests.begin
	if summary == nil {
		begin = a.riskRequests.beginPlan
	}
	request, ctx := begin()
	return func() tea.Msg {
		return riskAssessedMsg{request: request, event: a.runAssessment(ctx, current, summary, assess)}
	}
}

// runAssessment loads and compiles the policy before any plan, so a broken policy never runs Terraform.
func (a App) runAssessment(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, assess riskEvaluator) sdk.RiskAssessedEvent {
	event := sdk.RiskAssessedEvent{Summary: summary}
	selection, snapshot, failure := a.loadRiskPolicy(current)
	if failure != "" {
		event.Assessment = unavailableAssessment(failure)
		event.Hint = sdk.DiagnosticHint(sdk.RiskSettingsError(a.cfg.Plugins["risk"].Options))
		return event
	}
	policy := riskpolicy.Compile(ctx, snapshot)
	if summary == nil {
		if !policy.Compiled() {
			event.Assessment = assess(ctx, policy, sdk.PolicyEvidence{})
			return event
		}
		event.Summary, event.Err = planForAssessment(ctx, current)
		if event.Err != nil {
			return event
		}
	}
	event.Assessment = a.evaluateRisk(ctx, current, event.Summary, selection, policy, assess)
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

func (a App) evaluateRisk(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, selection config.RiskPolicySelection, policy riskpolicy.Policy, assess riskEvaluator) sdk.PolicyAssessment {
	snapshot := policy.Snapshot()
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
	result := assess(ctx, policy, evidence)
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
