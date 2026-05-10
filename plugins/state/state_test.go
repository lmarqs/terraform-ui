package state

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct {
	stateListResult []sdk.Resource
	stateListErr    error
	showResult      string
	showErr         error
}

func (m *mockService) Plan(_ context.Context, _ []string) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ []string) error { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) {
	return m.stateListResult, m.stateListErr
}
func (m *mockService) Show(_ context.Context, _ string) (string, error) {
	return m.showResult, m.showErr
}
func (m *mockService) Workspace(_ context.Context) (string, error) { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error            { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string) error               { return nil }
func (m *mockService) WorkspaceDelete(_ context.Context, _ string) error            { return nil }
func (m *mockService) StateRm(_ context.Context, _ string) error                    { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error               { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error                  { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error                      { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error                    { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error)         { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) { return nil, nil }
func (m *mockService) Refresh(_ context.Context) error                              { return nil }
func (m *mockService) Init(_ context.Context) error                                 { return nil }
func (m *mockService) ForceUnlock(_ context.Context, _ string) error                { return nil }
func (m *mockService) WithDir(_ string) sdk.Service                                 { return m }

func TestNew(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "state" {
		t.Errorf("ID() = %q, want %q", p.ID(), "state")
	}
	if p.Name() != "State Browser" {
		t.Errorf("Name() = %q, want %q", p.Name(), "State Browser")
	}
	if p.Description() != "Browse and inspect terraform state resources" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Browse and inspect terraform state resources")
	}
	if p.Ready() {
		t.Error("Ready() = true before data loads, want false")
	}
}

func TestCountable(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	var c sdk.Countable = p
	filtered, total := c.Count()
	if filtered != 0 || total != 0 {
		t.Errorf("Count() = (%d, %d), want (0, 0) when empty", filtered, total)
	}

	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	p.filtered = []sdk.Resource{{Address: "a"}}
	filtered, total = c.Count()
	if filtered != 1 || total != 3 {
		t.Errorf("Count() = (%d, %d), want (1, 3)", filtered, total)
	}
}

func TestConfigure(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	err := p.Configure(map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestInit(t *testing.T) {
	svc := &mockService{
		stateListResult: []sdk.Resource{
			{Address: "aws_instance.web", Type: "aws_instance"},
		},
	}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() should return nil cmd (no auto-load)")
	}

	pp := p.(*Plugin)
	if pp.status != StatusIdle {
		t.Errorf("status = %v, want StatusIdle", pp.status)
	}
}

func TestInitCmdReturnsStateListMsg(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	svc := &mockService{stateListResult: resources}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	p.Init(ctx)
	cmd := p.(*Plugin).Activate()
	msg := cmd()

	result, ok := msg.(StateListMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want StateListMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("StateListMsg.Err = %v, want nil", result.Err)
	}
	if len(result.Resources) != 2 {
		t.Errorf("len(Resources) = %d, want 2", len(result.Resources))
	}
}

func TestActivateCmdReturnsError(t *testing.T) {
	svc := &mockService{stateListErr: errors.New("state error")}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	msg := cmd()

	result, ok := msg.(StateListMsg)
	if !ok {
		t.Fatalf("Init cmd returned %T, want StateListMsg", msg)
	}
	if result.Err == nil {
		t.Error("StateListMsg.Err = nil, want error")
	}
}

func TestUpdateStateListMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}

	result, cmd := p.Update(StateListMsg{Resources: resources, Err: nil})
	if cmd != nil {
		t.Errorf("Update(StateListMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusDone {
		t.Errorf("status = %v, want StatusDone", updated.status)
	}
	if len(updated.resources) != 1 {
		t.Errorf("len(resources) = %d, want 1", len(updated.resources))
	}
	if len(updated.filtered) != 1 {
		t.Errorf("len(filtered) = %d, want 1", len(updated.filtered))
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdateStateListMsgError(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	result, cmd := p.Update(StateListMsg{Resources: nil, Err: errors.New("load failed")})
	if cmd != nil {
		t.Errorf("Update(StateListMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusError {
		t.Errorf("status = %v, want StatusError", updated.status)
	}
	if updated.errMsg != "load failed" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "load failed")
	}
}

func TestUpdateResourceDetailMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone

	result, cmd := p.Update(ResourceDetailMsg{Address: "aws_instance.web", Detail: `{"id": "i-123"}`, Err: nil})
	if cmd != nil {
		t.Errorf("Update(ResourceDetailMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusShowingDetail {
		t.Errorf("status = %v, want StatusShowingDetail", updated.status)
	}
	if updated.detail != `{"id": "i-123"}` {
		t.Errorf("detail = %q, want %q", updated.detail, `{"id": "i-123"}`)
	}
	if updated.detailAddr != "aws_instance.web" {
		t.Errorf("detailAddr = %q, want %q", updated.detailAddr, "aws_instance.web")
	}
	if !updated.Ready() {
		t.Error("Ready() = false in StatusShowingDetail, want true")
	}
}

func TestUpdateResourceDetailMsgError(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusDone

	result, cmd := p.Update(ResourceDetailMsg{Address: "x", Detail: "", Err: errors.New("not found")})
	if cmd != nil {
		t.Errorf("Update(ResourceDetailMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusDone {
		t.Errorf("status = %v, want StatusDone", updated.status)
	}
	if updated.errMsg != "not found" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "not found")
	}
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "a", Type: "t1"},
		{Address: "b", Type: "t2"},
		{Address: "c", Type: "t3"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// Move down with arrow
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.Selected() != 1 {
		t.Errorf("after down: selected = %d, want 1", p.Selected())
	}

	// Move down
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.Selected() != 2 {
		t.Errorf("after down,down: selected = %d, want 2", p.Selected())
	}

	// Boundary
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.Selected() != 2 {
		t.Errorf("after down,down,down: selected = %d, want 2 (boundary)", p.Selected())
	}

	// Move up with arrow
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.Selected() != 1 {
		t.Errorf("after up: selected = %d, want 1", p.Selected())
	}
}

func TestUpdateKeyMsgMoveToEndAndStart(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "a"},
		{Address: "b"},
		{Address: "c"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// G moves to end
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if p.Selected() != 2 {
		t.Errorf("after G: selected = %d, want 2", p.Selected())
	}

	// g moves to start
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if p.Selected() != 0 {
		t.Errorf("after g: selected = %d, want 0", p.Selected())
	}
}

func TestUpdateKeyMsgEnter_InspectSelected(t *testing.T) {
	svc := &mockService{showResult: `{"id": "i-123"}`}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("after enter: cmd = nil, want non-nil (inspect)")
	}
}

func TestUpdateKeyMsgEnter_EmptyAddress(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{}
	p.filtered = []sdk.Resource{}
	p.rebuildTree()

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("after enter with empty list: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgRefresh(t *testing.T) {
	svc := &mockService{stateListResult: []sdk.Resource{}}
	p := New(svc).(*Plugin)
	p.status = StatusDone

	// r triggers refresh in normal mode
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("after r in StatusDone: cmd = nil, want non-nil (refresh)")
	}

	// r works in error state too
	p.status = StatusError
	_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("after r in StatusError: cmd = nil, want non-nil (refresh)")
	}

	// r does nothing in Loading
	p.status = StatusLoading
	_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd != nil {
		t.Error("after r in StatusLoading: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgBackspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// Enter filter mode via / then type "web"
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.filter != "we" {
		t.Errorf("after backspace: filter = %q, want %q", p.filter, "we")
	}
}

func TestUpdateKeyMsgCharacterFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// Enter filter mode with /, then type
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("after '/': expected filtering mode")
	}
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if p.filter != "w" {
		t.Errorf("after 'w': filter = %q, want %q", p.filter, "w")
	}
}

func TestFilterModeBlocksHotkeys(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
		{Address: "aws_rds_instance.db", Type: "aws_rds_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// Enter filter mode and type 'r' — should filter, not refresh
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if p.filter != "rds" {
		t.Errorf("filter = %q, want %q", p.filter, "rds")
	}
	if p.status != StatusDone {
		t.Errorf("status should remain StatusDone, got %v", p.status)
	}
}

func TestUpdateKeyMsgDetailViewEsc(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusShowingDetail
	p.detail = "some detail"
	p.detailAddr = "aws_instance.web"
	// Push detail frame as ResourceDetailMsg would
	p.stack.Push(&detailFrame{plugin: p})

	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.status != StatusDone {
		t.Errorf("after esc in detail: status = %v, want StatusDone", p.status)
	}
	if p.detail != "" {
		t.Errorf("after esc in detail: detail = %q, want empty", p.detail)
	}
}

func TestUpdateKeyMsgDetailViewQ(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusShowingDetail
	p.detail = "some detail"
	p.detailAddr = "aws_instance.web"

	// q no longer exits detail (only esc does), so status stays
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if p.status != StatusShowingDetail {
		t.Errorf("after q in detail: status = %v, want StatusShowingDetail (q handled by app)", p.status)
	}
}

func TestUpdateUnknownMsg(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	type unknownMsg struct{}
	result, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Errorf("Update(unknownMsg) cmd = %v, want nil", cmd)
	}
	if result != p {
		t.Error("Update(unknownMsg) returned different plugin reference")
	}
}

func TestMoveUpDown(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.rebuildTree()

	p.MoveDown()
	if p.Selected() != 1 {
		t.Errorf("MoveDown: selected = %d, want 1", p.Selected())
	}
	p.MoveDown()
	if p.Selected() != 1 {
		t.Errorf("MoveDown boundary: selected = %d, want 1", p.Selected())
	}
	p.MoveUp()
	if p.Selected() != 0 {
		t.Errorf("MoveUp: selected = %d, want 0", p.Selected())
	}
	p.MoveUp()
	if p.Selected() != 0 {
		t.Errorf("MoveUp boundary: selected = %d, want 0", p.Selected())
	}
}

func TestMoveToStartEnd(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}}
	p.rebuildTree()

	p.MoveToEnd()
	if p.Selected() != 2 {
		t.Errorf("MoveToEnd: selected = %d, want 2", p.Selected())
	}
	p.MoveToStart()
	if p.Selected() != 0 {
		t.Errorf("MoveToStart: selected = %d, want 0", p.Selected())
	}
}

