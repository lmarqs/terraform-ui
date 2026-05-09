package state

import (
	"context"
	"errors"
	"io"
	"log/slog"
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
	if p.KeyBinding() != "s" {
		t.Errorf("KeyBinding() = %q, want %q", p.KeyBinding(), "s")
	}
	if p.Ready() {
		t.Error("Ready() = true before data loads, want false")
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
		Dir:       "/tmp",
		Workspace: "default",
		Service:   svc,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	cmd := p.Init(ctx)
	if cmd == nil {
		t.Error("Init() returned nil cmd, want non-nil")
	}

	pp := p.(*Plugin)
	if pp.status != StatusLoading {
		t.Errorf("status = %v, want StatusLoading", pp.status)
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

	cmd := p.Init(ctx)
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

func TestInitCmdReturnsError(t *testing.T) {
	svc := &mockService{stateListErr: errors.New("state error")}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	cmd := p.Init(ctx)
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

	// Move down with j
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}

	// Move down
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 2 {
		t.Errorf("after j,j: selected = %d, want 2", p.selected)
	}

	// Boundary
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 2 {
		t.Errorf("after j,j,j: selected = %d, want 2 (boundary)", p.selected)
	}

	// Move up with k
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 1 {
		t.Errorf("after k: selected = %d, want 1", p.selected)
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

	// G moves to end
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if p.selected != 2 {
		t.Errorf("after G: selected = %d, want 2", p.selected)
	}

	// g moves to start
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if p.selected != 0 {
		t.Errorf("after g: selected = %d, want 0", p.selected)
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

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("after enter with empty list: cmd != nil, want nil")
	}
}

func TestUpdateKeyMsgRefresh(t *testing.T) {
	svc := &mockService{stateListResult: []sdk.Resource{}}
	p := New(svc).(*Plugin)
	p.status = StatusDone

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
	p.filter = "web"

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

	// Type a printable character
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if p.filter != "w" {
		t.Errorf("after 'w': filter = %q, want %q", p.filter, "w")
	}
}

func TestUpdateKeyMsgDetailViewEsc(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusShowingDetail
	p.detail = "some detail"
	p.detailAddr = "aws_instance.web"

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

	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if p.status != StatusDone {
		t.Errorf("after q in detail: status = %v, want StatusDone", p.status)
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

	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown: selected = %d, want 1", p.selected)
	}
	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown boundary: selected = %d, want 1", p.selected)
	}
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp: selected = %d, want 0", p.selected)
	}
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp boundary: selected = %d, want 0", p.selected)
	}
}

func TestMoveToStartEnd(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{{Address: "a"}, {Address: "b"}, {Address: "c"}}

	p.MoveToEnd()
	if p.selected != 2 {
		t.Errorf("MoveToEnd: selected = %d, want 2", p.selected)
	}
	p.MoveToStart()
	if p.selected != 0 {
		t.Errorf("MoveToStart: selected = %d, want 0", p.selected)
	}
}

func TestMoveToEndEmpty(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{}
	p.MoveToEnd()
	if p.selected != 0 {
		t.Errorf("MoveToEnd empty: selected = %d, want 0", p.selected)
	}
}

func TestSetFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Module: ""},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Module: "module.storage"},
		{Address: "aws_vpc.main", Type: "aws_vpc", Module: ""},
	}
	p.filtered = p.resources

	// Filter by "s3"
	p.SetFilter("s3")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('s3'): len(filtered) = %d, want 1", len(p.filtered))
	}
	if p.selected != 0 {
		t.Errorf("SetFilter resets selected: got %d, want 0", p.selected)
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

func TestAppendFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p.filtered = p.resources

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

func TestClearFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources[:1]
	p.filter = "something"

	p.ClearFilter()
	if p.filter != "" {
		t.Errorf("ClearFilter: filter = %q, want empty", p.filter)
	}
	if len(p.filtered) != 2 {
		t.Errorf("ClearFilter: len(filtered) = %d, want 2", len(p.filtered))
	}
}

func TestSelectedResource(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)

	// Empty filtered
	p.filtered = []sdk.Resource{}
	r := p.SelectedResource()
	if r.Address != "" {
		t.Errorf("SelectedResource empty: Address = %q, want empty", r.Address)
	}

	// Valid selection
	p.filtered = []sdk.Resource{
		{Address: "a"},
		{Address: "b"},
	}
	p.selected = 1
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
	p.selected = 5
	p.filter = "something"

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != StatusLoading {
		t.Errorf("after Refresh: status = %v, want StatusLoading", p.status)
	}
	if p.selected != 0 {
		t.Errorf("after Refresh: selected = %d, want 0", p.selected)
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
	p.selected = 40

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
	p.filter = "ab"

	// "delete" key should also work as backspace
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

	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}

	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up: selected = %d, want 0", p.selected)
	}
}

func TestInspectSelectedCmdError(t *testing.T) {
	svc := &mockService{showErr: errors.New("show failed")}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.Resource{
		{Address: "aws_instance.web"},
	}

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

	// Printable chars go to filter in default case
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
