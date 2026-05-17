package validate

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
)

func TestNew(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "validate" {
		t.Errorf("ID() = %q, want %q", p.ID(), "validate")
	}
	if p.Name() != "Validate" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Validate")
	}
	if p.Description() != "Run terraform validate" {
		t.Errorf("Description() = %q, want %q", p.Description(), "Run terraform validate")
	}
	if p.Ready() {
		t.Error("Ready() = true before data loads, want false")
	}
}

func TestConfigure(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	err := p.Configure(map[string]interface{}{"key": "value"})
	if err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
}

func TestInit(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Workspace:  "default",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	cmd := p.Init(ctx)
	if cmd != nil {
		t.Error("Init() should return nil cmd")
	}

	pp := p.(*Plugin)
	if pp.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle", pp.status)
	}
}

func TestActivate(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	ctx := &sdk.Context{
		WorkingDir: "/tmp",
		Service:    svc,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)

	pp := p.(*Plugin)
	cmd := pp.Activate()
	if cmd == nil {
		t.Error("Activate() returned nil cmd, want non-nil")
	}
	if pp.status != sdk.StatusLoading {
		t.Errorf("status = %v, want sdk.StatusLoading", pp.status)
	}
}

func TestActivateCmdReturnsValidateResultMsg(t *testing.T) {
	diags := []sdk.Diagnostic{
		{Severity: "error", Summary: "Missing required argument", File: "main.tf", Line: 10},
	}
	svc := &sdktest.MockService{
		ValidateFn: func(_ context.Context) ([]sdk.Diagnostic, error) {
			return diags, nil
		},
	}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want tea.BatchMsg", msg)
	}
	var result ValidateResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(ValidateResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain ValidateResultMsg")
	}
	if result.Err != nil {
		t.Errorf("ValidateResultMsg.Err = %v, want nil", result.Err)
	}
	if len(result.Diagnostics) != 1 {
		t.Errorf("len(Diagnostics) = %d, want 1", len(result.Diagnostics))
	}
}

func TestActivateCmdReturnsError(t *testing.T) {
	svc := &sdktest.MockService{
		ValidateFn: func(_ context.Context) ([]sdk.Diagnostic, error) {
			return nil, errors.New("validate failed")
		},
	}
	p := New(svc)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)

	cmd := p.(*Plugin).Activate()
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("Activate cmd returned %T, want tea.BatchMsg", msg)
	}
	var result ValidateResultMsg
	found := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		if r, ok := subCmd().(ValidateResultMsg); ok {
			result = r
			found = true
		}
	}
	if !found {
		t.Fatal("batch did not contain ValidateResultMsg")
	}
	if result.Err == nil {
		t.Error("ValidateResultMsg.Err = nil, want error")
	}
}

func TestUpdateValidateResultSuccess(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusLoading

	diags := []sdk.Diagnostic{
		{Severity: "warning", Summary: "Deprecated attribute"},
		{Severity: "error", Summary: "Invalid reference"},
	}

	result, cmd := p.Update(ValidateResultMsg{Diagnostics: diags, Err: nil})
	if cmd != nil {
		t.Errorf("Update(ValidateResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", updated.status)
	}
	if updated.diagnostics == nil {
		t.Error("diagnostics = nil, want non-nil")
	}
	if !updated.Ready() {
		t.Error("Ready() = false after success, want true")
	}
}