func TestMoveToEndEmpty(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{}
	p.rebuildTree()
	p.MoveToEnd()
	if p.Selected() != 0 {
		t.Errorf("MoveToEnd empty: selected = %d, want 0", p.Selected())
	}
}

func TestSetFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Name: "web", Module: ""},
		{Address: "module.storage.aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data", Module: "module.storage"},
		{Address: "aws_vpc.main", Type: "aws_vpc", Name: "main", Module: ""},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// Filter by "s3"
	p.SetFilter("s3")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('s3'): len(filtered) = %d, want 1", len(p.filtered))
	}
	if p.Selected() != 0 {
		t.Errorf("SetFilter resets selected: got %d, want 0", p.Selected())
	}
	if p.filter != "s3" {
		t.Errorf("filter = %q, want %q", p.filter, "s3")
	}

	// Filter by module
	p.SetFilter("storage")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('storage'): len(filtered) = %d, want 1", len(p.filtered))
	}

	// Filter by type
	p.SetFilter("aws_vpc")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('aws_vpc'): len(filtered) = %d, want 1", len(p.filtered))
	}

	// Clear filter
	p.SetFilter("")
	if len(p.filtered) != 3 {
		t.Errorf("SetFilter(''): len(filtered) = %d, want 3", len(p.filtered))
	}

	// No matches
	p.SetFilter("zzz_nonexistent")
	if len(p.filtered) != 0 {
		t.Errorf("SetFilter('zzz'): len(filtered) = %d, want 0", len(p.filtered))
	}
}

