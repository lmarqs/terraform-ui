package init

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	sdkframes "github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

// newTestResultFrame creates a resultFrame backed by a no-op StreamFrame for unit tests.
func newTestResultFrame(timer *ui.Timer) *resultFrame {
	lw, ch := sdkframes.NewLineWriter()
	lw.Close()
	sf := sdkframes.NewStreamFrame("test", ch, nil)
	return newResultFrame(timer, sf)
}

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)

	if p.ID() != "init" {
		t.Errorf("ID() = %q, want %q", p.ID(), "init")
	}
	if p.Name() != "Init" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Init")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if !p.Ready() {
		t.Error("Ready() should always be true")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if cmd := p.Init(&sdk.Context{Service: svc}); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
}

func TestActivate_WhenCalled_ShouldPushFormFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()

	if p.stack.Peek() == nil {
		t.Fatal("Activate() should push form onto stack")
	}
	if p.stack.Peek().ID() != "form" {
		t.Errorf("top frame ID = %q, want %q", p.stack.Peek().ID(), "form")
	}
}

func TestPlugin_WhenFieldsToggled_ShouldFlipBooleanValues(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	if p.upgrade {
		t.Fatal("upgrade should start false")
	}
	p.upgrade = !p.upgrade
	if !p.upgrade {
		t.Fatal("toggle should flip to true")
	}

	if p.reconfigure {
		t.Fatal("reconfigure should start false")
	}
	if !p.backend {
		t.Fatal("backend should start true")
	}
}

func TestUpdate_WhenInitSubmitMsg_ShouldPushResultFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()

	p.Update(initSubmitMsg{})

	top := p.stack.Peek()
	if top == nil {
		t.Fatal("submit should push result frame onto stack")
	}
	if top.ID() != "result" {
		t.Errorf("top frame ID = %q, want %q", top.ID(), "result")
	}
}

func TestUpdate_WhenInitResultSuccess_ShouldEmitDeactivate(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	_, cmd := p.Update(InitResultMsg{Err: nil})
	if cmd == nil {
		t.Fatal("success should return a command")
	}
}

func TestUpdate_WhenInitResultError_ShouldStayOnResultFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	p.Update(InitResultMsg{Err: errors.New("backend error")})

	top := p.stack.Peek()
	if top == nil {
		t.Fatal("error should keep result frame on stack")
	}
	rf, ok := top.(*resultFrame)
	if !ok {
		t.Fatalf("top frame is %T, want *resultFrame", top)
	}
	if rf.status != sdk.StatusError {
		t.Errorf("result frame status = %v, want StatusError", rf.status)
	}
	if rf.errMsg != "backend error" {
		t.Errorf("result frame errMsg = %q, want %q", rf.errMsg, "backend error")
	}
}

func TestResultFrame_WhenEnterPressedInError_ShouldReturnToForm(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})
	p.Update(InitResultMsg{Err: errors.New("fail")})

	// Enter on error frame should pop back to form
	p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e', 'n', 't', 'e', 'r'}})

	// Actually send as proper enter key
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})

	top := p.stack.Peek()
	if top == nil {
		t.Fatal("after Enter on error, form should be visible")
	}
	if top.ID() != "form" {
		t.Errorf("top frame ID = %q, want %q (form should be back)", top.ID(), "form")
	}
}

func TestHandleChdirChanged_WhenCalled_ShouldResetStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()

	p.HandleChdirChanged(sdk.ChdirChangedEvent{AbsPath: "/new/dir"})

	if !p.stack.IsEmpty() {
		t.Error("chdir should reset the stack")
	}
}

func TestUpdate_WhenTimerTickMsg_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	_, cmd := p.Update(ui.TimerTickMsg{})
	_ = cmd
}

func TestView_WhenFormActive_ShouldDelegateToStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()

	view := p.View(80, 24)
	if view == "" {
		t.Error("View should delegate to form frame")
	}
}

func TestView_GivenLoadingWithNoStreamOutput_ShouldShowProgressMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	view := p.View(80, 24)
	if !strings.Contains(view, "Running terraform init") {
		t.Errorf("View in Loading state should show progress message, got %q", view)
	}
}