func TestUpdateValidateResultErrorsSortedFirst(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusLoading

	diags := []sdk.Diagnostic{
		{Severity: "warning", Summary: "Deprecated attribute"},
		{Severity: "error", Summary: "Invalid reference"},
		{Severity: "warning", Summary: "Another warning"},
		{Severity: "error", Summary: "Missing argument"},
	}

	result, _ := p.Update(ValidateResultMsg{Diagnostics: diags, Err: nil})
	updated := result.(*Plugin)

	// Errors should come first
	if updated.diagnostics[0].Severity != "error" {
		t.Errorf("diagnostics[0].Severity = %q, want \"error\"", updated.diagnostics[0].Severity)
	}
	if updated.diagnostics[1].Severity != "error" {
		t.Errorf("diagnostics[1].Severity = %q, want \"error\"", updated.diagnostics[1].Severity)
	}
	if updated.diagnostics[2].Severity != "warning" {
		t.Errorf("diagnostics[2].Severity = %q, want \"warning\"", updated.diagnostics[2].Severity)
	}
	if updated.diagnostics[3].Severity != "warning" {
		t.Errorf("diagnostics[3].Severity = %q, want \"warning\"", updated.diagnostics[3].Severity)
	}
}

func TestUpdateValidateResultError(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusLoading

	result, cmd := p.Update(ValidateResultMsg{Diagnostics: nil, Err: errors.New("terraform error")})
	if cmd != nil {
		t.Errorf("Update(ValidateResultMsg) cmd = %v, want nil", cmd)
	}

	updated := result.(*Plugin)
	if updated.status != sdk.StatusError {
		t.Errorf("status = %v, want sdk.StatusError", updated.status)
	}
	if updated.errMsg != "terraform error" {
		t.Errorf("errMsg = %q, want %q", updated.errMsg, "terraform error")
	}
}

func TestUpdateValidateResultZeroDiagnostics(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusLoading

	result, _ := p.Update(ValidateResultMsg{Diagnostics: []sdk.Diagnostic{}, Err: nil})
	updated := result.(*Plugin)

	if updated.status != sdk.StatusDone {
		t.Errorf("status = %v, want sdk.StatusDone", updated.status)
	}
	if len(updated.diagnostics) != 0 {
		t.Errorf("len(diagnostics) = %d, want 0", len(updated.diagnostics))
	}
}

func TestUpdateKeyMsgNavigation(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusDone
	pp.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "a"},
		{Severity: "error", Summary: "b"},
		{Severity: "warning", Summary: "c"},
	}

	// Move down
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if pp.selected != 1 {
		t.Errorf("after j: selected = %d, want 1", pp.selected)
	}

	// Move down again
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if pp.selected != 2 {
		t.Errorf("after j,j: selected = %d, want 2", pp.selected)
	}

	// Move down at boundary
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if pp.selected != 2 {
		t.Errorf("after j,j,j: selected = %d, want 2 (boundary)", pp.selected)
	}

	// Move up
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if pp.selected != 1 {
		t.Errorf("after k: selected = %d, want 1", pp.selected)
	}

	// Move up to start
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if pp.selected != 0 {
		t.Errorf("after k,k: selected = %d, want 0", pp.selected)
	}

	// Move up at boundary
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if pp.selected != 0 {
		t.Errorf("after k,k,k: selected = %d, want 0 (boundary)", pp.selected)
	}
}

func TestUpdateKeyMsgMoveToEndAndStart(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusDone
	pp.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "a"},
		{Severity: "error", Summary: "b"},
		{Severity: "warning", Summary: "c"},
	}

	// G moves to end
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if pp.selected != 2 {
		t.Errorf("after G: selected = %d, want 2", pp.selected)
	}

	// g moves to start
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if pp.selected != 0 {
		t.Errorf("after g: selected = %d, want 0", pp.selected)
	}
}

func TestUpdateKeyMsgToggleExpand(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusDone
	pp.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "Invalid reference", Detail: "The resource does not exist"},
	}

	// Toggle expand with enter
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !pp.IsExpanded(0) {
		t.Error("after enter: expanded[0] = false, want true")
	}

	// Toggle again
	p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if pp.IsExpanded(0) {
		t.Error("after enter,enter: expanded[0] = true, want false")
	}

	// Toggle with i
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if !pp.IsExpanded(0) {
		t.Error("after i: expanded[0] = false, want true")
	}
}