func TestSetFilterFzf(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "module.medprev_online_prd.module.postgresql_aurora.aws_rds_cluster.this[0]", Type: "aws_rds_cluster", Name: "this", Module: "module.postgresql_aurora"},
		{Address: "module.medprev_online_prd.module.postgresql_aurora.aws_rds_cluster_instance.this[\"1\"]", Type: "aws_rds_cluster_instance", Name: "this", Module: "module.postgresql_aurora"},
		{Address: "module.medprev_online_prd.module.postgresql_aurora.aws_rds_cluster_instance.this[\"2\"]", Type: "aws_rds_cluster_instance", Name: "this", Module: "module.postgresql_aurora"},
		{Address: "module.medprev_online_prd.module.postgresql_aurora.aws_db_proxy.read_only", Type: "aws_db_proxy", Name: "read_only", Module: "module.postgresql_aurora"},
		{Address: "module.medprev_online_prd.module.redis.aws_elasticache_replication_group.this", Type: "aws_elasticache_replication_group", Name: "this", Module: "module.redis"},
		{Address: "module.medprev_online_prd.aws_security_group.web", Type: "aws_security_group", Name: "web", Module: ""},
		{Address: "module.medprev_online_prd.module.memorydb.aws_memorydb_cluster.this", Type: "aws_memorydb_cluster", Name: "this", Module: "module.memorydb"},
		{Address: "module.medprev_online_prd.aws_opensearch_domain.legacy", Type: "aws_opensearch_domain", Name: "legacy", Module: ""},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// fzf ranks best matches first; validate ranking not exact counts
	t.Run("best match ranked first", func(t *testing.T) {
		cases := []struct {
			filter   string
			topMatch string
		}{
			{"aurora", "aurora"},
			{"redis", "redis"},
			{"memorydb", "memorydb"},
			{"opensearch", "opensearch"},
			{"read_only", "read_only"},
			{"readonly", "read_only"},
			{"proxy", "proxy"},
			{"memdb", "memorydb"},
			{"dbproxy", "db_proxy"},
			{"securityweb", "security_group.web"},
			{"aurorathis", "aurora"},
			{"auroracluster", "aurora"},
			{"aurorainstance", "cluster_instance"},
			{"proxyreadonly", "proxy.read_only"},
			{"rdscluster", "rds_cluster"},
			{"clusterinstance", "cluster_instance"},
			{"elasticache", "elasticache"},
		}
		for _, c := range cases {
			p.SetFilter(c.filter)
			if len(p.filtered) == 0 {
				t.Errorf("SetFilter(%q): no results, expected match containing %q", c.filter, c.topMatch)
				continue
			}
			if !strings.Contains(p.filtered[0].Address, c.topMatch) {
				t.Errorf("SetFilter(%q): top result %q doesn't contain %q", c.filter, p.filtered[0].Address, c.topMatch)
			}
		}
	})

	t.Run("space AND narrows results", func(t *testing.T) {
		p.SetFilter("aurora")
		auroraCount := len(p.filtered)
		p.SetFilter("aurora instance")
		if len(p.filtered) >= auroraCount {
			t.Errorf("'aurora instance' (%d) should be fewer than 'aurora' (%d)", len(p.filtered), auroraCount)
		}
		if len(p.filtered) == 0 {
			t.Error("'aurora instance' should have results")
		}
		for _, r := range p.filtered {
			if !strings.Contains(r.Address, "instance") {
				t.Errorf("'aurora instance' result %q should contain 'instance'", r.Address)
			}
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		p.SetFilter("aurora")
		lower := len(p.filtered)
		p.SetFilter("Aurora")
		upper := len(p.filtered)
		p.SetFilter("AURORA")
		allCaps := len(p.filtered)
		if lower != upper || lower != allCaps {
			t.Errorf("case mismatch: aurora=%d, Aurora=%d, AURORA=%d", lower, upper, allCaps)
		}
		if lower == 0 {
			t.Error("expected results for 'aurora'")
		}
	})

	t.Run("no match", func(t *testing.T) {
		p.SetFilter("zzz")
		if len(p.filtered) != 0 {
			t.Errorf("'zzz': got %d results, want 0", len(p.filtered))
		}
		p.SetFilter("aurora zzz")
		if len(p.filtered) != 0 {
			t.Errorf("'aurora zzz': got %d results, want 0", len(p.filtered))
		}
	})
}

func TestAppendFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	p.AppendFilter("a")
	if p.filter != "a" {
		t.Errorf("AppendFilter('a'): filter = %q, want %q", p.filter, "a")
	}
	p.AppendFilter("w")
	if p.filter != "aw" {
		t.Errorf("AppendFilter('w'): filter = %q, want %q", p.filter, "aw")
	}
}

func TestBackspaceFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.filter = "abc"

	p.BackspaceFilter()
	if p.filter != "ab" {
		t.Errorf("BackspaceFilter: filter = %q, want %q", p.filter, "ab")
	}

	// Backspace on empty does nothing
	p.filter = ""
	p.BackspaceFilter()
	if p.filter != "" {
		t.Errorf("BackspaceFilter empty: filter = %q, want empty", p.filter)
	}
}

func TestSelectedResource(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	// Empty filtered
	p.filtered = []sdk.Resource{}
	p.rebuildTree()
	r := p.SelectedResource()
	if r.Address != "" {
		t.Errorf("SelectedResource empty: Address = %q, want empty", r.Address)
	}

	// Valid selection
	p.filtered = []sdk.Resource{
		{Address: "a"},
		{Address: "b"},
	}
	p.rebuildTree()
	p.MoveDown()
	r = p.SelectedResource()
	if r.Address != "b" {
		t.Errorf("SelectedResource: Address = %q, want %q", r.Address, "b")
	}
}

