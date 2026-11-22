package riskassess

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestRequestPlan_GivenBrokenPolicy_ShouldPlanWithTheEffectiveTargetsAndPublishTheSummary(t *testing.T) {
	configured := []string{"module.networking"}
	tests := []struct {
		name string
		pins sdk.Pins
		want []string
	}{
		{name: "no pins plans the configured targets", want: configured},
		{name: "pins replace the configured targets", pins: sdk.Pins{"aws_instance.web"}, want: []string{"aws_instance.web"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writePolicyFile(t, root, "tfui.hcl", `defaults { risk_policy = "missing.rego" }`)
			planned := &sdk.PlanSummary{}
			var artifact string
			svc := &sdktest.MockService{}
			svc.PlanFn = func(_ context.Context, opts sdk.PlanOptions) (*sdk.PlanSummary, error) {
				artifact = opts.PlanFile
				if err := os.WriteFile(artifact, []byte("saved plan"), 0600); err != nil {
					t.Fatal(err)
				}
				return planned, nil
			}
			h := newHarness(t, root, svc)
			h.coord.cfg.Plugins = map[string]config.PluginConfig{"plan": {Options: map[string]interface{}{"targets": configured}}}
			h.current = h.current.WithPins(tt.pins)

			id, cmd := h.coord.RequestPlan(requester, h.current)
			result := cmd().(PlanResult)

			if !h.coord.PlanCurrent(result) {
				t.Fatal("a result with no Advance since its request was reported stale")
			}
			if ev := result.Event; id == 0 || ev.Request != id || ev.Summary != planned || ev.Err != nil {
				t.Fatalf("event = %+v, want the plan's summary under the nonzero ID %d RequestPlan returned", ev, id)
			}
			if len(svc.PlanCalls) != 1 {
				t.Fatalf("Service.Plan called %d times, want 1", len(svc.PlanCalls))
			}
			if got := svc.PlanCalls[0].Targets; !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("plan targets = %q, want %q", got, tt.want)
			}
			if _, err := os.Stat(artifact); artifact == "" || !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("temporary artifact %q: %v; want one supplied and removed", artifact, err)
			}
		})
	}
}

func TestRequestPlan_GivenPlanFailure_ShouldPublishTerraformsError(t *testing.T) {
	failure := errors.New("terraform plan: no configuration files")
	svc := &sdktest.MockService{}
	svc.PlanFn = func(context.Context, sdk.PlanOptions) (*sdk.PlanSummary, error) { return nil, failure }
	h := newHarness(t, t.TempDir(), svc)

	_, cmd := h.coord.RequestPlan(requester, h.current)

	if ev := cmd().(PlanResult).Event; ev.Summary != nil || !errors.Is(ev.Err, failure) {
		t.Fatalf("event = %+v, want terraform's error as is", ev)
	}
}

func TestRequestPlan_GivenCancelOrAdvanceAfterRequest_ShouldCancelThePlanAndOnlyAdvanceMakesItStale(t *testing.T) {
	tests := []struct {
		name      string
		end       func(*Coordinator)
		wantStale bool
	}{
		{name: "cancel", end: func(c *Coordinator) { c.Cancel(requester) }},
		{name: "a second request", end: func(c *Coordinator) { c.RequestPlan(requester, &sdk.Context{}) }},
		{name: "advance", end: (*Coordinator).Advance, wantStale: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &sdktest.MockService{}
			h := newHarness(t, t.TempDir(), svc)
			_, cmd := h.coord.RequestPlan(requester, h.current)

			tt.end(h.coord)
			result := cmd().(PlanResult)

			if !errors.Is(result.Event.Err, context.Canceled) || len(svc.PlanCalls) != 0 {
				t.Fatalf("err=%v Service.Plan=%d; want the plan cancelled before Terraform runs", result.Event.Err, len(svc.PlanCalls))
			}
			if stale := !h.coord.PlanCurrent(result); stale != tt.wantStale {
				t.Fatalf("stale = %v, want %v", stale, tt.wantStale)
			}
		})
	}
}
