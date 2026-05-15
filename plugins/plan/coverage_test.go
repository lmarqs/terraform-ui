package plan

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func newTestPlugin(svc sdk.Service) *Plugin {
	p := New(svc).(*Plugin)
	ctx := &sdk.Context{
		Service: svc,
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Pins:    sdk.NewPinService(),
	}
	p.Init(ctx)
	return p
}

func TestPlugin_WhenCreated_ShouldExposeStack(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if p.Stack() == nil {
		t.Error("Stack() = nil, want non-nil")
	}
	if p.Stack().Depth() != 1 {
		t.Errorf("Stack().Depth() = %d, want 1", p.Stack().Depth())
	}
}

func TestPlugin_WhenCreated_ShouldReportNotBusy(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	if p.Busy() {
		t.Error("Busy() = true, want false when status is Idle")
	}
}

func TestPlugin_WhenLoading_ShouldReportBusy(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusLoading
	if !p.Busy() {
		t.Error("Busy() = false, want true when status is Loading")
	}
}

func TestPlugin_WhenDone_ShouldReportNotBusy(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	p.status = sdk.StatusDone
	if p.Busy() {
		t.Error("Busy() = true, want false when status is Done")
	}
}

func TestPlugin_WhenChdirChanged_ShouldResetState(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}}}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.errMsg = "old error"

	cmd := p.HandleChdirChanged(sdk.ChdirChangedEvent{
		RelPath: "modules/vpc",
		AbsPath: "/projects/infra/modules/vpc",
	})

	if cmd != nil {
		t.Error("HandleChdirChanged() cmd != nil, want nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle", p.status)
	}
	if p.summary != nil {
		t.Error("summary != nil after reset")
	}
	if p.tree.Cursor() != 0 {
		t.Error("cursor != 0 after reset")
	}
	if p.errMsg != "" {
		t.Errorf("errMsg = %q, want empty", p.errMsg)
	}
	if p.scopedContext != "/projects/infra/modules/vpc" {
		t.Errorf("scopedContext = %q, want %q", p.scopedContext, "/projects/infra/modules/vpc")
	}
}

func TestPlugin_WhenPlanInvalidated_ShouldResetState(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{{Resource: sdk.Resource{Address: "a"}}}}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.errMsg = "something"

	cmd := p.HandlePlanInvalidated(sdk.PlanInvalidatedEvent{})

	if cmd != nil {
		t.Error("HandlePlanInvalidated() cmd != nil, want nil")
	}
	if p.status != sdk.StatusIdle {
		t.Errorf("status = %v, want Idle", p.status)
	}
	if p.summary != nil {
		t.Error("summary != nil after reset")
	}
	if p.tree.Cursor() != 0 {
		t.Error("cursor != 0 after reset")
	}
	if p.errMsg != "" {
		t.Errorf("errMsg = %q, want empty", p.errMsg)
	}
}

func TestPlugin_WhenActivatedWhileLoading_ShouldReturnNil(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() while loading returned non-nil cmd, want nil")
	}
}

func TestPlugin_WhenActivatedWhileDone_ShouldReturnNil(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone

	cmd := p.Activate()
	if cmd != nil {
		t.Error("Activate() while done returned non-nil cmd, want nil")
	}
}

func TestPlugin_WhenActivatedWhileError_ShouldRetriggerPlan(t *testing.T) {
	svc := &mockService{planResult: &sdk.PlanSummary{}}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError

	cmd := p.Activate()
	if cmd == nil {
		t.Error("Activate() while error returned nil cmd, want non-nil")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("status = %v, want Loading", p.status)
	}
}

func TestPlugin_WhenPlanResultNilSummary_ShouldNotEmitPlanCompletedEvent(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusLoading

	_, cmd := p.Update(PlanResultMsg{Summary: nil, Err: nil})
	if cmd == nil {
		t.Fatal("Update(PlanResultMsg) cmd = nil, want StateRefreshedEvent cmd")
	}
	msg := cmd()
	if _, ok := msg.(sdk.PlanCompletedEvent); ok {
		t.Error("nil summary should not emit PlanCompletedEvent")
	}
	if _, ok := msg.(sdk.StateRefreshedEvent); !ok {
		t.Errorf("cmd() = %T, want sdk.StateRefreshedEvent", msg)
	}
	if p.status != sdk.StatusDone {
		t.Errorf("status = %v, want Done", p.status)
	}
}

