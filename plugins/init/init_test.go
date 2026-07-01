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
	if cmd := p.Init(&sdk.PluginDeps{Service: svc}); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
}

func TestActivate_WhenCalled_ShouldPushFormFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})

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
	p.Activate(Input{})

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
	p.Activate(Input{})
	p.Update(initSubmitMsg{})

	_, cmd := p.Update(InitResultMsg{Err: nil})
	if cmd == nil {
		t.Fatal("success should return a command")
	}
}

func TestUpdate_WhenInitResultError_ShouldStayOnResultFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})
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
	p.Activate(Input{})
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

func TestUpdate_WhenTimerTickMsg_ShouldNotPanic(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})
	p.Update(initSubmitMsg{})

	_, cmd := p.Update(ui.TimerTickMsg{})
	_ = cmd
}

func TestView_WhenFormActive_ShouldDelegateToStack(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})

	view := p.View(80, 24)
	if view == "" {
		t.Error("View should delegate to form frame")
	}
}

func TestView_GivenLoadingWithNoStreamOutput_ShouldShowProgressMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})
	p.Update(initSubmitMsg{})

	view := p.View(80, 24)
	// Before any output arrives, the stream shows its elapsed header so the user
	// gets immediate feedback instead of a blank panel.
	if !strings.Contains(view, "terraform init") {
		t.Errorf("View in Loading state should show progress header, got %q", view)
	}
	if !strings.Contains(view, "0s") {
		t.Errorf("View in Loading state should show elapsed time, got %q", view)
	}
}

func TestView_WhenResultFrameError_ShouldReturnNonEmpty(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})
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

	p.Activate(Input{})
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

func TestPlugin_WhenEditText_ShouldEmitRequestInputMsg(t *testing.T) {
	cmd := editText("from-module", "", func(string) {})
	if cmd == nil {
		t.Fatal("editText() = nil")
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
	f := false
	p.Activate(Input{Backend: &f})
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
	if svc.InitCalls[0].Backend != sdk.BackendDisabled {
		t.Errorf("Backend = %v, want BackendDisabled", svc.InitCalls[0].Backend)
	}
}

func TestPlugin_WhenSubmitWithBackendTrue_ShouldLeaveBackendNil(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.backend = true
	p.Activate(Input{})
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
	if svc.InitCalls[0].Backend != sdk.BackendDefault {
		t.Errorf("Backend = %v, want BackendDefault", svc.InitCalls[0].Backend)
	}
}

func TestPlugin_WhenSubmitWithTypedOptions_ShouldForwardAll(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Activate(Input{})
	p.forceCopy = true
	p.get = false
	p.fromModule = "./template"
	p.pluginDir = []string{"/plugins"}
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
	got := svc.InitCalls[0]
	if !got.ForceCopy {
		t.Error("ForceCopy should be true")
	}
	if got.Get == nil || *got.Get {
		t.Errorf("Get = %v, want pointer to false", got.Get)
	}
	if got.FromModule != "./template" {
		t.Errorf("FromModule = %q, want %q", got.FromModule, "./template")
	}
	if len(got.PluginDir) != 1 || got.PluginDir[0] != "/plugins" {
		t.Errorf("PluginDir = %v, want [/plugins]", got.PluginDir)
	}
}

// TestPlugin_ShouldNotOfferLockOptions guards against re-introducing the
// -lock / -lock-timeout init flags. terraform-exec rejects them for any
// Terraform >= 0.15, so exposing them made every init fail on modern binaries.
func TestPlugin_ShouldNotOfferLockOptions(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})
	view := p.View(120, 40)
	if strings.Contains(view, "lock") {
		t.Errorf("init form must not offer lock/lock-timeout options, got view:\n%s", view)
	}
}

func TestPlugin_WhenSubmitWithDefaults_ShouldSendGetTrue(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Activate(Input{})
	_, cmd := p.Update(initSubmitMsg{})
	if cmd != nil {
		for _, msg := range executeBatch(cmd) {
			p.Update(msg)
		}
	}

	if len(svc.InitCalls) == 0 {
		t.Fatal("Init should have been called")
	}
	got := svc.InitCalls[0]
	if got.Get == nil || !*got.Get {
		t.Errorf("Get = %v, want pointer to true (terraform default)", got.Get)
	}
}

func TestDisplay_WhenEmpty_ShouldReturnNone(t *testing.T) {
	if got := display(""); got != "(none)" {
		t.Errorf("display(\"\") = %q, want %q", got, "(none)")
	}
}

func TestDisplay_WhenSet_ShouldReturnValue(t *testing.T) {
	if got := display("30s"); got != "30s" {
		t.Errorf("display(\"30s\") = %q, want %q", got, "30s")
	}
}