func TestView_WhenResultFrameError_ShouldReturnNonEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})
	p.Update(InitResultMsg{Err: errors.New("something failed")})

	view := p.View(80, 24)
	if view == "" {
		t.Error("View in Error state should not be empty")
	}
}

func TestBusy_WhenStatusChanges_ShouldReflectLoadingState(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)

	if p.Busy() {
		t.Error("should not be busy before submit")
	}

	p.Activate()
	p.Update(initSubmitMsg{})

	if !p.Busy() {
		t.Error("should be busy during loading")
	}

	p.Update(InitResultMsg{Err: errors.New("fail")})

	if p.Busy() {
		t.Error("should not be busy after result")
	}
}

func TestPlugin_WhenStack_ShouldReturnNonNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	if p.Stack() == nil {
		t.Fatal("Stack() = nil, want non-nil")
	}
}

func TestPlugin_WhenSubmitFromForm_ShouldEmitInitSubmitMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	cmd := p.submitFromForm()
	if cmd == nil {
		t.Fatal("submitFromForm() = nil")
	}
	msg := cmd()
	if _, ok := msg.(initSubmitMsg); !ok {
		t.Errorf("cmd() = %T, want initSubmitMsg", msg)
	}
}

func TestPlugin_WhenEditExtraArgs_ShouldEmitRequestInputMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	cmd := p.editExtraArgs()
	if cmd == nil {
		t.Fatal("editExtraArgs() = nil")
	}
	msg := cmd()
	if _, ok := msg.(sdk.RequestInputMsg); !ok {
		t.Errorf("cmd() = %T, want RequestInputMsg", msg)
	}
}

func TestPlugin_WhenCancelWithNilFn_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.cancelFn = nil
	p.Cancel()
}

func TestPlugin_WhenCancelWithFn_ShouldCallAndClear(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	called := false
	p.cancelFn = func() { called = true }
	p.Cancel()
	if !called {
		t.Error("Cancel() should call cancelFn")
	}
	if p.cancelFn != nil {
		t.Error("Cancel() should clear cancelFn")
	}
}

func TestPlugin_WhenSubmitWithBackendFalse_ShouldSetBackendPtr(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.backend = false
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if len(svc.InitCalls) == 0 {
		t.Fatal("Init should have been called")
	}
	if svc.InitCalls[0].Backend == nil {
		t.Fatal("Backend should be non-nil when backend=false")
	}
	if *svc.InitCalls[0].Backend != false {
		t.Error("Backend should be &false")
	}
}

func TestPlugin_WhenSubmitWithBackendTrue_ShouldLeaveBackendNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.backend = true
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if len(svc.InitCalls) == 0 {
		t.Fatal("Init should have been called")
	}
	if svc.InitCalls[0].Backend != nil {
		t.Error("Backend should be nil when backend=true")
	}
}

func TestPlugin_WhenSubmitWithExtraArgs_ShouldSplitFields(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.extraArgs = "-lock=false -input=false"
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if len(svc.InitCalls) == 0 {
		t.Fatal("Init should have been called")
	}
	if len(svc.InitCalls[0].ExtraArgs) != 2 {
		t.Fatalf("ExtraArgs len = %d, want 2", len(svc.InitCalls[0].ExtraArgs))
	}
	if svc.InitCalls[0].ExtraArgs[0] != "-lock=false" {
		t.Errorf("ExtraArgs[0] = %q, want %q", svc.InitCalls[0].ExtraArgs[0], "-lock=false")
	}
}

func TestPlugin_WhenExtraArgsDisplayEmpty_ShouldReturnNone(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.extraArgs = ""
	if got := p.extraArgsDisplay(); got != "(none)" {
		t.Errorf("extraArgsDisplay() = %q, want %q", got, "(none)")
	}
}

func TestPlugin_WhenExtraArgsDisplaySet_ShouldReturnValue(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.extraArgs = "-lock=false"
	if got := p.extraArgsDisplay(); got != "-lock=false" {
		t.Errorf("extraArgsDisplay() = %q, want %q", got, "-lock=false")
	}
}

func TestPlugin_WhenOutput_ShouldReturnSuccessMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	data, err := p.Output(false)
	if err != nil {
		t.Fatalf("Output() error = %v", err)
	}
	if string(data) != "Initialized successfully.\n" {
		t.Errorf("Output() = %q, want %q", string(data), "Initialized successfully.\n")
	}
}

