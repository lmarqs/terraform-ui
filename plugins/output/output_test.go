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
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

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

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return sampleOutputs(), nil
		},
	}
	p := New(svc)

	if p.ID() != "output" {
		t.Errorf("ID() = %q, want %q", p.ID(), "output")
	}
	if p.Name() != "Outputs" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Outputs")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	if cmd := p.Init(ctx); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before data loads")
	}
}

func TestCount_WhenOutputsPresent_ShouldReturnFilteredAndTotal(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	var c sdk.Countable = p
	filtered, total := c.Count()
	if filtered != 0 || total != 0 {
		t.Errorf("Count() = (%d, %d), want (0, 0) when empty", filtered, total)
	}

	p.outputs = []sdk.OutputValue{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	p.filtered = p.outputs[:1]
	filtered, total = c.Count()
	if filtered != 1 || total != 3 {
		t.Errorf("Count() = (%d, %d), want (1, 3)", filtered, total)
	}
}

func TestActivate_WhenServiceSucceeds_ShouldReturnOutputResultMsg(t *testing.T) {
	outputs := sampleOutputs()
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return outputs, nil
		},
	}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}

	p.Init(ctx)
	cmd := p.(*Plugin).Activate()
	if cmd == nil {
		t.Fatal("Activate() returned nil cmd, want non-nil")
	}
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want tea.BatchMsg", msg)
	}
	var result OutputResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(OutputResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain OutputResultMsg")
	}
	if result.Err != nil {
		t.Errorf("OutputResultMsg.Err = %v, want nil", result.Err)
	}
	if len(result.Outputs) != 3 {
		t.Errorf("len(Outputs) = %d, want 3", len(result.Outputs))
	}
}

func TestActivate_WhenServiceFails_ShouldReturnOutputResultMsgWithError(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return nil, errors.New("output error")
		},
	}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	if cmd == nil {
		t.Fatal("Activate() returned nil cmd, want non-nil")
	}
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want tea.BatchMsg", msg)
	}
	var result OutputResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(OutputResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain OutputResultMsg")
	}
	if result.Err == nil {
		t.Error("OutputResultMsg.Err = nil, want error")
	}
}

func TestUpdate_WhenOutputResultSuccess_ShouldTransitionToDone(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusLoading

	outputs := sampleOutputs()

	result, cmd := p.Update(OutputResultMsg{Outputs: outputs, Err: nil})
	if cmd != nil {
		t.Errorf("Update(OutputResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", updated.status)
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

func TestUpdate_WhenOutputResultReceived_ShouldSortOutputsByName(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusLoading

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

func TestUpdate_WhenOutputResultError_ShouldTransitionToError(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusLoading

	result, cmd := p.Update(OutputResultMsg{Outputs: nil, Err: errors.New("load failed")})
	if cmd != nil {
		t.Errorf("Update(OutputResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want sdk.StatusError", updated.status)
	}
	if updated.errMsg != "load failed" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "load failed")
	}
}

func TestUpdate_WhenArrowKeysPressed_ShouldNavigateUpAndDown(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
		{Name: "c", Type: "string", Value: "3"},
	}
	p.filtered = p.outputs

	// Move down with arrow
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down: selected = %d, want 1", p.selected)
	}

	// Move down again
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 2 {
		t.Errorf("after down,down: selected = %d, want 2", p.selected)
	}

	// Boundary
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 2 {
		t.Errorf("after down,down,down: selected = %d, want 2 (boundary)", p.selected)
	}

	// Move up with arrow
	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 1 {
		t.Errorf("after up: selected = %d, want 1", p.selected)
	}
}

func TestUpdate_WhenJKPressed_ShouldNavigateUpAndDown(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
	}
	p.filtered = p.outputs

	// j moves down
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", p.selected)
	}

	// k moves up
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.selected != 0 {
		t.Errorf("after k: selected = %d, want 0", p.selected)
	}
}