func TestPlugin_WhenOutput_ShouldReturnSuccessMessage(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	data, err := p.Stdout()
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
	p.Activate(Input{Upgrade: true, Reconfigure: true})
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
	p.Activate(Input{})

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

func TestPlugin_WhenSpaceOnUpgradeField_ShouldToggleUpgrade(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.upgrade = false
	p.Activate(Input{})

	// Cursor starts on the first toggle (upgrade); Space flips it.
	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !p.upgrade {
		t.Error("expected upgrade=true after pressing space on upgrade field")
	}

	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if p.upgrade {
		t.Error("expected upgrade=false after second space on upgrade field")
	}
}

func TestPlugin_WhenSpaceOnReconfigureField_ShouldToggle(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.reconfigure = false
	p.Activate(Input{})

	// Move down to reconfigure field
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !p.reconfigure {
		t.Error("expected reconfigure=true after pressing space on reconfigure field")
	}
}

func TestPlugin_WhenSpaceOnBackendField_ShouldToggle(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.backend = true
	p.Activate(Input{})

	// Move down to backend field (3rd selectable)
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if p.backend {
		t.Error("expected backend=false after pressing space on backend field")
	}
}

func TestPlugin_WhenEnterOnCheckbox_ShouldSubmitForm(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.upgrade = false
	p.Activate(Input{})

	// Enter confirms the form from any field: it runs init, never toggles.
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if p.upgrade {
		t.Error("enter on a checkbox must not toggle it")
	}
	if cmd == nil {
		t.Fatal("enter on a checkbox should submit the form")
	}
	if _, ok := cmd().(initSubmitMsg); !ok {
		t.Errorf("enter should emit initSubmitMsg, got %T", cmd())
	}
}

func TestPlugin_WhenSpaceOnTextField_ShouldEmitInputRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})

	// Move down past the 5 toggle fields to the first text field (from-module).
	for i := 0; i < 5; i++ {
		p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeySpace})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for space on text field")
	}
	if _, ok := cmd().(sdk.RequestInputMsg); !ok {
		t.Errorf("expected RequestInputMsg, got %T", cmd())
	}
}

func TestPlugin_WhenTextFieldCallback_ShouldStoreValue(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	field := textField("from-module", &p.fromModule)

	cmd := field.OnSpace()
	reqMsg, ok := cmd().(sdk.RequestInputMsg)
	if !ok {
		t.Fatalf("textField OnSpace returned %T, want RequestInputMsg", cmd())
	}
	if result := reqMsg.Request.Callback("./template"); result != nil {
		t.Error("callback should return nil cmd")
	}
	if p.fromModule != "./template" {
		t.Errorf("fromModule = %q, want %q", p.fromModule, "./template")
	}
}

func TestPlugin_WhenListFieldCallback_ShouldSplitAndStore(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	field := listField("plugin-dir", &p.pluginDir)

	reqMsg := field.OnSpace()().(sdk.RequestInputMsg)
	reqMsg.Request.Callback("/a /b")
	if len(p.pluginDir) != 2 || p.pluginDir[0] != "/a" || p.pluginDir[1] != "/b" {
		t.Errorf("pluginDir = %v, want [/a /b]", p.pluginDir)
	}

	// Clearing the field resets the slice to nil.
	reqMsg.Request.Callback("   ")
	if p.pluginDir != nil {
		t.Errorf("pluginDir = %v, want nil after clearing", p.pluginDir)
	}
}

func TestPlugin_WhenEnterOnTextField_ShouldSubmitNotEdit(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})

	// Enter on a text field confirms the form rather than opening the editor.
	for i := 0; i < 6; i++ {
		p.stack.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter on text field should submit the form")
	}
	if _, ok := cmd().(initSubmitMsg); !ok {
		t.Errorf("enter should emit initSubmitMsg, got %T", cmd())
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
	p.Activate(Input{})
	p.Update(initSubmitMsg{})

	type customMsg struct{}
	_, cmd := p.Update(customMsg{})
	if cmd != nil {
		t.Error("unhandled msg with result frame on stack should return nil cmd")
	}
}

func TestActivate_WhenUpgradeInput_ShouldAutoSubmitWithUpgrade(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.Activate(Input{Upgrade: true})
	if cmd == nil {
		t.Fatal("Activate with Upgrade should return auto-submit cmd")
	}
	msg := cmd()
	if _, ok := msg.(initSubmitMsg); !ok {
		t.Errorf("cmd() = %T, want initSubmitMsg", msg)
	}
	if !p.upgrade {
		t.Error("upgrade should be true")
	}
}

func TestActivate_WhenReconfigureInput_ShouldAutoSubmitWithReconfigure(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.Activate(Input{Reconfigure: true})
	if cmd == nil {
		t.Fatal("Activate with Reconfigure should return auto-submit cmd")
	}
	if !p.reconfigure {
		t.Error("reconfigure should be true")
	}
}