func TestResultFrame_WhenViewInDone_ShouldShowSuccess(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	rf.status = sdk.StatusDone
	view := rf.View(80, 24)
	if view == "" {
		t.Error("View in Done should not be empty")
	}
}

func TestResultFrame_WhenHintsInLoading_ShouldReturnBack(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	hints := rf.Hints()
	if len(hints) == 0 {
		t.Fatal("Hints in Loading returned empty")
	}
}

func TestResultFrame_WhenHintsInError_ShouldReturnEnterAndBack(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	rf.status = sdk.StatusError
	hints := rf.Hints()
	if len(hints) != 2 {
		t.Fatalf("Hints in Error: len = %d, want 2", len(hints))
	}
	if hints[0].Key != "Enter" {
		t.Errorf("hints[0].Key = %q, want %q", hints[0].Key, "Enter")
	}
}

func TestResultFrame_WhenHintsInDone_ShouldReturnNil(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	rf.status = sdk.StatusDone
	if hints := rf.Hints(); hints != nil {
		t.Errorf("Hints in Done = %v, want nil", hints)
	}
}

func TestResultFrame_WhenSuccess_ShouldReturnCmd(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	_, cmd := rf.Update(InitResultMsg{Err: nil, Duration: time.Second})
	if cmd == nil {
		t.Fatal("success should return cmd")
	}
	if rf.status != sdk.StatusDone {
		t.Errorf("status = %v, want Done", rf.status)
	}
	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("cmd() returned %T, want tea.BatchMsg", msg)
	}
	foundInvalidated := false
	foundDeactivate := false
	for _, subCmd := range batchMsg {
		if subCmd == nil {
			continue
		}
		subMsg := subCmd()
		switch subMsg.(type) {
		case sdk.PlanInvalidatedEvent:
			foundInvalidated = true
		case sdk.DeactivateMsg:
			foundDeactivate = true
		}
	}
	if !foundInvalidated {
		t.Error("batch should contain PlanInvalidatedEvent")
	}
	if !foundDeactivate {
		t.Error("batch should contain DeactivateMsg")
	}
}

func TestResultFrame_WhenError_ShouldSetErrorStatus(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	_, cmd := rf.Update(InitResultMsg{Err: errors.New("fail"), Duration: time.Second})
	if cmd != nil {
		t.Error("error should return nil cmd")
	}
	if rf.status != sdk.StatusError {
		t.Errorf("status = %v, want Error", rf.status)
	}
	if rf.errMsg != "fail" {
		t.Errorf("errMsg = %q, want %q", rf.errMsg, "fail")
	}
}

func TestResultFrame_WhenEnterInError_ShouldPop(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	rf.status = sdk.StatusError
	frame, _ := rf.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if frame != nil {
		t.Error("Enter in error should return nil to pop")
	}
}

func TestResultFrame_WhenEnterInLoading_ShouldNotPop(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	frame, _ := rf.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if frame == nil {
		t.Error("Enter in loading should not pop")
	}
}

func TestPlugin_WhenSubmitWithUpgrade_ShouldPassUpgradeOption(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.upgrade = true
	p.reconfigure = true
	p.Activate()
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if len(svc.InitCalls) == 0 {
		t.Fatal("Init should have been called")
	}
	if !svc.InitCalls[0].Upgrade {
		t.Error("Upgrade should be true")
	}
	if !svc.InitCalls[0].Reconfigure {
		t.Error("Reconfigure should be true")
	}
}

func executeBatch(cmd tea.Cmd) []tea.Msg {
	msg := cmd()
	if batchMsg, ok := msg.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batchMsg {
			if c != nil {
				msgs = append(msgs, c())
			}
		}
		return msgs
	}
	return []tea.Msg{msg}
}

func TestPlugin_WhenUpdateWithUnhandledMsg_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()

	type unknownMsg struct{}
	_, cmd := p.Update(unknownMsg{})
	if cmd != nil {
		t.Error("Update(unknownMsg) should return nil cmd")
	}
}

func TestPlugin_WhenUpdateTimerTickWithNoTopFrame_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	// Stack empty - no result frame
	p.stack.Reset()

	_, cmd := p.Update(ui.TimerTickMsg{})
	if cmd != nil {
		t.Error("TimerTickMsg with empty stack should return nil cmd")
	}
}