func TestUpdate_WhenGgPressed_ShouldMoveToStartAndEnd(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
	}
	p.filtered = p.outputs

	// G moves to end
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if p.selected != 2 {
		t.Errorf("after G: selected = %d, want 2", p.selected)
	}

	// g moves to start
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if p.selected != 0 {
		t.Errorf("after g: selected = %d, want 0", p.selected)
	}
}

func TestUpdate_WhenCtrlRPressed_ShouldTriggerRefresh(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return sampleOutputs(), nil
		},
	}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone

	// ctrl+r triggers refresh in normal mode
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("after ctrl+r in sdk.StatusDone: cmd = nil, want non-nil (refresh)")
	}

	// ctrl+r works in error state too
	p.status = sdk.StatusError
	cmd = p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("after ctrl+r in sdk.StatusError: cmd = nil, want non-nil (refresh)")
	}

	// ctrl+r does nothing in Loading
	p.status = sdk.StatusLoading
	cmd = p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("after ctrl+r in sdk.StatusLoading: cmd != nil, want nil")
	}
}

func TestUpdate_WhenEscPressed_ShouldEmitDeactivateMsg(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{{Name: "a"}}
	p.filtered = p.outputs

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("after esc in normal mode: cmd = nil, want non-nil (deactivate)")
	}

	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Errorf("esc cmd returned %T, want DeactivateMsg", msg)
	}
}

func TestUpdate_WhenSlashPressed_ShouldEnterFilterMode(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
		{Name: "db_password", Type: "string", Value: "secret", Sensitive: true},
	}
	p.filtered = p.outputs

	// Enter filter mode with /
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !p.filtering {
		t.Fatal("after '/': expected filtering mode")
	}

	// Type 'v'
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	if p.filter != "v" {
		t.Errorf("after 'v': filter = %q, want %q", p.filter, "v")
	}

	// Type 'p'
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	if p.filter != "vp" {
		t.Errorf("after 'p': filter = %q, want %q", p.filter, "vp")
	}
}

func TestUpdate_WhenEscInFilterMode_ShouldExitFilterMode(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{{Name: "a"}}
	p.filtered = p.outputs
	p.filtering = true

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.filtering {
		t.Error("after esc in filter mode: filtering = true, want false")
	}
}

func TestUpdate_WhenBackspaceInFilterMode_ShouldRemoveLastChar(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs
	p.filter = "vpc"
	p.filtering = true

	p.stack.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.filter != "vp" {
		t.Errorf("after backspace: filter = %q, want %q", p.filter, "vp")
	}
}

func TestUpdate_WhenDeleteInFilterMode_ShouldRemoveLastChar(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{{Name: "a"}}
	p.filtered = p.outputs
	p.filter = "ab"
	p.filtering = true

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDelete})
	if p.filter != "a" {
		t.Errorf("after delete: filter = %q, want %q", p.filter, "a")
	}
}

func TestFilterMode_WhenTypingKeys_ShouldAppendToFilterNotTriggerHotkeys(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
		{Name: "region", Type: "string", Value: "us-east-1"},
	}
	p.filtered = p.outputs

	// Enter filter mode and type 'r' — should filter, not refresh
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if p.filter != "r" {
		t.Errorf("filter = %q, want %q", p.filter, "r")
	}
	if p.status != sdk.StatusDone {
		t.Errorf("status should remain sdk.StatusDone, got %v", p.status)
	}
}

func TestFilterMode_WhenArrowKeys_ShouldNavigateList(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
	}
	p.filtered = p.outputs
	p.filtering = true

	// down in filter mode
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.selected != 1 {
		t.Errorf("after down in filter mode: selected = %d, want 1", p.selected)
	}

	// up in filter mode
	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.selected != 0 {
		t.Errorf("after up in filter mode: selected = %d, want 0", p.selected)
	}
}