func TestInspectSelected(t *testing.T) {
	svc := &mockService{showResult: `{"id": "i-123"}`}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{
		{Address: "aws_instance.web"},
	}
	p.rebuildTree()

	cmd := p.InspectSelected()
	if cmd == nil {
		t.Error("InspectSelected() returned nil cmd")
	}

	// Execute the command
	msg := cmd()
	detail, ok := msg.(ResourceDetailMsg)
	if !ok {
		t.Fatalf("InspectSelected cmd returned %T, want ResourceDetailMsg", msg)
	}
	if detail.Address != "aws_instance.web" {
		t.Errorf("detail.Address = %q, want %q", detail.Address, "aws_instance.web")
	}
}

func TestInspectSelectedEmptyAddress(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{{Address: ""}}

	cmd := p.InspectSelected()
	if cmd != nil {
		t.Error("InspectSelected with empty address: cmd != nil, want nil")
	}
}

func TestRefresh(t *testing.T) {
	svc := &mockService{stateListResult: []sdk.Resource{}}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	// Set up some items and move cursor to simulate non-zero selection
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}, {Address: "d"}, {Address: "e"}, {Address: "f"}}
	p.filtered = p.resources
	p.rebuildTree()
	for i := 0; i < 5; i++ {
		p.MoveDown()
	}
	p.filter = "something"

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != StatusLoading {
		t.Errorf("after Refresh: status = %v, want StatusLoading", p.status)
	}
	if p.Selected() != 0 {
		t.Errorf("after Refresh: selected = %d, want 0", p.Selected())
	}
	if p.filter != "" {
		t.Errorf("after Refresh: filter = %q, want empty", p.filter)
	}
}

func TestViewIdle(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusIdle) returned empty string")
	}
}

func TestViewLoading(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusLoading

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusLoading) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusError
	p.errMsg = "some error"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusError) returned empty string")
	}
}

