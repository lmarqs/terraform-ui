package plan

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestActivate_GivenPolicySourceFailure_ShouldKeepReviewAndReportUnavailable(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return &sdk.PlanSummary{ToDelete: 1, Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "aws_db_instance.database"}, Action: sdk.ActionDelete, Risk: sdk.RiskCritical}}}, nil
	}}
	harness := sdktest.NewDeps(svc)
	harness.Deps.RiskPolicyDiagnostic = func() string { return "source_invalid" }
	p := New(svc).(*Plugin)
	p.Init(harness.Deps)
	p = drainActivate(t, p, Input{})
	defer p.Shutdown()
	if p.Status() != sdk.StatusDone || p.ExitCode() != 2 {
		t.Fatalf("Plan status = %v, exit = %d; want Done and 2", p.Status(), p.ExitCode())
	}
	stdout, err := p.Stdout()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stdout), "aws_db_instance.database") || strings.Contains(string(stdout), "Risk:") {
		t.Fatalf("Plan review output = %q", stdout)
	}
	if got := string(p.Stderr()); got != "risk unavailable: source_invalid\n" {
		t.Fatalf("Stderr = %q, want fixed policy diagnostic", got)
	}
	view := p.View(100, 30)
	if !strings.Contains(view, "aws_db_instance.database") || !strings.Contains(view, "risk unavailable: source_invalid") || strings.Contains(view, "CRITICAL") {
		t.Fatalf("Plan view = %q, want resource with unavailable risk", view)
	}
}

func TestActivate_GivenTerraformFailureAndPolicyFailure_ShouldPreserveTerraformError(t *testing.T) {
	svc := &sdktest.MockService{PlanFn: func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) {
		return nil, errors.New("terraform planning failed")
	}}
	harness := sdktest.NewDeps(svc)
	harness.Deps.RiskPolicyDiagnostic = func() string { return "source_invalid" }
	p := New(svc).(*Plugin)
	p.Init(harness.Deps)
	p = drainActivate(t, p, Input{})
	defer p.Shutdown()
	if p.Status() != sdk.StatusError || p.ExitCode() != 1 || string(p.Stderr()) != "terraform planning failed\n" {
		t.Fatalf("Terraform failure changed: status = %v, exit = %d, stderr = %q", p.Status(), p.ExitCode(), p.Stderr())
	}
}

func TestActivate_GivenPlanJSONAndPolicyFailure_ShouldSkipPolicyAndPreserveBytes(t *testing.T) {
	raw := []byte(" {\"resource_changes\": []}\n")
	svc := &sdktest.MockService{PlanJSONFn: func(context.Context, sdk.PlanOptions) ([]byte, error) { return raw, nil }}
	harness := sdktest.NewDeps(svc)
	checks := 0
	harness.Deps.RiskPolicyDiagnostic = func() string { checks++; return "source_unreadable" }
	p := New(svc).(*Plugin)
	p.Init(harness.Deps)
	p = drainActivate(t, p, Input{JSON: true})
	defer p.Shutdown()
	output, err := p.Stdout()
	if err != nil || string(output) != string(raw) || len(p.Stderr()) != 0 || checks != 0 {
		t.Fatalf("JSON pass-through changed: stdout = %q, err = %v, stderr = %q, policy checks = %d", output, err, p.Stderr(), checks)
	}
}