func TestUpdateKeyMsgRefresh(t *testing.T) {
	svc := &sdktest.MockService{
		ValidateFn: func(_ context.Context) ([]sdk.Diagnostic, error) {
			return []sdk.Diagnostic{}, nil
		},
	}
	p := New(svc)
	pp := p.(*Plugin)
	pp.status = sdk.StatusDone

	// ctrl+r triggers refresh when status is Done
	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("after ctrl+r in sdk.StatusDone: cmd = nil, want non-nil (refresh)")
	}

	// ctrl+r triggers refresh when status is Error
	pp.status = sdk.StatusError
	_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Error("after ctrl+r in sdk.StatusError: cmd = nil, want non-nil (refresh)")
	}

	// ctrl+r does nothing when Loading
	pp.status = sdk.StatusLoading
	_, cmd = p.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("after ctrl+r in sdk.StatusLoading: cmd != nil, want nil")
	}
}

func TestUpdateUnknownMsg(t *testing.T) {
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

func TestMoveUpDown(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "a"},
		{Severity: "error", Summary: "b"},
	}

	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown: selected = %d, want 1", p.selected)
	}
	p.MoveDown()
	if p.selected != 1 {
		t.Errorf("MoveDown (boundary): selected = %d, want 1", p.selected)
	}
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp: selected = %d, want 0", p.selected)
	}
	p.MoveUp()
	if p.selected != 0 {
		t.Errorf("MoveUp (boundary): selected = %d, want 0", p.selected)
	}
}

func TestMoveDownNilDiagnostics(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.diagnostics = nil
	p.MoveDown()
	if p.selected != 0 {
		t.Errorf("MoveDown nil diagnostics: selected = %d, want 0", p.selected)
	}
}

func TestMoveToStartEnd(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "a"},
		{Severity: "error", Summary: "b"},
		{Severity: "warning", Summary: "c"},
	}

	p.MoveToEnd()
	if p.selected != 2 {
		t.Errorf("MoveToEnd: selected = %d, want 2", p.selected)
	}
	p.MoveToStart()
	if p.selected != 0 {
		t.Errorf("MoveToStart: selected = %d, want 0", p.selected)
	}
}

func TestMoveToEndNilDiagnostics(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.diagnostics = nil
	p.MoveToEnd()
	if p.selected != 0 {
		t.Errorf("MoveToEnd nil diagnostics: selected = %d, want 0", p.selected)
	}
}

func TestMoveToEndEmptyDiagnostics(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.diagnostics = []sdk.Diagnostic{}
	p.MoveToEnd()
	if p.selected != 0 {
		t.Errorf("MoveToEnd empty: selected = %d, want 0", p.selected)
	}
}

func TestToggleExpand(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.selected = 2

	p.ToggleExpand()
	if !p.IsExpanded(2) {
		t.Error("ToggleExpand: IsExpanded(2) = false, want true")
	}
	p.ToggleExpand()
	if p.IsExpanded(2) {
		t.Error("ToggleExpand: IsExpanded(2) = true, want false")
	}
}

func TestIsExpanded(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	if p.IsExpanded(0) {
		t.Error("IsExpanded(0) = true before toggle, want false")
	}
}

func TestRefresh(t *testing.T) {
	svc := &sdktest.MockService{
		ValidateFn: func(_ context.Context) ([]sdk.Diagnostic, error) {
			return []sdk.Diagnostic{}, nil
		},
	}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.selected = 5
	p.expander.Toggle(0)

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
}

func TestViewIdle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusIdle

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusIdle) returned empty string")
	}
}

func TestViewLoading(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusLoading

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusLoading) returned empty string")
	}
}

func TestViewError(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusError
	p.errMsg = "some error"

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusError) returned empty string")
	}
}

func TestViewDoneNoDiagnostics(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, no diagnostics) returned empty string")
	}
	if !strings.Contains(view, "Configuration is valid") {
		t.Error("View(sdk.StatusDone, no diagnostics) should contain success message")
	}
}

