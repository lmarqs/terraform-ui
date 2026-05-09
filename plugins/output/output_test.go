package output

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type mockService struct {
	outputResult map[string]sdk.OutputValue
	outputErr    error
}

func (m *mockService) Plan(_ context.Context, _ []string) (*sdk.PlanSummary, error) {
	return &sdk.PlanSummary{}, nil
}
func (m *mockService) Apply(_ context.Context, _ []string) error { return nil }
func (m *mockService) StateList(_ context.Context) ([]sdk.Resource, error) {
	return nil, nil
}
func (m *mockService) Show(_ context.Context, _ string) (string, error) { return "", nil }
func (m *mockService) Workspace(_ context.Context) (string, error)      { return "default", nil }
func (m *mockService) WorkspaceList(_ context.Context) ([]string, error) {
	return []string{"default"}, nil
}
func (m *mockService) WorkspaceSelect(_ context.Context, _ string) error  { return nil }
func (m *mockService) WorkspaceNew(_ context.Context, _ string) error     { return nil }
func (m *mockService) WorkspaceDelete(_ context.Context, _ string) error  { return nil }
func (m *mockService) StateRm(_ context.Context, _ string) error          { return nil }
func (m *mockService) StateMove(_ context.Context, _, _ string) error     { return nil }
func (m *mockService) Import(_ context.Context, _, _ string) error        { return nil }
func (m *mockService) Taint(_ context.Context, _ string) error            { return nil }
func (m *mockService) Untaint(_ context.Context, _ string) error          { return nil }
func (m *mockService) Validate(_ context.Context) ([]sdk.Diagnostic, error) { return nil, nil }
func (m *mockService) Output(_ context.Context) (map[string]sdk.OutputValue, error) {
	return m.outputResult, m.outputErr
}
func (m *mockService) Refresh(_ context.Context) error { return nil }
func (m *mockService) Init(_ context.Context) error    { return nil }
func (m *mockService) WithDir(_ string) sdk.Service    { return m }

func sampleOutputs() map[string]sdk.OutputValue {
	return map[string]sdk.OutputValue{
		"vpc_id": {
			Name:      "vpc_id",
			Value:     "vpc-abc123",
			Type:      "string",
			Sensitive: false,
		},
		"db_password": {
			Name:      "db_password",
			Value:     "secret123",
			Type:      "string",
			Sensitive: true,
		},
		"instance_ids": {
			Name:      "instance_ids",
			Value:     []interface{}{"i-111", "i-222"},
			Type:      "list",
			Sensitive: false,
		},
	}
}