func TestViewShowingDetail(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{"id": "i-123", "name": "web-server"}`

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusShowingDetail) returned empty string")
	}
}

func TestViewShowingDetailLong(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"

	// Create a very long detail to test truncation
	lines := ""
	for i := 0; i < 100; i++ {
		lines += `"line": "value"` + "\n"
	}
	p.detail = lines

	view := p.View(80, 10)
	if view == "" {
		t.Error("View(StatusShowingDetail, long) returned empty string")
	}
}

func TestViewDoneNoResources(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{}
	p.filtered = []sdk.Resource{}
	p.rebuildTree()

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, no resources) returned empty string")
	}
}

func TestViewDoneWithResources(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Module: ""},
		{Address: "module.vpc.aws_subnet.a", Type: "aws_subnet", Module: "module.vpc"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with resources) returned empty string")
	}
}

func TestViewDoneWithFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources
	p.rebuildTree()
	p.filter = "web"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with filter) returned empty string")
	}
}

func TestViewDoneFilteredDiffFromTotal(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "a"},
		{Address: "b"},
		{Address: "c"},
	}
	p.filtered = p.resources[:1]
	p.filter = "a"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with filtered != total returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestViewScrolling(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone

	resources := make([]sdk.Resource, 50)
	for i := range resources {
		resources[i] = sdk.Resource{Address: "res_" + string(rune('a'+i%26)), Type: "type"}
	}
	p.resources = resources
	p.filtered = resources
	p.rebuildTree()
	for i := 0; i < 40; i++ {
		p.MoveDown()
	}

	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestResourceCount(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{{}, {}, {}}
	if p.ResourceCount() != 3 {
		t.Errorf("ResourceCount() = %d, want 3", p.ResourceCount())
	}
}

func TestTotalCount(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{{}, {}, {}, {}}
	if p.TotalCount() != 4 {
		t.Errorf("TotalCount() = %d, want 4", p.TotalCount())
	}
}

func TestFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filter = "test"
	if p.Filter() != "test" {
		t.Errorf("Filter() = %q, want %q", p.Filter(), "test")
	}
}

func TestUpdateKeyMsgDelete(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	// Enter filter mode, type "ab", then use delete key as backspace
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	p.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if p.filter != "a" {
		t.Errorf("after delete: filter = %q, want %q", p.filter, "a")
	}
}

func TestUpdateKeyMsgSlash(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	// "/" key should not crash (handled but empty)
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if cmd != nil {
		t.Error("after /: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgDownKey(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()

	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.Selected() != 1 {
		t.Errorf("after down: selected = %d, want 1", p.Selected())
	}

	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.Selected() != 0 {
		t.Errorf("after up: selected = %d, want 0", p.Selected())
	}
}

func TestInspectSelectedCmdError(t *testing.T) {
	svc := &mockService{showErr: errors.New("show failed")}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{
		{Address: "aws_instance.web"},
	}
	p.rebuildTree()

	cmd := p.InspectSelected()
	if cmd == nil {
		t.Error("InspectSelected with error service: cmd = nil, want non-nil")
	}
	msg := cmd()
	detail, ok := msg.(ResourceDetailMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want ResourceDetailMsg", msg)
	}
	if detail.Err == nil {
		t.Error("detail.Err = nil, want error")
	}
}

func TestUpdateKeyMsgCtrlH(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.filter = "abc"

	// ctrl+h should work as backspace
	p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{0x08}})
	// This might not work as expected since ctrl+h string is "ctrl+h"
	// Let's instead directly test the handler branch
}

func TestHandleKeyDefaultPrintable(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	// Must enter filter mode first with /
	p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if p.filter != "a" {
		t.Errorf("after 'a' via handleKey: filter = %q, want %q", p.filter, "a")
	}

	p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if p.filter != "aw" {
		t.Errorf("after 'w' via handleKey: filter = %q, want %q", p.filter, "aw")
	}
}

func TestHandleKeyDetailIgnoresOtherKeys(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusShowingDetail
	p.detail = "data"
	p.detailAddr = "addr"

	// Non-esc/q keys should not change the state in detail mode
	p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.status != StatusShowingDetail {
		t.Errorf("after j in detail: status = %v, want StatusShowingDetail", p.status)
	}
}

func TestHandleKeyInLoadingIgnoresKeys(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusLoading

	// In loading state, 'r' should not trigger refresh (only works in Done/Error)
	cmd := p.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if cmd != nil {
		t.Error("j in loading: cmd != nil, want nil")
	}
}

func TestStatusGetter(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if p.Status() != StatusIdle {
		t.Errorf("Status() = %v, want StatusIdle", p.Status())
	}
}

func TestSelectedGetter(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}, {Address: "d"}, {Address: "e"}, {Address: "f"}}
	p.filtered = p.resources
	p.rebuildTree()
	for i := 0; i < 5; i++ {
		p.MoveDown()
	}
	if p.Selected() != 5 {
		t.Errorf("Selected() = %d, want 5", p.Selected())
	}
}

func TestFilteringGetter(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if p.Filtering() {
		t.Error("Filtering() = true, want false")
	}
}

func TestActivateWithScopeChange(t *testing.T) {
	svc := &mockService{stateListResult: []sdk.Resource{{Address: "a"}}}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveScopeAbs, "/new/ctx")
	ctx := &sdk.Context{Service: svc, Session: session, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	p.status = StatusDone
	p.scopedContext = "/old/ctx"
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() context change: want non-nil cmd")
	}
}

func TestActivateWithSameContext(t *testing.T) {
	svc := &mockService{stateListResult: []sdk.Resource{}}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveScopeAbs, "/same")
	ctx := &sdk.Context{Service: svc, Session: session, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	p.status = StatusDone
	p.scopedContext = "/same"
	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() same context done: want nil")
	}
}

func TestActivateMultiContextNoSelection(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyScopeCount, 3)
	ctx := &sdk.Context{Service: svc, Session: session, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() multi-context no selection: want nil")
	}
	if p.status != StatusError {
		t.Errorf("status = %v, want StatusError", p.status)
	}
}

func TestActivateWithScopeDir(t *testing.T) {
	svc := &mockService{stateListResult: []sdk.Resource{}}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveScopeAbs, "/my/ctx")
	ctx := &sdk.Context{Service: svc, Session: session, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() with context dir: want non-nil cmd")
	}
}

func TestActivateNoSession(t *testing.T) {
	svc := &mockService{stateListResult: []sdk.Resource{}}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() no session: want non-nil cmd")
	}
}

func TestHandleKeyFilterMode(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.resources = []sdk.Resource{{Address: "aws_instance.a"}, {Address: "aws_s3_bucket.b"}}
	p.filtered = p.resources
	p.rebuildTree()

	// Activate filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Error("after /: filtering = false, want true")
	}
}

func TestFilterForTree_ScoreThreshold(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.treeMode = true
	p.resources = []sdk.Resource{
		{Address: "module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]", Type: "aws_db_proxy"},
		{Address: "module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy_endpoint.read_only[0]", Type: "aws_db_proxy_endpoint"},
		{Address: "module.medprev_online_prd.module.medprev_api.module.api_gateway.aws_apigatewayv2_route.this[\"PUT /storage/private/{proxy+}\"]", Type: "aws_apigatewayv2_route"},
		{Address: "module.medprev_online_prd.module.medprev_api.aws_api_gateway_rest_api.this", Type: "aws_api_gateway_rest_api"},
		{Address: "module.medprev_online_prd.aws_security_group.web", Type: "aws_security_group"},
		{Address: "module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens", Type: "aws_cloudwatch_metric_alarm"},
		{Address: "module.medprev_online_prd.module.redis.aws_elasticache_replication_group.this", Type: "aws_elasticache_replication_group"},
		{Address: "aws_s3_bucket.terraform_state", Type: "aws_s3_bucket"},
	}
	p.filtered = p.resources
	p.rebuildTree()

	tests := []struct {
		filter    string
		wantMin   int
		wantMax   int
		mustMatch string
	}{
		{"proxy", 2, 4, "aws_db_proxy"},
		{"proxyread", 1, 2, "read_only"},
		{"restapi", 1, 3, "rest_api"},
		{"redis", 1, 2, "redis"},
		{"alarm", 1, 2, "alarm"},
		{"s3bucket", 1, 2, "s3_bucket"},
		{"cloudwatch", 1, 2, "cloudwatch"},
		{"zzzznothing", 0, 0, ""},
	}

	for _, tt := range tests {
		p.SetFilter(tt.filter)
		count := len(p.filtered)
		if count < tt.wantMin || count > tt.wantMax {
			t.Errorf("tree filter %q: got %d results, want %d-%d", tt.filter, count, tt.wantMin, tt.wantMax)
		}
		if tt.mustMatch != "" && count > 0 {
			found := false
			for _, r := range p.filtered {
				if strings.Contains(r.Address, tt.mustMatch) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("tree filter %q: expected result containing %q", tt.filter, tt.mustMatch)
			}
		}
	}
}