func TestFilterMode_WhenJKPressed_ShouldAppendToFilter(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
	}
	p.filtered = p.outputs
	p.filtering = true

	// j in filter mode appends to filter (only arrow keys navigate)
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if p.filter != "j" {
		t.Errorf("j in filter mode: filter = %q, want %q (appended to filter)", p.filter, "j")
	}

	// k also appends to filter
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if p.filter != "jk" {
		t.Errorf("k in filter mode: filter = %q, want %q (appended to filter)", p.filter, "jk")
	}
}

func TestNavigation_WhenMoving_ShouldUpdateSelectionWithBounds(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestNavigation_WhenMoveToStartEnd_ShouldJumpToExtremes(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestNavigation_WhenEmptyList_ShouldNotMoveSelection(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.OutputValue{}
	p.MoveToEnd()
	if p.selected != 0 {
		t.Errorf("MoveToEnd empty: selected = %d, want 0", p.selected)
	}
}

func TestSetFilter_WhenCalled_ShouldFilterOutputsByName(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestSetFilter_WhenDifferentCase_ShouldMatchCaseInsensitively(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestAppendFilter_WhenCalled_ShouldGrowFilterString(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestBackspaceFilter_WhenCalled_ShouldRemoveLastCharacter(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestFormatValue_GivenOutputValue_ShouldReturnFormattedString(t *testing.T) {
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

func TestView_WhenIdle_ShouldShowWaitingMessage(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if !strings.Contains(view, "Loading") {
		t.Errorf("view should indicate loading state, got %q", view)
	}
}

func TestView_WhenLoading_ShouldShowLoadingMessage(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading

	view := p.View(80, 24)
	if !strings.Contains(view, "Loading") || !strings.Contains(view, "outputs") {
		t.Errorf("view should indicate loading terraform outputs, got %q", view)
	}
}

func TestView_WhenError_ShouldShowErrorMessage(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "some error"

	view := p.View(80, 24)
	if !strings.Contains(view, "Error") || !strings.Contains(view, "some error") {
		t.Errorf("view should show error message 'some error', got %q", view)
	}
}

func TestView_WhenDoneWithNoOutputs_ShouldShowEmptyState(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{}
	p.filtered = []sdk.OutputValue{}

	view := p.View(80, 24)
	if !strings.Contains(view, "No outputs found") {
		t.Errorf("view should indicate no outputs found, got %q", view)
	}
}

func TestView_WhenDoneWithOutputs_ShouldRedactSensitiveValues(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-abc123", Sensitive: false},
		{Name: "db_password", Type: "string", Value: "secret", Sensitive: true},
	}
	p.filtered = p.outputs

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, with outputs) returned empty string")
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

func TestView_WhenDoneWithFilter_ShouldShowFilteredOutputs(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-123"},
	}
	p.filtered = p.outputs
	p.filter = "vpc"

	view := p.View(80, 24)
	if !strings.Contains(view, "vpc_id") {
		t.Errorf("view should contain filtered output name 'vpc_id', got %q", view)
	}
	if !strings.Contains(view, "filter") {
		t.Errorf("view should show filter indicator, got %q", view)
	}
}

func TestView_WhenFilteredDiffersFromTotal_ShouldShowFilteredCount(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
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

func TestView_WhenInvalidStatus_ShouldReturnEmptyString(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.Status(99)

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestView_WhenManyOutputs_ShouldHandleScrolling(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone

	outputs := make([]sdk.OutputValue, 50)
	for i := range outputs {
		outputs[i] = sdk.OutputValue{Name: fmt.Sprintf("output_%d", i), Type: "string", Value: fmt.Sprintf("val_%d", i)}
	}
	p.outputs = outputs
	p.filtered = outputs
	p.selected = 40

	// verifies rendering completes without panic under scrolling
	view := p.View(80, 10)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestView_WhenInFilteringMode_ShouldShowFilterPrompt(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
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

func TestOutputCount_WhenFiltered_ShouldReturnFilteredCount(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.filtered = []sdk.OutputValue{{}, {}, {}}
	if p.OutputCount() != 3 {
		t.Errorf("OutputCount() = %d, want 3", p.OutputCount())
	}
}

func TestTotalCount_WhenOutputsLoaded_ShouldReturnAllOutputsCount(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.outputs = []sdk.OutputValue{{}, {}, {}, {}}
	if p.TotalCount() != 4 {
		t.Errorf("TotalCount() = %d, want 4", p.TotalCount())
	}
}

func TestRefresh_WhenCalled_ShouldResetStateAndReturnCmd(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return sampleOutputs(), nil
		},
	}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.selected = 5
	p.filter = "something"

	cmd := p.Refresh()
	if cmd == nil {
		t.Error("Refresh() returned nil cmd")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("after Refresh: status = %v, want sdk.StatusLoading", p.status)
	}
	if p.selected != 0 {
		t.Errorf("after Refresh: selected = %d, want 0", p.selected)
	}
	if p.filter != "" {
		t.Errorf("after Refresh: filter = %q, want empty", p.filter)
	}
}

func TestUpdate_WhenUnknownMsg_ShouldReturnSelfWithNoCmd(t *testing.T) {
	svc := &sdktest.MockService{}
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

func TestHandleChdirChanged_WhenCalled_ShouldResetToIdleWithNewContext(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return sampleOutputs(), nil
		},
	}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	p.status = sdk.StatusDone
	p.scopedContext = "/old/ctx"
	p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/ctx"})
	if p.scopedContext != "/new/ctx" {
		t.Errorf("scopedContext = %q, want %q", p.scopedContext, "/new/ctx")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle after HandleChdirChanged", p.status)
	}
	// Activate should now trigger loading since status is Idle
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() after HandleChdirChanged: want non-nil cmd")
	}
}

func TestActivate_WhenSameContextAlreadyDone_ShouldReturnNilCmd(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return sampleOutputs(), nil
		},
	}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	p.status = sdk.StatusDone
	p.scopedContext = "/same"
	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() same context done: want nil")
	}
}

func TestActivate_WhenNoContextSelected_ShouldProceedWithLoading(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	// Without ChdirGuard, Activate proceeds with loading (no scope gating)
	if cmd == nil {
		t.Error("Activate() multi-context no selection: want non-nil cmd (loads outputs)")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want sdk.StatusLoading", p.status)
	}
}

func TestActivate_WhenScopeDirSet_ShouldReturnLoadCmd(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return sampleOutputs(), nil
		},
	}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() with context dir: want non-nil cmd")
	}
}