func TestViewDoneWithDiagnostics(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "Invalid reference", File: "main.tf", Line: 10, Detail: "Resource not found"},
		{Severity: "warning", Summary: "Deprecated attribute", File: "vars.tf", Line: 5},
		{Severity: "error", Summary: "Missing argument"},
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(sdk.StatusDone, with diagnostics) returned empty string")
	}
}

func TestViewDoneWithExpandedDetail(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "Invalid reference", Detail: "The resource aws_instance.foo does not exist"},
	}
	p.expander.Toggle(0)

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with expanded detail returned empty string")
	}
}

func TestViewDoneScrolling(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone

	// Create many diagnostics to trigger scrolling
	diags := make([]sdk.Diagnostic, 50)
	for i := range diags {
		diags[i] = sdk.Diagnostic{Severity: "error", Summary: fmt.Sprintf("error %d", i)}
	}
	p.diagnostics = diags
	p.selected = 45

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with scrolling returned empty string")
	}
}

func TestViewDefaultStatus(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.Status(99) // invalid status

	view := p.View(80, 24)
	if view != "" {
		t.Errorf("View(invalid status) = %q, want empty", view)
	}
}

func TestViewSmallHeight(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "Invalid reference"},
	}

	view := p.View(80, 5)
	if view == "" {
		t.Error("View with small height returned empty string")
	}
}

func TestSortDiagnostics(t *testing.T) {
	diags := []sdk.Diagnostic{
		{Severity: "warning", Summary: "w1"},
		{Severity: "error", Summary: "e1"},
		{Severity: "warning", Summary: "w2"},
		{Severity: "error", Summary: "e2"},
	}

	sorted := sortDiagnostics(diags)
	if len(sorted) != 4 {
		t.Fatalf("len(sorted) = %d, want 4", len(sorted))
	}
	if sorted[0].Summary != "e1" {
		t.Errorf("sorted[0].Summary = %q, want %q", sorted[0].Summary, "e1")
	}
	if sorted[1].Summary != "e2" {
		t.Errorf("sorted[1].Summary = %q, want %q", sorted[1].Summary, "e2")
	}
	if sorted[2].Summary != "w1" {
		t.Errorf("sorted[2].Summary = %q, want %q", sorted[2].Summary, "w1")
	}
	if sorted[3].Summary != "w2" {
		t.Errorf("sorted[3].Summary = %q, want %q", sorted[3].Summary, "w2")
	}
}

func TestSortDiagnosticsNil(t *testing.T) {
	sorted := sortDiagnostics(nil)
	if sorted != nil {
		t.Errorf("sortDiagnostics(nil) = %v, want nil", sorted)
	}
}

func TestSortDiagnosticsEmpty(t *testing.T) {
	sorted := sortDiagnostics([]sdk.Diagnostic{})
	if len(sorted) != 0 {
		t.Errorf("len(sorted) = %d, want 0", len(sorted))
	}
}

func TestDiagnostics(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	if p.Diagnostics() != nil {
		t.Error("Diagnostics() = non-nil before validate, want nil")
	}

	p.diagnostics = []sdk.Diagnostic{{Severity: "error", Summary: "test"}}
	if len(p.Diagnostics()) != 1 {
		t.Errorf("Diagnostics() len = %d, want 1", len(p.Diagnostics()))
	}
}

func TestStatus(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	if p.Status() != sdk.StatusIdle {
		t.Errorf("Status() = %v, want sdk.StatusIdle", p.Status())
	}
}

func TestSelected(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.selected = 5

	if p.Selected() != 5 {
		t.Errorf("Selected() = %d, want 5", p.Selected())
	}
}

func TestViewDiagnosticWithFileNoLine(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "warning", Summary: "Deprecated", File: "main.tf", Line: 0},
	}

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with file but no line returned empty string")
	}
}

func TestHints_WhenIdle_ShouldReturnConfirmAndBack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusIdle

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice for Idle status")
	}
	descs := hintDescs(hints)
	assertContains(t, descs, "confirm")
	assertContains(t, descs, "back")
}

func TestHints_WhenLoading_ShouldReturnBackOnly(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusLoading

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice for Loading status")
	}
	descs := hintDescs(hints)
	assertContains(t, descs, "back")
}

