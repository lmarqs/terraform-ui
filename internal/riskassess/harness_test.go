package riskassess

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

// harness drives a Coordinator as the App does: requests capture the active
// Context, a Context switch advances, and only current results are dispatched.
type harness struct {
	coord   *Coordinator
	root    string
	svc     *sdktest.MockService
	current *sdk.Context
	events  []sdk.RiskAssessedEvent
}

func newHarness(t *testing.T, root string, svc *sdktest.MockService) *harness {
	t.Helper()
	h := &harness{coord: New(config.Config{Dir: root}), root: root, svc: svc}
	h.current = h.contextFor("", "default")
	return h
}

func (h *harness) contextFor(chdir, workspace string) *sdk.Context {
	return &sdk.Context{Chdir: sdk.Chdir(chdir), WorkingDir: filepath.Join(h.root, chdir), Workspace: sdk.NewWorkspace(workspace), Service: h.svc}
}

func (h *harness) switchContext(chdir, workspace string) {
	h.current = h.contextFor(chdir, workspace)
	h.coord.Advance()
}

// requester is the plugin a harness request is made for unless a test names another.
const requester = "requester"

func (h *harness) request(summary *sdk.PlanSummary) tea.Cmd {
	return h.requestAs(requester, summary)
}

func (h *harness) requestAs(plugin string, summary *sdk.PlanSummary) tea.Cmd {
	_, cmd := h.coord.Request(plugin, h.current, summary)
	return cmd
}

func (h *harness) requestWith(summary *sdk.PlanSummary, assess Evaluator) tea.Cmd {
	h.coord.assess = assess
	return h.request(summary)
}

// deliver runs cmd off the Update loop, as bubbletea does, then dispatches its
// event only while the request is current.
func (h *harness) deliver(cmd tea.Cmd) {
	h.dispatch(cmd().(Result))
}

func (h *harness) dispatch(result Result) {
	if h.coord.Current(result) {
		h.events = append(h.events, result.Event)
	}
}

func onlyRiskEvent(t *testing.T, h *harness) sdk.RiskAssessedEvent {
	t.Helper()
	if len(h.events) != 1 {
		t.Fatalf("dispatched %d RiskAssessedEvents, want 1: %+v", len(h.events), h.events)
	}
	return h.events[0]
}

// requestRisk runs one assessment request and returns the event plugins receive.
func requestRisk(t *testing.T, h *harness, cmd tea.Cmd) sdk.RiskAssessedEvent {
	t.Helper()
	h.events = nil
	h.deliver(cmd)
	return onlyRiskEvent(t, h)
}

func reviewedRiskSummary() *sdk.PlanSummary {
	resource := sdk.PolicyResource{Address: "aws_db_instance.database", Mode: "managed", Type: "aws_db_instance", Name: "database", ProviderName: "registry.terraform.io/hashicorp/aws", Index: sdk.PolicyValue{Kind: "absent"}}
	return &sdk.PlanSummary{PolicyEvidence: &sdk.PolicyEvidence{PlanID: "reviewed-plan", Resources: []sdk.PolicyResourceEvidence{{Address: resource.Address, Actions: []string{"delete"}, Input: &sdk.PolicyInput{SchemaVersion: "tfui.risk.input/v1", Resource: resource, Change: sdk.PolicyChange{Actions: []string{"delete"}, Before: sdk.PolicyValue{Kind: "null"}, After: sdk.PolicyValue{Kind: "null"}, ReplacePaths: [][]any{}}}}}}}
}

func writePolicyFile(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
}