func TestActivate_WhenNoSession_ShouldReturnLoadCmd(t *testing.T) {
	svc := &sdktest.MockService{
		OutputFn: func(_ context.Context) (map[string]sdk.OutputValue, error) {
			return sampleOutputs(), nil
		},
	}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() no session: want non-nil cmd")
	}
}

func TestStatus_WhenNew_ShouldReturnIdle(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want sdk.StatusIdle", p.Status())
	}
}

func TestSelected_WhenSet_ShouldReturnCurrentIndex(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.selected = 5
	if p.Selected() != 5 {
		t.Errorf("Selected() = %d, want 5", p.Selected())
	}
}

func TestFiltering_WhenNotInFilterMode_ShouldReturnFalse(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	if p.Filtering() {
		t.Error("Filtering() = true, want false")
	}
}

func TestFilter_WhenSet_ShouldReturnFilterString(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.filter = "test"
	if p.Filter() != "test" {
		t.Errorf("Filter() = %q, want %q", p.Filter(), "test")
	}
}

func TestSortedOutputs_WhenMapGiven_ShouldReturnAlphabeticalSlice(t *testing.T) {
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

func TestSortedOutputs_WhenNil_ShouldReturnNil(t *testing.T) {
	result := sortedOutputs(nil)
	if result != nil {
		t.Errorf("sortedOutputs(nil) = %v, want nil", result)
	}
}

func TestSortedOutputs_WhenEmpty_ShouldReturnEmptySlice(t *testing.T) {
	result := sortedOutputs(map[string]sdk.OutputValue{})
	if len(result) != 0 {
		t.Errorf("sortedOutputs(empty) len = %d, want 0", len(result))
	}
}

func TestFilterMode_WhenGPressed_ShouldAppendToFilterNotNavigate(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
		{Name: "c", Type: "string", Value: "3"},
	}
	p.filtered = p.outputs
	p.filtering = true

	// G in filter mode appends to filter, does not navigate
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if p.filter != "G" {
		t.Errorf("after G in filter mode: filter = %q, want %q", p.filter, "G")
	}
	if p.selected != 0 {
		t.Errorf("after G in filter mode: selected = %d, want 0 (no navigation)", p.selected)
	}
}