func TestPlugin_WhenViewErrorWithLockInfo_ShouldShowLockDetails(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusError
	p.errMsg = "Error acquiring the state lock"
	p.lockInfo = &sdk.StateLock{
		ID:  "abc-123",
		Who: "user@host",
	}

	view := p.View(80, 24)
	if !strings.Contains(view, "abc-123") {
		t.Errorf("View with lockInfo should contain lock ID, got: %q", view)
	}
	if !strings.Contains(view, "State Lock Detected") {
		t.Errorf("View with lockInfo should contain 'State Lock Detected', got: %q", view)
	}
}

func TestPlugin_WhenTogglePin_ShouldPinAndUnpin(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.togglePin("aws_instance.web")
	if !p.pins.IsPinned("aws_instance.web") {
		t.Error("after togglePin: resource should be pinned")
	}

	p.togglePin("aws_instance.web")
	if p.pins.IsPinned("aws_instance.web") {
		t.Error("after second togglePin: resource should be unpinned")
	}
}

func TestPlugin_WhenTogglePinWithNilPins_ShouldNotPanic(t *testing.T) {
	svc := &mockService{}
	p := New(svc).(*Plugin)
	p.pins = nil

	p.togglePin("aws_instance.web")
	if p.isPinnedAddress("aws_instance.web") {
		t.Error("isPinnedAddress with nil pins should return false")
	}
}

func TestPlugin_WhenRequestApplyCallbackConfirmed_ShouldEmitApplyRequestMsg(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	cmd := p.requestApply()
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	resultCmd := reqMsg.Request.Callback("y")
	if resultCmd == nil {
		t.Fatal("callback('y') returned nil cmd")
	}

	resultMsg := resultCmd()
	if _, ok := resultMsg.(ApplyRequestMsg); !ok {
		t.Fatalf("callback result = %T, want ApplyRequestMsg", resultMsg)
	}
}

func TestPlugin_WhenRequestApplyCallbackDenied_ShouldReturnNil(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	cmd := p.requestApply()
	msg := cmd()
	reqMsg := msg.(sdk.RequestInputMsg)

	resultCmd := reqMsg.Request.Callback("n")
	if resultCmd != nil {
		t.Error("callback('n') returned non-nil cmd, want nil")
	}
}

// --- Frame tests ---

func TestListFrame_WhenCreated_ShouldHaveCorrectID(t *testing.T) {
	p := New(&mockService{}).(*Plugin)
	frame := p.stack.Peek()

	lf, ok := frame.(*listFrame)
	if !ok {
		t.Fatalf("top frame is %T, want *listFrame", frame)
	}
	if lf.ID() != "list" {
		t.Errorf("ID() = %q, want %q", lf.ID(), "list")
	}
}

func TestListFrame_WhenViewCalled_ShouldDelegateToPlugin(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusIdle

	view := p.stack.View(80, 24)
	if view == "" {
		t.Error("frame View() returned empty, want non-empty")
	}
}

func TestListFrame_WhenEscPressed_ShouldEmitDeactivateMsg(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc: cmd = nil, want DeactivateMsg cmd")
	}

	msg := cmd()
	if _, ok := msg.(sdk.DeactivateMsg); !ok {
		t.Fatalf("esc cmd produced %T, want sdk.DeactivateMsg", msg)
	}
}

func TestListFrame_WhenSpacePressed_ShouldTogglePin(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})

	if !p.pins.IsPinned("aws_instance.web") {
		t.Error("after space: resource should be pinned")
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})

	if p.pins.IsPinned("aws_instance.web") {
		t.Error("after second space: resource should be unpinned")
	}
}

func TestListFrame_WhenSpacePressedWithNoSelection_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd != nil {
		t.Error("space with no selection: cmd != nil, want nil")
	}
}