func TestPlugin_WhenUpdateInitResultMsgWithNoTopFrame_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.stack.Reset()

	_, cmd := p.Update(InitResultMsg{Err: nil, Duration: time.Second})
	if cmd != nil {
		t.Error("InitResultMsg with empty stack should return nil cmd")
	}
}

func TestPlugin_WhenFormFieldUpgradeSelected_ShouldToggleUpgrade(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.upgrade = false
	p.Activate()

	// Form is on top of stack with cursor on first selectable field (upgrade)
	// Press enter to invoke OnSelect
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !p.upgrade {
		t.Error("expected upgrade=true after pressing enter on upgrade field")
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.upgrade {
		t.Error("expected upgrade=false after second press on upgrade field")
	}
}

func TestPlugin_WhenFormFieldReconfigureSelected_ShouldToggle(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.reconfigure = false
	p.Activate()

	// Move down to reconfigure field
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !p.reconfigure {
		t.Error("expected reconfigure=true after pressing enter on reconfigure field")
	}
}

func TestPlugin_WhenFormFieldBackendSelected_ShouldToggle(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.backend = true
	p.Activate()

	// Move down to backend field (3rd selectable)
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.backend {
		t.Error("expected backend=false after pressing enter on backend field")
	}
}

func TestPlugin_WhenFormFieldExtraArgsSelected_ShouldEmitInputRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()

	// Move down to extra args field (4th selectable)
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for extra args field selection")
	}
	msg := cmd()
	if _, ok := msg.(sdk.RequestInputMsg); !ok {
		t.Errorf("expected RequestInputMsg, got %T", msg)
	}
}

func TestPlugin_WhenEditExtraArgsCallback_ShouldStoreValue(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.extraArgs = ""

	cmd := p.editExtraArgs()
	msg := cmd()
	reqMsg, ok := msg.(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("editExtraArgs cmd returned %T, want RequestInputMsg", msg)
	}

	result := reqMsg.Request.Callback("-lock=false")
	if result != nil {
		t.Error("callback should return nil cmd")
	}
	if p.extraArgs != "-lock=false" {
		t.Errorf("extraArgs = %q, want %q", p.extraArgs, "-lock=false")
	}
}

func TestResultFrame_WhenTimerTickMsg_ShouldReturnTickCmd(t *testing.T) {
	var timer ui.Timer
	timer.Start()
	rf := newTestResultFrame(&timer)

	_, cmd := rf.Update(ui.TimerTickMsg{})
	if cmd == nil {
		t.Error("TimerTickMsg with running timer should return non-nil cmd")
	}
}

func TestResultFrame_WhenTimerTickMsgNotRunning_ShouldReturnNilCmd(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)

	_, cmd := rf.Update(ui.TimerTickMsg{})
	if cmd != nil {
		t.Error("TimerTickMsg with stopped timer should return nil cmd")
	}
}

func TestResultFrame_WhenUnhandledKeyInLoading_ShouldReturnSelf(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)

	frame, cmd := rf.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if frame != rf {
		t.Error("unhandled key in loading should return self")
	}
	if cmd != nil {
		t.Error("unhandled key should return nil cmd")
	}
}

func TestResultFrame_WhenNonEnterKeyInError_ShouldReturnSelf(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	rf.status = sdk.StatusError

	frame, cmd := rf.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if frame != rf {
		t.Error("non-enter key in error should return self (not pop)")
	}
	if cmd != nil {
		t.Error("non-enter key in error should return nil cmd")
	}
}

func TestPlugin_WhenUpdateWithUnhandledMsgAndStackHasFrame_ShouldReturnNil(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate()
	p.Update(initSubmitMsg{})

	type customMsg struct{}
	_, cmd := p.Update(customMsg{})
	if cmd != nil {
		t.Error("unhandled msg with result frame on stack should return nil cmd")
	}
}

func TestActivateWithArgs_WhenUpgradeFlag_ShouldAutoSubmitWithUpgrade(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.ActivateWithArgs([]string{"--upgrade"})
	if cmd == nil {
		t.Fatal("ActivateWithArgs should return auto-submit cmd")
	}
	msg := cmd()
	if _, ok := msg.(initSubmitMsg); !ok {
		t.Errorf("cmd() = %T, want initSubmitMsg", msg)
	}
	if !p.upgrade {
		t.Error("upgrade should be true")
	}
}