func TestHints_WhenError_ShouldReturnRetryAndBack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusError

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice for Error status")
	}
	descs := hintDescs(hints)
	assertContains(t, descs, "retry")
	assertContains(t, descs, "back")
}

func TestHints_WhenDoneWithDiagnostics_ShouldReturnInspectRefreshBack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "some error"},
	}

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice for Done status with diagnostics")
	}
	descs := hintDescs(hints)
	assertContains(t, descs, "inspect")
	assertContains(t, descs, "refresh")
	assertContains(t, descs, "back")
}

func TestHints_WhenDoneWithoutDiagnostics_ShouldReturnRefreshAndBack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{}

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice for Done status without diagnostics")
	}
	descs := hintDescs(hints)
	assertContains(t, descs, "refresh")
	assertContains(t, descs, "back")
	assertNotContains(t, descs, "inspect")
}

func TestHints_WhenUnknownStatus_ShouldReturnBackOnly(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.Status(99)

	hints := p.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty slice for unknown status")
	}
	descs := hintDescs(hints)
	assertContains(t, descs, "back")
}

func TestHandleChdirChanged_ShouldResetStateAndUpdateService(t *testing.T) {
	svc := &sdktest.MockService{
		ValidateFn: func(_ context.Context) ([]sdk.Diagnostic, error) {
			return []sdk.Diagnostic{{Severity: "error", Summary: "old error"}}, nil
		},
	}
	p := New(svc).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{{Severity: "error", Summary: "old"}}
	p.errMsg = "old error"
	p.selected = 3
	p.expander.Toggle(0)

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{
		RelPath: "modules/vpc",
		AbsPath: "/project/modules/vpc",
	})

	if cmd != nil {
		t.Error("HandleChdirChanged() should return nil cmd")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want sdk.StatusIdle", p.status)
	}
	if p.diagnostics != nil {
		t.Errorf("diagnostics = %v, want nil", p.diagnostics)
	}
	if p.errMsg != "" {
		t.Errorf("errMsg = %q, want empty", p.errMsg)
	}
	if p.selected != 0 {
		t.Errorf("selected = %d, want 0", p.selected)
	}
	if p.IsExpanded(0) {
		t.Error("expanded[0] = true, want false after reset")
	}
	if p.scopedContext != "/project/modules/vpc" {
		t.Errorf("scopedContext = %q, want %q", p.scopedContext, "/project/modules/vpc")
	}
}

func TestHandleChdirChanged_ShouldCallWithDir(t *testing.T) {
	originalSvc := &sdktest.MockService{}
	p := New(originalSvc).(*Plugin)
	ctx := &sdk.Context{
		Service: originalSvc,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)

	p.HandleChdirChanged(sdk.ChdirChangedEvent{
		RelPath: "modules/vpc",
		AbsPath: "/project/modules/vpc",
	})

	if p.svc == nil {
		t.Fatal("svc should not be nil after HandleChdirChanged")
	}
}

func TestActivate_WhenAlreadyLoading_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		Service: svc,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)
	p.status = sdk.StatusLoading

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() when Loading should return nil cmd")
	}
}

func TestActivate_WhenDone_ShouldReturnNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		Service: svc,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)
	p.status = sdk.StatusDone

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() when Done should return nil cmd")
	}
}

func TestActivate_WhenError_ShouldRetriggerValidation(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		Service: svc,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	p.Init(ctx)
	p.status = sdk.StatusError

	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() when Error should return non-nil cmd")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want sdk.StatusLoading", p.status)
	}
}

func hintDescs(hints []sdk.KeyHint) []string {
	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Description
	}
	return descs
}

func assertContains(t *testing.T, descs []string, want string) {
	t.Helper()
	for _, d := range descs {
		if d == want {
			return
		}
	}
	t.Errorf("expected hints to contain %q, got %v", want, descs)
}

