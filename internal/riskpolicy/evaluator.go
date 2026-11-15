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

// Assess compiles one immutable snapshot once and evaluates each eligible resource
// independently. It does not acquire plans, reload source or publish lifecycle events.
func Assess(ctx context.Context, snapshot Snapshot, evidence sdk.PolicyEvidence) (result sdk.PolicyAssessment) {
	result = sdk.PolicyAssessment{
		Status: "unavailable", PlanID: evidence.PlanID,
		Policy: sdk.PolicyIdentity{Kind: snapshot.Kind(), ID: snapshot.ID(), Digest: snapshot.Digest(), EvaluatorVersion: evaluatorVersion, CapabilityDigest: capabilityDigest},
	}
	defer func() {
		if ctx.Err() != nil {
			result.Status, result.ErrorCode = "cancelled", "cancelled"
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
			result.ErrorCode = "input_incomplete"
			return result
		}
	}
	query, err := prepare(ctx, snapshot)
	if err != nil {
		result.ErrorCode = "compilation_failed"
		return result
	}
	classified, failed := 0, 0
	overall := sdk.RiskNone
	for _, resource := range evidence.Resources {
		row := sdk.PolicyResourceAssessment{Address: resource.Address, Actions: slices.Clone(resource.Actions), Exclusion: resource.Exclusion}
		if resource.Exclusion != "" {
			row.Status = "not_assessed"
		} else {
			row.Status = "unavailable"
			level, reasons, code := evaluate(ctx, query, resource.Input)
			if ctx.Err() != nil {
				return result
			}
			row.ErrorCode = code
			if code != "" {
				failed++
			} else {
				row.Status, row.Level, row.Reasons = "classified", &level, reasons
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
		result.Status = "partial"
	case failed > 0:
		result.Status = "unavailable"
	case classified == 0:
		result.Status = "empty"
	default:
		result.Status, result.OverallLevel = "complete", &overall
	}
	return result
}

func prepare(ctx context.Context, snapshot Snapshot) (rego.PreparedEvalQuery, error) {
	options := []func(*rego.Rego){rego.Query("data.tfui.risk.v1.classification"), rego.Capabilities(capabilities()), rego.StrictBuiltinErrors(true), rego.EnablePrintStatements(false)}
	if len(snapshot.modules) == 0 {
		return rego.PreparedEvalQuery{}, errors.New("source_invalid")
	}
	for _, source := range snapshot.Modules() {
		module, err := ast.ParseModule(source.Name, source.Source)
		if err != nil {
			return rego.PreparedEvalQuery{}, err
		}
		if deniedOutput(module) {
			return rego.PreparedEvalQuery{}, errors.New("compilation_failed")
		}
		options = append(options, rego.ParsedModule(module))
	}
	return rego.New(options...).PrepareForEval(ctx)
}

func evaluate(ctx context.Context, query rego.PreparedEvalQuery, input *sdk.PolicyInput) (sdk.RiskLevel, []sdk.PolicyReason, string) {
	results, err := query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		var evaluationError *topdown.Error
		if errors.As(err, &evaluationError) && evaluationError.Code == topdown.ConflictErr {
			return 0, nil, "evaluation_conflict"
		}
		return 0, nil, "evaluation_failed"
	}
	return classifyResults(results, input)
}

func classifyResults(results rego.ResultSet, input *sdk.PolicyInput) (sdk.RiskLevel, []sdk.PolicyReason, string) {
	if len(results) == 0 {
		return 0, nil, "result_undefined"
	}
	if len(results) != 1 || len(results[0].Expressions) != 1 {
		return 0, nil, "result_cardinality"
	}
	return validateResult(results[0].Expressions[0].Value, input)
}