func TestActivateWithArgs_WhenReconfigureFlag_ShouldAutoSubmitWithReconfigure(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.ActivateWithArgs([]string{"--reconfigure"})
	if cmd == nil {
		t.Fatal("ActivateWithArgs should return auto-submit cmd")
	}
	if !p.reconfigure {
		t.Error("reconfigure should be true")
	}
}

func TestActivateWithArgs_WhenBackendFalse_ShouldSetBackendFalse(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.ActivateWithArgs([]string{"--backend=false"})
	if cmd == nil {
		t.Fatal("ActivateWithArgs should return auto-submit cmd")
	}
	if p.backend {
		t.Error("backend should be false")
	}
}

func TestActivateWithArgs_WhenBackendTrue_ShouldLeaveBackendTrue(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.backend = false // start with false to verify it gets set
	p.ActivateWithArgs([]string{"--backend=true"})
	if !p.backend {
		t.Error("backend should be true")
	}
}

func TestActivateWithArgs_WhenBackendConfig_ShouldPassToService(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.ActivateWithArgs([]string{"--backend-config=path/to/config.hcl", "--backend-config=key=value"})

	// Trigger the submit
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		msgs := executeBatch(cmd)
		for _, msg := range msgs {
			p.Update(msg)
		}
	}

	if len(svc.InitCalls) == 0 {
		t.Fatal("Init should have been called")
	}
	if len(svc.InitCalls[0].BackendConfig) != 2 {
		t.Fatalf("BackendConfig len = %d, want 2", len(svc.InitCalls[0].BackendConfig))
	}
	if svc.InitCalls[0].BackendConfig[0] != "path/to/config.hcl" {
		t.Errorf("BackendConfig[0] = %q, want %q", svc.InitCalls[0].BackendConfig[0], "path/to/config.hcl")
	}
	if svc.InitCalls[0].BackendConfig[1] != "key=value" {
		t.Errorf("BackendConfig[1] = %q, want %q", svc.InitCalls[0].BackendConfig[1], "key=value")
	}
}

func TestActivateWithArgs_WhenMultipleFlags_ShouldSetAll(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.ActivateWithArgs([]string{"--upgrade", "--reconfigure", "--backend=false"})
	if !p.upgrade {
		t.Error("upgrade should be true")
	}
	if !p.reconfigure {
		t.Error("reconfigure should be true")
	}
	if p.backend {
		t.Error("backend should be false")
	}
}

func TestActivateWithArgs_WhenInteractiveFlag_ShouldShowFormWithoutAutoSubmit(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.ActivateWithArgs([]string{"--interactive", "--upgrade"})
	if cmd != nil {
		t.Error("--interactive should NOT auto-submit (return nil cmd)")
	}
	if !p.upgrade {
		t.Error("upgrade should still be pre-filled")
	}
	if p.stack.Peek() == nil || p.stack.Peek().ID() != "form" {
		t.Error("form should be on stack")
	}
}

func TestActivateWithArgs_WhenEmptyArgs_ShouldAutoSubmitWithDefaults(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.ActivateWithArgs([]string{})
	if cmd == nil {
		t.Fatal("empty args should still auto-submit (terraform default behavior)")
	}
}

func TestActivateWithArgs_ShouldResetPreviousState(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	// Set some previous state
	p.upgrade = true
	p.reconfigure = true
	p.backend = false
	p.backendConfigs = []string{"old"}
	p.extraArgs = "old args"

	p.ActivateWithArgs([]string{"--upgrade"})

	if !p.upgrade {
		t.Error("upgrade should be true (from args)")
	}
	if p.reconfigure {
		t.Error("reconfigure should be reset to false")
	}
	if !p.backend {
		t.Error("backend should be reset to true (default)")
	}
	if len(p.backendConfigs) != 0 {
		t.Errorf("backendConfigs should be reset, got %v", p.backendConfigs)
	}
	if p.extraArgs != "" {
		t.Errorf("extraArgs should be reset, got %q", p.extraArgs)
	}
}