func TestListFrame_WhenAPressedWithResults_ShouldRequestApply(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("a key with results: cmd = nil, want requestApply cmd")
	}

	msg := cmd()
	if _, ok := msg.(sdk.RequestInputMsg); !ok {
		t.Fatalf("a key cmd produced %T, want sdk.RequestInputMsg", msg)
	}
}

func TestListFrame_WhenAPressedWithNoResults_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("a key with no changes: cmd != nil, want nil")
	}
}

func TestListFrame_WhenAPressedWhileNotDone_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusLoading

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd != nil {
		t.Error("a key while loading: cmd != nil, want nil")
	}
}

func TestListFrame_WhenUPressedWithLockInfo_ShouldNavigateToForceUnlock(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "lock-123"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd == nil {
		t.Fatal("u key with lockInfo: cmd = nil, want NavigateMsg cmd")
	}

	msg := cmd()
	navMsg, ok := msg.(sdk.NavigateMsg)
	if !ok {
		t.Fatalf("u key cmd produced %T, want sdk.NavigateMsg", msg)
	}
	if navMsg.PluginID != "forceunlock" {
		t.Errorf("NavigateMsg.PluginID = %q, want %q", navMsg.PluginID, "forceunlock")
	}
}

func TestListFrame_WhenUPressedWithoutLockInfo_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusError
	p.lockInfo = nil

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Error("u key without lockInfo: cmd != nil, want nil")
	}
}

func TestListFrame_WhenUPressedWhileNotError_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.lockInfo = &sdk.StateLock{ID: "lock-123"}

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	if cmd != nil {
		t.Error("u key while not error: cmd != nil, want nil")
	}
}

func TestListFrame_WhenCtrlRPressedWhileIdle_ShouldDoNothing(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusIdle

	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd != nil {
		t.Error("ctrl+r while idle: cmd != nil, want nil")
	}
}

func TestListFrame_WhenDownKeyPressed_ShouldMoveDown(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.tree.Cursor() != 1 {
		t.Errorf("after down: selected = %d, want 1", p.tree.Cursor())
	}
}

func TestListFrame_WhenUpKeyPressed_ShouldMoveUp(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}},
			{Resource: sdk.Resource{Address: "b"}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()
	p.MoveDown()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.tree.Cursor() != 0 {
		t.Errorf("after up: selected = %d, want 0", p.tree.Cursor())
	}
}

func TestListFrame_WhenIKeyPressed_ShouldOpenInspect(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, AttributeDiffs: []sdk.AttributeDiff{{Key: "name"}}},
		},
	}
	p.filtered = p.summary.Changes
	p.rebuildTree()

	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	if p.stack.Peek().ID() != "inspect" {
		t.Errorf("after i: top frame ID = %q, want %q", p.stack.Peek().ID(), "inspect")
	}
}

func TestListFrame_WhenNonKeyMsgReceived_ShouldReturnSelf(t *testing.T) {
	p := newTestPlugin(&mockService{})
	type customMsg struct{}
	cmd := p.stack.Update(customMsg{})
	if cmd != nil {
		t.Error("non-KeyMsg: cmd != nil, want nil")
	}
}

func TestListFrame_WhenHintsCalledIdle_ShouldReturnConfirmAndBack(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusIdle

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty for Idle state")
	}

	hasBack := false
	hasConfirm := false
	for _, h := range hints {
		if h.Key == "q" {
			hasBack = true
		}
		if h.Key == "Enter" {
			hasConfirm = true
		}
	}
	if !hasBack {
		t.Error("Hints(Idle): missing 'q' back hint")
	}
	if !hasConfirm {
		t.Error("Hints(Idle): missing 'Enter' confirm hint")
	}
}

func TestListFrame_WhenHintsCalledLoading_ShouldReturnBackOnly(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusLoading

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty for Loading state")
	}

	hasBack := false
	for _, h := range hints {
		if h.Key == "q" {
			hasBack = true
		}
	}
	if !hasBack {
		t.Error("Hints(Loading): missing 'q' back hint")
	}
}