func TestStack_WhenCalled_ShouldReturnStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	s := p.Stack()
	if s == nil {
		t.Fatal("Stack() returned nil")
	}
	if s != p.stack {
		t.Error("Stack() returned different instance than internal stack")
	}
}

func TestListFrame_WhenCreated_ShouldHaveCorrectID(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	frame := p.stack.Peek()
	if frame == nil {
		t.Fatal("stack is empty, expected listFrame")
	}
	if frame.ID() != "list" {
		t.Errorf("listFrame.ID() = %q, want %q", frame.ID(), "list")
	}
}

func TestListFrame_WhenViewCalled_ShouldDelegateToPlugin(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-abc123"},
	}
	p.filtered = p.outputs

	view := p.stack.View(80, 24)
	if view == "" {
		t.Error("stack.View() returned empty string")
	}
	if !strings.Contains(view, "vpc-abc123") {
		t.Error("stack.View() should contain output value")
	}
}

func TestListFrame_WhenFilteringHints_ShouldShowCancel(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.filtering = true

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in filtering mode")
	}
	found := false
	for _, h := range hints {
		if h.Key == "Esc" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Hints() in filtering mode should include Esc/cancel hint")
	}
}

func TestListFrame_WhenErrorHints_ShouldShowRetryAndBack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in error state")
	}
}

func TestListFrame_WhenDoneHints_ShouldShowFilterRefreshBack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in done state")
	}
}

func TestListFrame_WhenLoadingHints_ShouldShowBack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice in loading state")
	}
}

func TestFilterMode_WhenSlashPressed_ShouldBeNoOp(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
	}
	p.filtered = p.outputs
	p.filtering = true
	p.filter = "test"

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if p.filter != "test" {
		t.Errorf("after / in filter mode: filter = %q, want %q (no-op)", p.filter, "test")
	}
}

func TestRenderOutputs_WhenNarrowWidth_ShouldUseMinimumContentWidth(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Type: "string", Value: "vpc-abc123"},
	}
	p.filtered = p.outputs

	view := p.View(20, 24)
	if view == "" {
		t.Error("View with narrow width returned empty string")
	}
	if !strings.Contains(view, "vpc-abc123") {
		t.Error("View with narrow width should still show output value")
	}
}

func TestRenderOutputs_WhenSmallHeight_ShouldUseMinimumVisibleLines(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
		{Name: "c", Type: "string", Value: "3"},
		{Name: "d", Type: "string", Value: "4"},
		{Name: "e", Type: "string", Value: "5"},
	}
	p.filtered = p.outputs

	view := p.View(80, 3)
	if view == "" {
		t.Error("View with small height returned empty string")
	}
}

func TestListFrame_WhenNonKeyMsg_ShouldReturnSelf(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	cmd := p.stack.Update(OutputResultMsg{})
	if cmd != nil {
		t.Error("non-key message through stack should return nil cmd")
	}
}