func TestActivate_WhenBackendFalse_ShouldSetBackendFalse(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	f := false
	cmd := p.Activate(Input{Backend: &f})
	if cmd == nil {
		t.Fatal("Activate with Backend=false should return auto-submit cmd")
	}
	if p.backend {
		t.Error("backend should be false")
	}
}

func TestActivate_WhenBackendTrue_ShouldSetBackendTrue(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.backend = false
	tr := true
	p.Activate(Input{Backend: &tr})
	if !p.backend {
		t.Error("backend should be true")
	}
}

func TestActivate_WhenBackendConfig_ShouldPassToService(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Activate(Input{BackendConfig: []string{"path/to/config.hcl", "key=value"}})

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

func TestActivate_WhenMultipleInputFields_ShouldSetAll(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	f := false
	p.Activate(Input{Upgrade: true, Reconfigure: true, Backend: &f})
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

func TestActivate_WhenEmptyInput_ShouldShowFormWithoutAutoSubmit(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.Activate(Input{})
	if cmd != nil {
		t.Error("empty Input should not auto-submit (show form instead)")
	}
}

func TestUpdate_WhenStreamLineMsgWithResultFrame_ShouldRouteToTopFrame(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.Activate(Input{})
	p.Update(initSubmitMsg{})
	// result frame is now on top of stack

	_, cmd := p.Update(sdkframes.StreamLineMsg{Line: "output"})
	if cmd == nil {
		t.Fatal("StreamLineMsg with result frame on stack should return non-nil cmd (WaitForLine)")
	}
}

func TestUpdate_WhenStreamLineMsgWithEmptyStack_ShouldReturnWaitForLineCmd(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	// stack is empty immediately after New (before Activate)
	lw, ch := sdkframes.NewLineWriter()
	p.ch = ch

	_, cmd := p.Update(sdkframes.StreamLineMsg{Line: "orphan line"})
	if cmd == nil {
		t.Fatal("StreamLineMsg with empty stack should return WaitForLine cmd")
	}
	lw.Close()
}

func TestResultFrame_WhenKeyMsgInDone_ShouldDelegateToStream(t *testing.T) {
	var timer ui.Timer
	rf := newTestResultFrame(&timer)
	rf.status = sdk.StatusDone

	// Any key in Done state delegates to the stream frame
	frame, _ := rf.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if frame != rf {
		t.Error("key in Done should return self (not pop)")
	}
}

func TestResultFrame_WhenViewInLoadingWithStreamOutput_ShouldReturnStreamView(t *testing.T) {
	var timer ui.Timer
	lw, ch := sdkframes.NewLineWriter()
	sf := sdkframes.NewStreamFrame("test", ch, nil)
	rf := newResultFrame(&timer, sf)

	// Seed a line through the result frame so stream.View() returns non-empty
	rf.Update(sdkframes.StreamLineMsg{Line: "terraform output"})

	view := rf.View(80, 24)
	if view == "" {
		t.Error("View in Loading with stream output should return non-empty string")
	}
	lw.Close()
}

func TestActivate_ShouldResetPreviousState(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.upgrade = true
	p.reconfigure = true
	p.backend = false
	p.get = false
	p.forceCopy = true
	p.fromModule = "./old"
	p.pluginDir = []string{"/old"}
	p.backendConfigs = []string{"old"}

	p.Activate(Input{Upgrade: true})

	if !p.upgrade {
		t.Error("upgrade should be true (from args)")
	}
	if p.reconfigure {
		t.Error("reconfigure should be reset to false")
	}
	if !p.backend {
		t.Error("backend should be reset to true (default)")
	}
	if !p.get {
		t.Error("get should be reset to true (default)")
	}
	if p.forceCopy {
		t.Error("forceCopy should be reset to false")
	}
	if p.fromModule != "" {
		t.Errorf("fromModule should be reset, got %q", p.fromModule)
	}
	if len(p.pluginDir) != 0 {
		t.Errorf("pluginDir should be reset, got %v", p.pluginDir)
	}
	if len(p.backendConfigs) != 0 {
		t.Errorf("backendConfigs should be reset, got %v", p.backendConfigs)
	}
}

func TestHandleContextChanged_ShouldResetStack(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	p.Activate(Input{}) // pushes form
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})
	if cmd != nil {
		t.Error("HandleContextChanged returned non-nil cmd")
	}
	if p.stack.Depth() != 0 {
		t.Errorf("stack not reset, depth = %d", p.stack.Depth())
	}
}

func TestHandleContextChanged_WhenNextNil_ShouldBeNoOp(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc).(*Plugin)
	cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil})
	if cmd != nil {
		t.Error("HandleContextChanged with nil Next returned non-nil cmd")
	}
}