func TestListFrame_WhenHintsCalledErrorWithLock_ShouldIncludeUnlock(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusError
	p.lockInfo = &sdk.StateLock{ID: "lock-abc"}

	hints := p.stack.Hints()
	hasUnlock := false
	for _, h := range hints {
		if h.Key == "u" {
			hasUnlock = true
		}
	}
	if !hasUnlock {
		t.Error("Hints(Error with lock): missing 'u' force-unlock hint")
	}
}

func TestListFrame_WhenHintsCalledErrorWithoutLock_ShouldNotIncludeUnlock(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusError
	p.lockInfo = nil

	hints := p.stack.Hints()
	for _, h := range hints {
		if h.Key == "u" {
			t.Error("Hints(Error without lock): should not include 'u' force-unlock hint")
		}
	}
}

func TestListFrame_WhenHintsCalledDoneWithChanges_ShouldIncludeApplyAndPin(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "a"}, Action: sdk.ActionCreate},
		},
	}

	hints := p.stack.Hints()
	hasApply := false
	hasPin := false
	hasInspect := false
	for _, h := range hints {
		switch h.Key {
		case "a":
			hasApply = true
		case "Space":
			hasPin = true
		case "Enter":
			hasInspect = true
		}
	}
	if !hasApply {
		t.Error("Hints(Done with changes): missing 'a' apply hint")
	}
	if !hasPin {
		t.Error("Hints(Done with changes): missing 'Space' pin hint")
	}
	if !hasInspect {
		t.Error("Hints(Done with changes): missing 'Enter' inspect hint")
	}
}

func TestListFrame_WhenHintsCalledDoneNoChanges_ShouldIncludeRefreshAndBack(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{Changes: []sdk.PlanChange{}}

	hints := p.stack.Hints()
	hasRefresh := false
	hasBack := false
	for _, h := range hints {
		switch h.Key {
		case "^r":
			hasRefresh = true
		case "q":
			hasBack = true
		}
	}
	if !hasRefresh {
		t.Error("Hints(Done no changes): missing '^r' refresh hint")
	}
	if !hasBack {
		t.Error("Hints(Done no changes): missing 'q' back hint")
	}
}

func TestListFrame_WhenHintsCalledDoneNilSummary_ShouldIncludeRefreshAndBack(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusDone
	p.summary = nil

	hints := p.stack.Hints()
	hasRefresh := false
	hasBack := false
	for _, h := range hints {
		switch h.Key {
		case "^r":
			hasRefresh = true
		case "q":
			hasBack = true
		}
	}
	if !hasRefresh {
		t.Error("Hints(Done nil summary): missing '^r' refresh hint")
	}
	if !hasBack {
		t.Error("Hints(Done nil summary): missing 'q' back hint")
	}
}

func TestListFrame_WhenHintsCalledUnknownStatus_ShouldReturnBackOnly(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.Status(99)

	hints := p.stack.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints() returned empty for unknown status")
	}
	hasBack := false
	for _, h := range hints {
		if h.Key == "q" {
			hasBack = true
		}
	}
	if !hasBack {
		t.Error("Hints(unknown status): missing 'q' back hint")
	}
}

func TestPlugin_WhenViewLoadingState_ShouldShowRunningMessage(t *testing.T) {
	p := newTestPlugin(&mockService{})
	p.status = sdk.StatusLoading

	view := p.View(80, 24)
	if view == "" {
		t.Error("View(Loading) returned empty string")
	}
}

func TestPlugin_WhenPinnedResourceRendered_ShouldShowPinMark(t *testing.T) {
	svc := &mockService{}
	p := newTestPlugin(svc)
	p.status = sdk.StatusDone
	p.summary = &sdk.PlanSummary{
		Changes: []sdk.PlanChange{
			{Resource: sdk.Resource{Address: "aws_instance.web"}, Action: sdk.ActionCreate},
		},
		ToCreate: 1,
	}
	p.pins.Toggle("aws_instance.web")

	view := p.View(80, 24)
	if view == "" {
		t.Error("View with pinned resource returned empty")
	}
}