func TestOutput_WhenJsonTrue_ShouldReturnJSONMap(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Value: "vpc-abc123", Type: "string", Sensitive: false},
		{Name: "db_password", Value: "secret", Type: "string", Sensitive: true},
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"vpc_id"`) {
		t.Error("JSON should contain vpc_id key")
	}
	if !strings.Contains(s, `"sensitive": true`) {
		t.Error("JSON should contain sensitive: true")
	}
}

func TestOutput_WhenJsonTrueEmpty_ShouldReturnEmptyObject(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outputs = []sdk.OutputValue{}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	if !strings.Contains(string(data), "{}") {
		t.Errorf("JSON for empty = %q, want '{}'", string(data))
	}
}

func TestOutput_WhenJsonFalse_ShouldReturnKeyValueText(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outputs = []sdk.OutputValue{
		{Name: "vpc_id", Value: "vpc-abc123", Type: "string", Sensitive: false},
		{Name: "db_password", Value: "secret", Type: "string", Sensitive: true},
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "vpc_id = vpc-abc123") {
		t.Error("text should contain 'vpc_id = vpc-abc123'")
	}
	if !strings.Contains(s, "db_password = (sensitive)") {
		t.Error("text should redact sensitive as '(sensitive)'")
	}
}

func TestOutput_WhenJsonFalseEmpty_ShouldReturnEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outputs = []sdk.OutputValue{}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	if len(data) != 0 {
		t.Errorf("text for empty = %q, want empty", string(data))
	}
}

func TestOutput_WhenJsonTrueNilOutputs_ShouldReturnEmptyObject(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outputs = nil

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	if !strings.Contains(string(data), "{}") {
		t.Errorf("JSON for nil outputs = %q, want '{}'", string(data))
	}
}

func TestOutput_WhenJsonFalseNilOutputs_ShouldReturnEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.outputs = nil

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	if len(data) != 0 {
		t.Errorf("text for nil outputs = %q, want empty", string(data))
	}
}

func TestUpdate_WhenUnhandledMsgType_ShouldPassThrough(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone

	type customMsg struct{}
	result, cmd := p.Update(customMsg{})
	if cmd != nil {
		t.Error("unhandled msg should return nil cmd")
	}
	if result.(*Plugin) != p {
		t.Error("unhandled msg should return same plugin")
	}
}

func TestUpdate_WhenTimerTickMsg_ShouldReturnTickCmd(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	p.timer.Start()

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd == nil {
		t.Error("TimerTickMsg while timer running: cmd = nil, want non-nil")
	}
}

func TestUpdate_WhenTimerTickMsgNotRunning_ShouldReturnNilCmd(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd != nil {
		t.Error("TimerTickMsg while timer stopped: cmd != nil, want nil")
	}
}

func TestCursorPosition_WhenDoneWithOutputs_ShouldReturnOneBasedPositionAndTotal(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.outputs = []sdk.OutputValue{
		{Name: "a", Type: "string", Value: "1"},
		{Name: "b", Type: "string", Value: "2"},
		{Name: "c", Type: "string", Value: "3"},
	}
	p.filtered = p.outputs
	p.selected = 0

	pos, total := p.CursorPosition()
	if pos != 1 || total != 3 {
		t.Errorf("CursorPosition() = (%d, %d), want (1, 3)", pos, total)
	}

	p.selected = 2
	pos, total = p.CursorPosition()
	if pos != 3 || total != 3 {
		t.Errorf("CursorPosition() after move = (%d, %d), want (3, 3)", pos, total)
	}
}

func TestCursorPosition_WhenNotDoneOrEmpty_ShouldReturnZeros(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	pos, total := p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() idle = (%d, %d), want (0, 0)", pos, total)
	}

	p.status = sdk.StatusDone
	p.filtered = []sdk.OutputValue{}
	pos, total = p.CursorPosition()
	if pos != 0 || total != 0 {
		t.Errorf("CursorPosition() done+empty = (%d, %d), want (0, 0)", pos, total)
	}
}
