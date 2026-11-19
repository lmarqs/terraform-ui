// Package riskassess runs the host's risk assessment requests: it loads and
// compiles the selected policy, plans when the requester holds no summary,
// evaluates, caches and re-verifies the source, and tells which finished
// requests are still current.
package riskassess

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

// Evaluator assesses evidence under a compiled policy.
type Evaluator func(context.Context, riskpolicy.Policy, sdk.PolicyEvidence) sdk.PolicyAssessment

// Result is the message a request's Cmd produces. Dispatch Event only while
// Current reports the request current.
type Result struct {
	generation uint64
	Event      sdk.RiskAssessedEvent
}

// Coordinator is shared by App copies. Request, Cancel, Advance and Current
// run only on the Update loop; request Cmds run concurrently and share only the
// cache.
//
// Every request made since the last Advance shares one generation, and its
// context derives from that generation's context. Each requester holds at most
// one pending request context, which covers both the host-run plan and the
// policy evaluation.
type Coordinator struct {
	cfg   config.Config
	cache cache

	generation uint64
	ctx        context.Context
	cancel     context.CancelFunc
	issued     uint64
	requests   map[string]context.CancelFunc

	// assess is replaced only by this package's tests.
	assess Evaluator
}

// New returns a Coordinator reading policy selection and risk settings from cfg.
func New(cfg config.Config) *Coordinator {
	return &Coordinator{cfg: cfg, requests: map[string]context.CancelFunc{}, assess: assessPolicy}
}

func assessPolicy(ctx context.Context, policy riskpolicy.Policy, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
	return policy.Assess(ctx, evidence)
}

type cacheKey struct {
	current   *sdk.Context
	planID    string
	selection config.RiskPolicySelection
	digest    string
}

// cache holds one result; concurrent evaluations may finish in either order,
// but their captured Context identities never share a key.
type cache struct {
	mu     sync.Mutex
	key    cacheKey
	result sdk.PolicyAssessment
}

// Cancel ends requester's pending request. Its result is still dispatched, and
// the requester ignores it by AssessmentID.
func (c *Coordinator) Cancel(requester string) {
	if cancel, ok := c.requests[requester]; ok {
		cancel()
		delete(c.requests, requester)
	}
}

// Advance cancels every request issued so far and makes their results stale,
// for a Context change or a plan invalidation.
func (c *Coordinator) Advance() {
	for requester := range c.requests {
		c.Cancel(requester)
	}
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.generation++
}

// Current reports whether r belongs to a request issued since the last Advance.
func (c *Coordinator) Current(r Result) bool {
	return r.generation == c.generation
}

// Request assesses summary for requester under the policy current selects; a
// nil summary plans current first. It cancels requester's previous request.
func (c *Coordinator) Request(requester string, current *sdk.Context, summary *sdk.PlanSummary) (sdk.AssessmentID, tea.Cmd) {
	return c.RequestWith(requester, current, summary, c.assess)
}

// RequestWith is Request with assess as the evaluator.
func (c *Coordinator) RequestWith(requester string, current *sdk.Context, summary *sdk.PlanSummary, assess Evaluator) (sdk.AssessmentID, tea.Cmd) {
	if c.cancel == nil {
		c.ctx, c.cancel = context.WithCancel(context.Background())
	}
	c.Cancel(requester)
	ctx, cancel := context.WithCancel(c.ctx)
	c.requests[requester] = cancel
	c.issued++
	id, generation := sdk.AssessmentID(c.issued), c.generation
	return id, func() tea.Msg {
		defer cancel()
		event := c.run(ctx, current, summary, assess)
		event.Request = id
		return Result{generation: generation, Event: event}
	}
}

// run loads and compiles the policy before any plan, so a broken policy never runs Terraform.
func (c *Coordinator) run(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, assess Evaluator) sdk.RiskAssessedEvent {
	event := sdk.RiskAssessedEvent{Summary: summary}
	selection, snapshot, failure := c.loadPolicy(current)
	if failure != "" {
		event.Assessment = unavailable(failure)
		event.Hint = sdk.DiagnosticHint(sdk.RiskSettingsError(c.cfg.Plugins["risk"].Options))
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
	event.Assessment = c.evaluate(ctx, current, event.Summary, selection, policy, assess)
	return event
}

func planForAssessment(ctx context.Context, current *sdk.Context) (*sdk.PlanSummary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

func unavailable(code sdk.ErrorCode) sdk.PolicyAssessment {
	return sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: code}
}

// errorCode keeps a failure visible as a code even when err carries none.
func errorCode(err error, fallback sdk.ErrorCode) sdk.ErrorCode {
	var code sdk.ErrorCode
	if errors.As(err, &code) && code != "" {
		return code
	}
	return fallback
}

func (c *Coordinator) loadPolicy(current *sdk.Context) (config.RiskPolicySelection, riskpolicy.Snapshot, sdk.ErrorCode) {
	if err := sdk.RiskSettingsError(c.cfg.Plugins["risk"].Options); err != nil {
		return config.RiskPolicySelection{}, riskpolicy.Snapshot{}, sdk.ErrorConfigurationInvalid
	}
	selection, err := config.ResolveRiskPolicy(c.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return config.RiskPolicySelection{}, riskpolicy.Snapshot{}, errorCode(err, sdk.ErrorConfigurationInvalid)
	}
	snapshot, err := riskpolicy.LoadSource(selection, riskpolicy.BuiltinModules())
	if err != nil {
		return config.RiskPolicySelection{}, riskpolicy.Snapshot{}, errorCode(err, sdk.ErrorSourceUnreadable)
	}
	return selection, snapshot, ""
}

// evaluate publishes a result only when the source it was computed from is still the one selected.
func (c *Coordinator) evaluate(ctx context.Context, current *sdk.Context, summary *sdk.PlanSummary, selection config.RiskPolicySelection, policy riskpolicy.Policy, assess Evaluator) sdk.PolicyAssessment {
	snapshot := policy.Snapshot()
	evidence := sdk.PolicyEvidence{ErrorCode: sdk.ErrorInputIncomplete}
	if summary != nil && summary.PolicyEvidence != nil {
		evidence = *summary.PolicyEvidence
	}
	key := cacheKey{current: current, planID: evidence.PlanID, selection: selection, digest: snapshot.Digest()}
	if ctx.Err() == nil {
		c.cache.mu.Lock()
		cached := c.cache.key == key
		result := c.cache.result
		c.cache.mu.Unlock()
		if cached {
			return result
		}
	}
	result := assess(ctx, policy, evidence)
	if ctx.Err() != nil {
		return result
	}
	next, err := config.ResolveRiskPolicy(c.cfg.Dir, current.WorkingDir, current.Workspace.String())
	if err != nil {
		return unavailable(sdk.ErrorSourceChanged)
	}
	verified, err := riskpolicy.LoadSource(next, riskpolicy.BuiltinModules())
	if err != nil || next != selection || verified.Digest() != snapshot.Digest() {
		return unavailable(sdk.ErrorSourceChanged)
	}
	c.cache.mu.Lock()
	c.cache.key, c.cache.result = key, result
	c.cache.mu.Unlock()
	return result
}
