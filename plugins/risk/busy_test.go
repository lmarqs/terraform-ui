package risk

import (
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// Busy means Terraform runs for Risk: only a request without a summary makes the
// host run `terraform plan`. A request for a retained summary is policy-only and
// holds no directory lock, yet it still keeps Ready false so headless output waits.
func TestBusy_GivenPendingRequest_ShouldBeBusyOnlyWhenHostRunsTerraformPlan(t *testing.T) {
	for _, tc := range []struct {
		name     string
		retained bool
		wantBusy bool
	}{
		{"policy-only request for a retained summary", true, false},
		{"host-run plan for a nil-summary request", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p *Plugin
			if tc.retained {
				summary := lifecycleSummary("aws_instance.reviewed")
				p, _, _ = newAssessedPlugin(t, summary, classifiedAs("aws_instance.reviewed", sdk.RiskLow, "Reviewed."))
			} else {
				p, _ = newLifecyclePlugin(t, &sdktest.MockService{})
			}

			p.Activate(Input{})

			if p.Status() != sdk.StatusLoading || p.Ready() {
				t.Fatalf("setup: status=%v ready=%v; want a pending request that is not ready", p.Status(), p.Ready())
			}
			if p.Busy() != tc.wantBusy {
				t.Fatalf("Busy() = %v while the request is pending, want %v", p.Busy(), tc.wantBusy)
			}
		})
	}
}