func assertNotContains(t *testing.T, descs []string, notWant string) {
	t.Helper()
	for _, d := range descs {
		if d == notWant {
			t.Errorf("expected hints to NOT contain %q, got %v", notWant, descs)
			return
		}
	}
}

func TestOutput_WhenJsonTrueWithDiagnostics_ShouldReturnJSON(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "Missing arg", Detail: "Required", File: "main.tf", Line: 10},
		{Severity: "warning", Summary: "Deprecated"},
	}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"valid": false`) {
		t.Error("JSON should contain valid: false")
	}
	if !strings.Contains(s, `"error_count": 1`) {
		t.Error("JSON should contain error_count: 1")
	}
	if !strings.Contains(s, `"warning_count": 1`) {
		t.Error("JSON should contain warning_count: 1")
	}
	if !strings.Contains(s, `"Missing arg"`) {
		t.Error("JSON should contain diagnostic summary")
	}
}

func TestOutput_WhenJsonTrueNoDiagnostics_ShouldReturnValid(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{}

	data, err := p.Output(true)
	if err != nil {
		t.Fatalf("Output(true) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"valid": true`) {
		t.Error("JSON should contain valid: true")
	}
}

func TestOutput_WhenJsonFalseWithDiagnostics_ShouldReturnText(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "error", Summary: "Invalid ref", File: "main.tf", Line: 5, Detail: "Not found"},
		{Severity: "warning", Summary: "Deprecated", File: "vars.tf"},
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "Invalid ref") {
		t.Error("text should contain error summary")
	}
	if !strings.Contains(s, "main.tf:5") {
		t.Error("text should contain file:line")
	}
	if !strings.Contains(s, "Not found") {
		t.Error("text should contain detail")
	}
	if !strings.Contains(s, "Deprecated") {
		t.Error("text should contain warning summary")
	}
}

func TestOutput_WhenJsonFalseNoDiagnostics_ShouldReturnValid(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	if !strings.Contains(string(data), "Configuration is valid") {
		t.Error("text should contain 'Configuration is valid'")
	}
}

func TestExitCode_WhenNoErrors_ShouldReturnZero(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.diagnostics = []sdk.Diagnostic{{Severity: "warning", Summary: "x"}}
	if code := p.ExitCode(); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
}

func TestExitCode_WhenErrorsPresent_ShouldReturnOne(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.diagnostics = []sdk.Diagnostic{{Severity: "error", Summary: "x"}}
	if code := p.ExitCode(); code != 1 {
		t.Errorf("ExitCode() = %d, want 1", code)
	}
}

func TestExitCode_WhenNilDiagnostics_ShouldReturnZero(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.diagnostics = nil
	if code := p.ExitCode(); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
}

func TestCancel_WhenNil_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.cancelFn = nil
	p.Cancel()
}

func TestCancel_WhenSet_ShouldCallAndClear(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	called := false
	p.cancelFn = func() { called = true }
	p.Cancel()
	if !called {
		t.Error("Cancel() should call cancelFn")
	}
	if p.cancelFn != nil {
		t.Error("Cancel() should set cancelFn to nil")
	}
}

func TestUpdate_WhenUnhandledMsg_ShouldReturnSelfAndNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	type customMsg struct{}
	result, cmd := p.Update(customMsg{})
	if cmd != nil {
		t.Error("unhandled msg should return nil cmd")
	}
	if result.(*Plugin) != p {
		t.Error("unhandled msg should return same plugin")
	}
}

func TestOutput_WhenTextWithFileNoLine_ShouldFormatCorrectly(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{Service: svc, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	p.Init(ctx)
	p.status = sdk.StatusDone
	p.diagnostics = []sdk.Diagnostic{
		{Severity: "warning", Summary: "deprecated", File: "main.tf", Line: 0},
	}

	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output(false) error = %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "(main.tf)") {
		t.Errorf("expected '(main.tf)' without line number, got %q", s)
	}
	if strings.Contains(s, "main.tf:0") {
		t.Error("should not show line number 0")
	}
}