func TestNew(t *testing.T) {
	svc := &mockService{}
	p := New(svc)

	if p.ID() != "output" {
		t.Errorf("ID() = %q, want %q", p.ID(), "output")
	}
	if p.Name() != "Outputs" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Outputs")
	}
	if p.Description() != "View terraform outputs" {
		t.Errorf("Description() = %q, want %q", p.Description(), "View terraform outputs")
	}
	if p.KeyBinding() != "o" {
		t.Errorf("KeyBinding() = %q, want %q", p.KeyBinding(), "o")
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
	svc := &mockService{outputResult: sampleOutputs()}
	p := New(svc)
	ctx := &sdk.Context{
		Dir:       "/tmp",
		Workspace: "default",
		Service:   svc,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
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

func TestActivateCmdReturnsOutputResultMsg(t *testing.T) {
	outputs := sampleOutputs()
	svc := &mockService{outputResult: outputs}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	p.Init(ctx)
	cmd := p.(*Plugin).Activate()
	if cmd == nil {
		t.Fatal("Activate() returned nil cmd, want non-nil")
	}
	msg := cmd()

	result, ok := msg.(OutputResultMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want OutputResultMsg", msg)
	}
	if result.Err != nil {
		t.Errorf("OutputResultMsg.Err = %v, want nil", result.Err)
	}
	if len(result.Outputs) != 3 {
		t.Errorf("len(Outputs) = %d, want 3", len(result.Outputs))
	}
}

func TestActivateCmdReturnsError(t *testing.T) {
	svc := &mockService{outputErr: errors.New("output error")}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	if cmd == nil {
		t.Fatal("Activate() returned nil cmd, want non-nil")
	}
	msg := cmd()

	result, ok := msg.(OutputResultMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want OutputResultMsg", msg)
	}
	if result.Err == nil {
		t.Error("OutputResultMsg.Err = nil, want error")
	}
}

func TestUpdateOutputResultMsgSuccess(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	outputs := sampleOutputs()

	result, cmd := p.Update(OutputResultMsg{Outputs: outputs, Err: nil})
	if cmd != nil {
		t.Errorf("Update(OutputResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusDone {
		t.Errorf("status = %v, want StatusDone", updated.status)
	}
	if len(updated.outputs) != 3 {
		t.Errorf("len(outputs) = %d, want 3", len(updated.outputs))
	}
	if len(updated.filtered) != 3 {
		t.Errorf("len(filtered) = %d, want 3", len(updated.filtered))
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdateOutputResultMsgSorted(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	outputs := sampleOutputs()
	p.Update(OutputResultMsg{Outputs: outputs, Err: nil})

	// Outputs should be sorted by name
	if pp.outputs[0].Name != "db_password" {
		t.Errorf("outputs[0].Name = %q, want %q", pp.outputs[0].Name, "db_password")
	}
	if pp.outputs[1].Name != "instance_ids" {
		t.Errorf("outputs[1].Name = %q, want %q", pp.outputs[1].Name, "instance_ids")
	}
	if pp.outputs[2].Name != "vpc_id" {
		t.Errorf("outputs[2].Name = %q, want %q", pp.outputs[2].Name, "vpc_id")
	}
}

func TestUpdateOutputResultMsgError(t *testing.T) {
	svc := &mockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = StatusLoading

	result, cmd := p.Update(OutputResultMsg{Outputs: nil, Err: errors.New("load failed")})
	if cmd != nil {
		t.Errorf("Update(OutputResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != StatusError {
		t.Errorf("status = %v, want StatusError", updated.status)
	}
	if updated.errMsg != "load failed" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "load failed")
	}
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
		{Name: "c", Type: "string", Value: "3"},
	}
	p.filtered = p.outputs

	// Move down with arrow
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}

	// Move down again
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 2 {
		t.Errorf("after down,down: selected = %d, want 2", p.selected)
	}

	// Boundary
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 2 {
		t.Errorf("after down,down,down: selected = %d, want 2 (boundary)", p.selected)
	}

	// Move up with arrow
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 1 {
		t.Errorf("after up: selected = %d, want 1", p.selected)
	}
}

func TestUpdateKeyMsgJK(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
	}
	p.filtered = p.outputs

	// j moves down
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}

	// k moves up
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 0 {
		t.Errorf("after k: selected = %d, want 0", p.selected)
	}
}

func TestUpdateKeyMsgMoveToEndAndStart(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}
	p.filtered = p.outputs

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

func TestUpdateKeyMsgRefresh(t *testing.T) {
	svc := &mockService{outputResult: sampleOutputs()}
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

func TestUpdateKeyMsgEscDeactivates(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{{Name: "a"}}
	p.filtered = p.outputs

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("after esc in normal mode: cmd = nil, want non-nil (deactivate)")
	}

	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("esc cmd returned %T, want DeactivateMsg", msg)
	}
}

func TestUpdateKeyMsgFilterMode(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
		{Name: "db_password", Type: "string", Value: "secret", Sensitive: true},
	}
	p.filtered = p.outputs

	// Enter filter mode with /
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("after '/': expected filtering mode")
	}

	// Type 'v'
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if p.filter != "v" {
		t.Errorf("after 'v': filter = %q, want %q", p.filter, "v")
	}

	// Type 'p'
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if p.filter != "vp" {
		t.Errorf("after 'p': filter = %q, want %q", p.filter, "vp")
	}
}

func TestUpdateKeyMsgFilterModeEscExits(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{{Name: "a"}}
	p.filtered = p.outputs
	p.filtering = true

	p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.filtering {
		t.Error("after esc in filter mode: filtering = true, want false")
	}
}

func TestUpdateKeyMsgBackspace(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs
	p.filter = "vpc"
	p.filtering = true

	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.filter != "vp" {
		t.Errorf("after backspace: filter = %q, want %q", p.filter, "vp")
	}
}

func TestUpdateKeyMsgDelete(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{{Name: "a"}}
	p.filtered = p.outputs
	p.filter = "ab"
	p.filtering = true

	p.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if p.filter != "a" {
		t.Errorf("after delete: filter = %q, want %q", p.filter, "a")
	}
}

func TestFilterModeBlocksHotkeys(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
		{Name: "region", Type: "string", Value: "us-east-1"},
	}
	p.filtered = p.outputs

	// Enter filter mode and type 'r' — should filter, not refresh
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if p.filter != "r" {
		t.Errorf("filter = %q, want %q", p.filter, "r")
	}
	if p.status != StatusDone {
		t.Errorf("status should remain StatusDone, got %v", p.status)
	}
}

func TestFilterModeNavigationDown(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
	}
	p.filtered = p.outputs
	p.filtering = true

	// down in filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down in filter mode: selected = %d, want 1", p.selected)
	}

	// up in filter mode
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up in filter mode: selected = %d, want 0", p.selected)
	}
}

func TestFilterModeJKAppendsToFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
	}
	p.filtered = p.outputs
	p.filtering = true

	// j in filter mode appends to filter (only arrow keys navigate)
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.filter != "j" {
		t.Errorf("j in filter mode: filter = %q, want %q (appended to filter)", p.filter, "j")
	}

	// k also appends to filter
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.filter != "jk" {
		t.Errorf("k in filter mode: filter = %q, want %q (appended to filter)", p.filter, "jk")
	}
}

func TestMoveUpDown(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.OutputValue{{Name: "a"}, {Name: "b"}}

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
	p.filtered = []sdk.OutputValue{{Name: "a"}, {Name: "b"}, {Name: "c"}}

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
	p.filtered = []sdk.OutputValue{}
	p.MoveToEnd()
	if p.selected != 0 {
		t.Errorf("MoveToEnd empty: selected = %d, want 0", p.selected)
	}
}

func TestSetFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
		{Name: "db_password", Type: "string", Value: "secret", Sensitive: true},
		{Name: "instance_ids", Type: "list", Value: []interface{}{"i-1", "i-2"}},
	}
	p.filtered = p.outputs

	// Filter by "vpc"
	p.SetFilter("vpc")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('vpc'): len(filtered) = %d, want 1", len(p.filtered))
	}
	if p.filtered[0].Name != "vpc_id" {
		t.Errorf("SetFilter('vpc'): filtered[0].Name = %q, want %q", p.filtered[0].Name, "vpc_id")
	}
	if p.selected != 0 {
		t.Errorf("SetFilter resets selected: got %d, want 0", p.selected)
	}

	// Filter by "db"
	p.SetFilter("db")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('db'): len(filtered) = %d, want 1", len(p.filtered))
	}
	if p.filtered[0].Name != "db_password" {
		t.Errorf("SetFilter('db'): filtered[0].Name = %q, want %q", p.filtered[0].Name, "db_password")
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

func TestSetFilterCaseInsensitive(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.outputs = []sdk.OutputValue{
		{Name: "VPC_ID", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs

	p.SetFilter("vpc")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('vpc') case insensitive: len(filtered) = %d, want 1", len(p.filtered))
	}

	p.SetFilter("VPC")
	if len(p.filtered) != 1 {
		t.Errorf("SetFilter('VPC') case insensitive: len(filtered) = %d, want 1", len(p.filtered))
	}
}

func TestAppendFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs

	p.AppendFilter("v")
	if p.filter != "v" {
		t.Errorf("AppendFilter('v'): filter = %q, want %q", p.filter, "v")
	}
	p.AppendFilter("p")
	if p.filter != "vp" {
		t.Errorf("AppendFilter('p'): filter = %q, want %q", p.filter, "vp")
	}
}

func TestBackspaceFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs
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

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		output   sdk.OutputValue
		expected string
	}{
		{
			name:     "string value",
			output:   sdk.OutputValue{Name: "vpc_id", Value: "vpc-abc123", Type: "string", Sensitive: false},
			expected: "vpc-abc123",
		},
		{
			name:     "sensitive value is redacted",
			output:   sdk.OutputValue{Name: "db_password", Value: "secret123", Type: "string", Sensitive: true},
			expected: "(sensitive)",
		},
		{
			name:     "list value",
			output:   sdk.OutputValue{Name: "ids", Value: []interface{}{"a", "b"}, Type: "list", Sensitive: false},
			expected: "[a b]",
		},
		{
			name:     "nil value",
			output:   sdk.OutputValue{Name: "empty", Value: nil, Type: "string", Sensitive: false},
			expected: "<nil>",
		},
		{
			name:     "sensitive list is redacted",
			output:   sdk.OutputValue{Name: "secrets", Value: []interface{}{"x"}, Type: "list", Sensitive: true},
			expected: "(sensitive)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValue(tt.output)
			if result != tt.expected {
				t.Errorf("FormatValue() = %q, want %q", result, tt.expected)
			}
		})
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

func TestViewDoneNoOutputs(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{}
	p.filtered = []sdk.OutputValue{}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, no outputs) returned empty string")
	}
}

func TestViewDoneWithOutputs(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-abc123", Sensitive: false},
		{Name: "db_password", Type: "string", Value: "secret", Sensitive: true},
	}
	p.filtered = p.outputs

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with outputs) returned empty string")
	}
	// Sensitive value should not appear in view
	if strings.Contains(view, "secret") {
		t.Error("View should not show sensitive value 'secret'")
	}
	// Non-sensitive value should appear
	if !strings.Contains(view, "vpc-abc123") {
		t.Error("View should show non-sensitive value 'vpc-abc123'")
	}
	// Redacted marker should appear
	if !strings.Contains(view, "(sensitive)") {
		t.Error("View should show '(sensitive)' for redacted values")
	}
}

func TestViewDoneWithFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs
	p.filter = "vpc"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(StatusDone, with filter) returned empty string")
	}
}

func TestViewDoneFilteredDiffFromTotal(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
		{Name: "c", Type: "string", Value: "3"},
	}
	p.filtered = p.outputs[:1]
	p.filter = "a"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with filtered != total returned empty string")
	}
	// Should show "1/3 outputs"
	if !strings.Contains(view, "1/3") {
		t.Error("View should show '1/3' when filtered != total")
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

	outputs := make([]sdk.OutputValue, 50)
	for i := range outputs {
		outputs[i] = sdk.OutputValue{Name: fmt.Sprintf("output_%d", i), Type: "string", Value: fmt.Sprintf("val_%d", i)}
	}
	p.outputs = outputs
	p.filtered = outputs
	p.selected = 40

	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestViewFilteringMode(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs
	p.filtering = true
	p.filter = "vpc"

	view := p.View(80, 24)
	if !strings.Contains(view, "/") {
		t.Error("View in filtering mode should show '/' prompt")
	}
}

func TestOutputCount(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.OutputValue{{}, {}, {}}
	if p.OutputCount() != 3 {
		t.Errorf("OutputCount() = %d, want 3", p.OutputCount())
	}
}

func TestTotalCount(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.outputs = []sdk.OutputValue{{}, {}, {}, {}}
	if p.TotalCount() != 4 {
		t.Errorf("TotalCount() = %d, want 4", p.TotalCount())
	}
}

func TestRefresh(t *testing.T) {
	svc := &mockService{outputResult: sampleOutputs()}
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

func TestActivateWithContextChange(t *testing.T) {
	svc := &mockService{outputResult: sampleOutputs()}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveContextAbs, "/new/ctx")
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
	svc := &mockService{outputResult: sampleOutputs()}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveContextAbs, "/same")
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
	session.Set(sdk.SessionKeyContextCount, 3)
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

func TestActivateWithContextDir(t *testing.T) {
	svc := &mockService{outputResult: sampleOutputs()}
	p := New(svc).(*Plugin)
	session := sdk.NewSession()
	session.Set(sdk.SessionKeyActiveContextAbs, "/my/ctx")
	ctx := &sdk.Context{Service: svc, Session: session, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() with context dir: want non-nil cmd")
	}
}

func TestActivateNoSession(t *testing.T) {
	svc := &mockService{outputResult: sampleOutputs()}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() no session: want non-nil cmd")
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
	p.selected = 5
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

func TestFilterGetter(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.filter = "test"
	if p.Filter() != "test" {
		t.Errorf("Filter() = %q, want %q", p.Filter(), "test")
	}
}

func TestSortedOutputs(t *testing.T) {
	m := map[string]sdk.OutputValue{
		"z_output": {Name: "z_output", Value: "z"},
		"a_output": {Name: "a_output", Value: "a"},
		"m_output": {Name: "m_output", Value: "m"},
	}
	result := sortedOutputs(m)
	if len(result) != 3 {
		t.Fatalf("len(sortedOutputs) = %d, want 3", len(result))
	}
	if result[0].Name != "a_output" {
		t.Errorf("sortedOutputs[0].Name = %q, want %q", result[0].Name, "a_output")
	}
	if result[1].Name != "m_output" {
		t.Errorf("sortedOutputs[1].Name = %q, want %q", result[1].Name, "m_output")
	}
	if result[2].Name != "z_output" {
		t.Errorf("sortedOutputs[2].Name = %q, want %q", result[2].Name, "z_output")
	}
}

func TestSortedOutputsNil(t *testing.T) {
	result := sortedOutputs(nil)
	if result != nil {
		t.Errorf("sortedOutputs(nil) = %v, want nil", result)
	}
}

func TestSortedOutputsEmpty(t *testing.T) {
	result := sortedOutputs(map[string]sdk.OutputValue{})
	if len(result) != 0 {
		t.Errorf("sortedOutputs(empty) len = %d, want 0", len(result))
	}
}

func TestFilterModeGgAppendsToFilter(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.status = StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
		{Name: "c", Type: "string", Value: "3"},
	}
	p.filtered = p.outputs
	p.filtering = true

	// G in filter mode appends to filter, does not navigate
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if p.filter != "G" {
		t.Errorf("after G in filter mode: filter = %q, want %q", p.filter, "G")
	}
	if p.selected != 0 {
		t.Errorf("after G in filter mode: selected = %d, want 0 (no navigation)", p.selected)
	}
}
