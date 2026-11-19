package riskpolicy

import (
	"context"
	"errors"
	"slices"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/topdown"
)

// Policy is one immutable snapshot compiled once. Compiling before Terraform runs
// lets a policy that cannot compile fail without acquiring a plan.
type Policy struct {
	snapshot Snapshot
	query    rego.PreparedEvalQuery
	err      error
}

// Compile prepares snapshot for evaluation. A Policy that failed to compile
// assesses every plan as unavailable with compilation_failed.
func Compile(ctx context.Context, snapshot Snapshot) Policy {
	query, err := prepare(ctx, snapshot)
	return Policy{snapshot: snapshot, query: query, err: err}
}

// Compiled reports whether the snapshot compiled.
func (p Policy) Compiled() bool { return p.err == nil }

// Snapshot returns the source the policy was compiled from.
func (p Policy) Snapshot() Snapshot { return p.snapshot }

// Assess compiles snapshot and assesses evidence with it.
func Assess(ctx context.Context, snapshot Snapshot, evidence sdk.PolicyEvidence) sdk.PolicyAssessment {
	return Compile(ctx, snapshot).Assess(ctx, evidence)
}

// Assess evaluates each eligible resource independently. It does not acquire
// plans, reload source or publish lifecycle events.
func (p Policy) Assess(ctx context.Context, evidence sdk.PolicyEvidence) (result sdk.PolicyAssessment) {
	snapshot := p.snapshot
	result = sdk.PolicyAssessment{
		Status: sdk.AssessmentUnavailable, PlanID: evidence.PlanID,
		Policy: sdk.PolicyIdentity{Kind: snapshot.Kind(), ID: snapshot.ID(), Digest: snapshot.Digest(), EvaluatorVersion: evaluatorVersion, CapabilityDigest: capabilityDigest},
	}
	defer func() {
		if ctx.Err() != nil {
			result.Status, result.ErrorCode = sdk.AssessmentCancelled, sdk.ErrorCancelled
			result.Resources, result.OverallLevel = nil, nil
		}
	}()
	if ctx.Err() != nil {
		return result
	}
	if evidence.ErrorCode != "" {
		result.ErrorCode = evidence.ErrorCode
		return result
	}
	for _, resource := range evidence.Resources {
		if resource.Exclusion == "" && resource.Input == nil {
			result.ErrorCode = sdk.ErrorInputIncomplete
			return result
		}
	}
	if p.err != nil {
		result.ErrorCode = sdk.ErrorCompilationFailed
		return result
	}
	classified, failed := 0, 0
	overall := sdk.RiskNone
	for _, resource := range evidence.Resources {
		row := sdk.PolicyResourceAssessment{Address: resource.Address, Deposed: resource.Deposed, Actions: slices.Clone(resource.Actions), Exclusion: resource.Exclusion}
		if resource.Exclusion != "" {
			row.Status = sdk.ResourceNotAssessed
		} else {
			row.Status = sdk.ResourceUnavailable
			level, reasons, code := evaluate(ctx, p.query, resource.Input)
			if ctx.Err() != nil {
				return result
			}
			row.ErrorCode = code
			if code != "" {
				failed++
			} else {
				row.Status, row.Level, row.Reasons = sdk.ResourceClassified, &level, reasons
				classified++
				if level > overall {
					overall = level
				}
			}
		}
		result.Resources = append(result.Resources, row)
	}
	switch {
	case failed > 0 && classified > 0:
		result.Status = sdk.AssessmentPartial
	case failed > 0:
		result.Status = sdk.AssessmentUnavailable
	case classified == 0:
		result.Status = sdk.AssessmentEmpty
	default:
		result.Status, result.OverallLevel = sdk.AssessmentComplete, &overall
	}
	return result
}

func prepare(ctx context.Context, snapshot Snapshot) (rego.PreparedEvalQuery, error) {
	options := []func(*rego.Rego){rego.Query("data.tfui.risk.v1.classification"), rego.Capabilities(capabilities()), rego.StrictBuiltinErrors(true), rego.EnablePrintStatements(false)}
	if len(snapshot.modules) == 0 {
		return rego.PreparedEvalQuery{}, sdk.ErrorCompilationFailed
	}
	for _, source := range snapshot.Modules() {
		module, err := ast.ParseModule(source.Name, source.Source)
		if err != nil {
			return rego.PreparedEvalQuery{}, err
		}
		if deniedOutput(module) {
			return rego.PreparedEvalQuery{}, sdk.ErrorCompilationFailed
		}
		options = append(options, rego.ParsedModule(module))
	}
	return rego.New(options...).PrepareForEval(ctx)
}

func evaluate(ctx context.Context, query rego.PreparedEvalQuery, input *sdk.PolicyInput) (sdk.RiskLevel, []sdk.PolicyReason, sdk.ErrorCode) {
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		var evaluationError *topdown.Error
		if errors.As(err, &evaluationError) && evaluationError.Code == topdown.ConflictErr {
			return 0, nil, sdk.ErrorEvaluationConflict
		}
		return 0, nil, sdk.ErrorEvaluationFailed
	}
	return classifyResults(results, input)
}

func classifyResults(results rego.ResultSet, input *sdk.PolicyInput) (sdk.RiskLevel, []sdk.PolicyReason, sdk.ErrorCode) {
	if len(results) == 0 {
		return 0, nil, sdk.ErrorResultUndefined
	}
	if len(results) != 1 || len(results[0].Expressions) != 1 {
		return 0, nil, sdk.ErrorResultCardinality
	}
	return validateResult(results[0].Expressions[0].Value, input)
}
