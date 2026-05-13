package state

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

type trackingMockService struct {
	mockService
	taintCalled   []string
	untaintCalled []string
	stateRmCalled []string
	moveSource    string
	moveDest      string
	importAddr    string
	importID      string
	taintErr      error
	untaintErr    error
	stateRmErr    error
	stateMoveErr  error
	importErr     error
}

func (m *trackingMockService) Taint(_ context.Context, addr string) error {
	m.taintCalled = append(m.taintCalled, addr)
	return m.taintErr
}

func (m *trackingMockService) Untaint(_ context.Context, addr string) error {
	m.untaintCalled = append(m.untaintCalled, addr)
	return m.untaintErr
}

func (m *trackingMockService) StateRm(_ context.Context, addr string) error {
	m.stateRmCalled = append(m.stateRmCalled, addr)
	return m.stateRmErr
}

func (m *trackingMockService) StateMove(_ context.Context, source, dest string) error {
	m.moveSource = source
	m.moveDest = dest
	return m.stateMoveErr
}

func (m *trackingMockService) Import(_ context.Context, addr, id string) error {
	m.importAddr = addr
	m.importID = id
	return m.importErr
}

func newTrackingPlugin(svc *trackingMockService, resources []sdk.Resource) *Plugin {
	p := New(svc).(*Plugin)
	p.svc = svc
	p.status = sdk.StatusDone
	p.resources = resources
	p.filtered = resources
	p.pins = sdk.NewPinService()
	p.rebuildTree()
	return p
}

func TestRequestMove_WhenCalledWithAddress_ShouldProduceTextInputThenConfirm(t *testing.T) {
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, nil)

	t.Run("ShouldReturnRequestInputMsg", func(t *testing.T) {
		cmd := p.requestMove("aws_instance.web")
		msg := cmd()
		reqMsg, ok := msg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
		}
		if reqMsg.Request.Mode != sdk.InputRequestText {
			t.Errorf("expected InputRequestText, got %d", reqMsg.Request.Mode)
		}
		if reqMsg.Request.Prompt != "Move to:" {
			t.Errorf("expected prompt 'Move to:', got %q", reqMsg.Request.Prompt)
		}
		if reqMsg.Request.Default != "aws_instance.web" {
			t.Errorf("expected default 'aws_instance.web', got %q", reqMsg.Request.Default)
		}
	})

	t.Run("ShouldReturnNilWhenDestIsEmpty", func(t *testing.T) {
		cmd := p.requestMove("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("")
		if result != nil {
			t.Error("expected nil cmd for empty destination")
		}
	})

	t.Run("ShouldReturnNilWhenDestSameAsSource", func(t *testing.T) {
		cmd := p.requestMove("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("aws_instance.web")
		if result != nil {
			t.Error("expected nil cmd when dest equals source")
		}
	})

	t.Run("ShouldProduceConfirmAfterTextInput", func(t *testing.T) {
		cmd := p.requestMove("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("aws_instance.new_name")
		if nextCmd == nil {
			t.Fatal("expected non-nil cmd after providing destination")
		}
		nextMsg := nextCmd()
		confirmMsg, ok := nextMsg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg for confirm, got %T", nextMsg)
		}
		if confirmMsg.Request.Mode != sdk.InputRequestBool {
			t.Errorf("expected InputRequestBool, got %d", confirmMsg.Request.Mode)
		}
	})

	t.Run("ShouldExecuteMoveOnConfirmation", func(t *testing.T) {
		svc2 := &trackingMockService{}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestMove("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("aws_instance.new_name")
		nextMsg := nextCmd()
		confirmMsg := nextMsg.(sdk.RequestInputMsg)
		execCmd := confirmMsg.Request.Callback("y")
		if execCmd == nil {
			t.Fatal("expected non-nil cmd after confirmation")
		}
		result := execCmd()
		if _, ok := result.(StateMovedMsg); !ok {
			t.Fatalf("expected StateMovedMsg, got %T", result)
		}
		moved := result.(StateMovedMsg)
		if moved.Source != "aws_instance.web" {
			t.Errorf("expected source 'aws_instance.web', got %q", moved.Source)
		}
		if moved.Dest != "aws_instance.new_name" {
			t.Errorf("expected dest 'aws_instance.new_name', got %q", moved.Dest)
		}
		if svc2.moveSource != "aws_instance.web" || svc2.moveDest != "aws_instance.new_name" {
			t.Errorf("service.StateMove not called correctly: source=%q dest=%q", svc2.moveSource, svc2.moveDest)
		}
	})

	t.Run("ShouldReturnErrorOnMoveFailure", func(t *testing.T) {
		svc2 := &trackingMockService{stateMoveErr: errors.New("move failed")}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestMove("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("aws_instance.new_name")
		nextMsg := nextCmd()
		confirmMsg := nextMsg.(sdk.RequestInputMsg)
		execCmd := confirmMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error in StateListMsg")
		}
	})

	t.Run("ShouldReturnNilOnDecline", func(t *testing.T) {
		cmd := p.requestMove("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("aws_instance.new_name")
		nextMsg := nextCmd()
		confirmMsg := nextMsg.(sdk.RequestInputMsg)
		result := confirmMsg.Request.Callback("n")
		if result != nil {
			t.Error("expected nil cmd on decline")
		}
	})
}

func TestRequestTaint_WhenCalledWithAddress_ShouldConfirmThenExecute(t *testing.T) {
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, nil)

	t.Run("ShouldReturnConfirmRequest", func(t *testing.T) {
		cmd := p.requestTaint("aws_instance.web")
		msg := cmd()
		reqMsg, ok := msg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
		}
		if reqMsg.Request.Mode != sdk.InputRequestBool {
			t.Errorf("expected InputRequestBool, got %d", reqMsg.Request.Mode)
		}
	})

	t.Run("ShouldExecuteTaintOnConfirmation", func(t *testing.T) {
		svc2 := &trackingMockService{}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestTaint("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		if execCmd == nil {
			t.Fatal("expected non-nil cmd after confirmation")
		}
		result := execCmd()
		tainted, ok := result.(StateTaintedMsg)
		if !ok {
			t.Fatalf("expected StateTaintedMsg, got %T", result)
		}
		if len(tainted.Addresses) != 1 || tainted.Addresses[0] != "aws_instance.web" {
			t.Errorf("expected [aws_instance.web], got %v", tainted.Addresses)
		}
		if len(svc2.taintCalled) != 1 || svc2.taintCalled[0] != "aws_instance.web" {
			t.Errorf("service.Taint not called correctly: %v", svc2.taintCalled)
		}
	})

	t.Run("ShouldReturnErrorOnTaintFailure", func(t *testing.T) {
		svc2 := &trackingMockService{taintErr: errors.New("taint failed")}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestTaint("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error")
		}
	})

	t.Run("ShouldReturnNilOnDecline", func(t *testing.T) {
		cmd := p.requestTaint("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("n")
		if result != nil {
			t.Error("expected nil cmd on decline")
		}
	})
}

func TestRequestUntaint_WhenCalledWithAddress_ShouldConfirmThenExecute(t *testing.T) {
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, nil)

	t.Run("ShouldReturnConfirmRequest", func(t *testing.T) {
		cmd := p.requestUntaint("aws_instance.web")
		msg := cmd()
		reqMsg, ok := msg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
		}
		if reqMsg.Request.Mode != sdk.InputRequestBool {
			t.Errorf("expected InputRequestBool, got %d", reqMsg.Request.Mode)
		}
	})

	t.Run("ShouldExecuteUntaintOnConfirmation", func(t *testing.T) {
		svc2 := &trackingMockService{}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestUntaint("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		if execCmd == nil {
			t.Fatal("expected non-nil cmd after confirmation")
		}
		result := execCmd()
		untainted, ok := result.(StateUntaintedMsg)
		if !ok {
			t.Fatalf("expected StateUntaintedMsg, got %T", result)
		}
		if len(untainted.Addresses) != 1 || untainted.Addresses[0] != "aws_instance.web" {
			t.Errorf("expected [aws_instance.web], got %v", untainted.Addresses)
		}
		if len(svc2.untaintCalled) != 1 || svc2.untaintCalled[0] != "aws_instance.web" {
			t.Errorf("service.Untaint not called correctly: %v", svc2.untaintCalled)
		}
	})

	t.Run("ShouldReturnErrorOnUntaintFailure", func(t *testing.T) {
		svc2 := &trackingMockService{untaintErr: errors.New("untaint failed")}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestUntaint("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error")
		}
	})

	t.Run("ShouldReturnNilOnDecline", func(t *testing.T) {
		cmd := p.requestUntaint("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("n")
		if result != nil {
			t.Error("expected nil cmd on decline")
		}
	})
}

func TestRequestImport_WhenCalledWithAddress_ShouldPromptIDThenConfirm(t *testing.T) {
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, nil)

	t.Run("ShouldReturnTextInputForResourceID", func(t *testing.T) {
		cmd := p.requestImport("aws_instance.web")
		msg := cmd()
		reqMsg, ok := msg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
		}
		if reqMsg.Request.Mode != sdk.InputRequestText {
			t.Errorf("expected InputRequestText, got %d", reqMsg.Request.Mode)
		}
		if reqMsg.Request.Prompt != "Resource ID:" {
			t.Errorf("expected prompt 'Resource ID:', got %q", reqMsg.Request.Prompt)
		}
	})

	t.Run("ShouldReturnNilWhenIDIsEmpty", func(t *testing.T) {
		cmd := p.requestImport("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("")
		if result != nil {
			t.Error("expected nil cmd for empty ID")
		}
	})

	t.Run("ShouldProduceConfirmAfterIDInput", func(t *testing.T) {
		cmd := p.requestImport("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("i-12345")
		if nextCmd == nil {
			t.Fatal("expected non-nil cmd after providing ID")
		}
		nextMsg := nextCmd()
		confirmMsg, ok := nextMsg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg for confirm, got %T", nextMsg)
		}
		if confirmMsg.Request.Mode != sdk.InputRequestBool {
			t.Errorf("expected InputRequestBool, got %d", confirmMsg.Request.Mode)
		}
	})

	t.Run("ShouldExecuteImportOnConfirmation", func(t *testing.T) {
		svc2 := &trackingMockService{}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestImport("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("i-12345")
		nextMsg := nextCmd()
		confirmMsg := nextMsg.(sdk.RequestInputMsg)
		execCmd := confirmMsg.Request.Callback("y")
		if execCmd == nil {
			t.Fatal("expected non-nil cmd after confirmation")
		}
		result := execCmd()
		imported, ok := result.(StateImportedMsg)
		if !ok {
			t.Fatalf("expected StateImportedMsg, got %T", result)
		}
		if imported.Address != "aws_instance.web" {
			t.Errorf("expected address 'aws_instance.web', got %q", imported.Address)
		}
		if imported.ID != "i-12345" {
			t.Errorf("expected ID 'i-12345', got %q", imported.ID)
		}
		if svc2.importAddr != "aws_instance.web" || svc2.importID != "i-12345" {
			t.Errorf("service.Import not called correctly: addr=%q id=%q", svc2.importAddr, svc2.importID)
		}
	})

	t.Run("ShouldReturnErrorOnImportFailure", func(t *testing.T) {
		svc2 := &trackingMockService{importErr: errors.New("import failed")}
		p2 := newTrackingPlugin(svc2, nil)
		cmd := p2.requestImport("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("i-12345")
		nextMsg := nextCmd()
		confirmMsg := nextMsg.(sdk.RequestInputMsg)
		execCmd := confirmMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error")
		}
	})

	t.Run("ShouldReturnNilOnDecline", func(t *testing.T) {
		cmd := p.requestImport("aws_instance.web")
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		nextCmd := reqMsg.Request.Callback("i-12345")
		nextMsg := nextCmd()
		confirmMsg := nextMsg.(sdk.RequestInputMsg)
		result := confirmMsg.Request.Callback("n")
		if result != nil {
			t.Error("expected nil cmd on decline")
		}
	})
}

func TestBatchTaint_WhenCalledWithMultipleAddresses_ShouldConfirmThenTaintAll(t *testing.T) {
	addresses := []string{"aws_instance.a", "aws_instance.b", "aws_instance.c"}

	t.Run("ShouldReturnConfirmWithCount", func(t *testing.T) {
		svc := &trackingMockService{}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchTaint(addresses)
		msg := cmd()
		reqMsg, ok := msg.(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("expected sdk.RequestInputMsg, got %T", msg)
		}
		if reqMsg.Request.Mode != sdk.InputRequestBool {
			t.Errorf("expected InputRequestBool, got %d", reqMsg.Request.Mode)
		}
	})

	t.Run("ShouldTaintAllOnConfirmation", func(t *testing.T) {
		svc := &trackingMockService{}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchTaint(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		tainted, ok := result.(StateTaintedMsg)
		if !ok {
			t.Fatalf("expected StateTaintedMsg, got %T", result)
		}
		if len(tainted.Addresses) != 3 {
			t.Errorf("expected 3 addresses, got %d", len(tainted.Addresses))
		}
		if len(svc.taintCalled) != 3 {
			t.Errorf("expected 3 taint calls, got %d", len(svc.taintCalled))
		}
	})

	t.Run("ShouldStopOnFirstError", func(t *testing.T) {
		svc := &trackingMockService{taintErr: errors.New("taint failed")}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchTaint(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error")
		}
		if len(svc.taintCalled) != 1 {
			t.Errorf("expected 1 taint call before error, got %d", len(svc.taintCalled))
		}
	})

	t.Run("ShouldReturnNilOnDecline", func(t *testing.T) {
		svc := &trackingMockService{}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchTaint(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("n")
		if result != nil {
			t.Error("expected nil cmd on decline")
		}
	})
}

func TestBatchUntaint_WhenCalledWithMultipleAddresses_ShouldConfirmThenUntaintAll(t *testing.T) {
	addresses := []string{"aws_instance.a", "aws_instance.b"}

	t.Run("ShouldUntaintAllOnConfirmation", func(t *testing.T) {
		svc := &trackingMockService{}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchUntaint(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		untainted, ok := result.(StateUntaintedMsg)
		if !ok {
			t.Fatalf("expected StateUntaintedMsg, got %T", result)
		}
		if len(untainted.Addresses) != 2 {
			t.Errorf("expected 2 addresses, got %d", len(untainted.Addresses))
		}
		if len(svc.untaintCalled) != 2 {
			t.Errorf("expected 2 untaint calls, got %d", len(svc.untaintCalled))
		}
	})

	t.Run("ShouldStopOnFirstError", func(t *testing.T) {
		svc := &trackingMockService{untaintErr: errors.New("untaint failed")}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchUntaint(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error")
		}
	})
}

func TestBatchDelete_WhenCalledWithMultipleAddresses_ShouldConfirmThenDeleteAll(t *testing.T) {
	addresses := []string{"aws_instance.a", "aws_instance.b", "aws_instance.c"}

	t.Run("ShouldDeleteAllOnConfirmation", func(t *testing.T) {
		svc := &trackingMockService{}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchDelete(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		deleted, ok := result.(StateDeletedMsg)
		if !ok {
			t.Fatalf("expected StateDeletedMsg, got %T", result)
		}
		if deleted.Address != "3 resources" {
			t.Errorf("expected '3 resources', got %q", deleted.Address)
		}
		if len(svc.stateRmCalled) != 3 {
			t.Errorf("expected 3 stateRm calls, got %d", len(svc.stateRmCalled))
		}
	})

	t.Run("ShouldStopOnFirstError", func(t *testing.T) {
		svc := &trackingMockService{stateRmErr: errors.New("rm failed")}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchDelete(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		execCmd := reqMsg.Request.Callback("y")
		result := execCmd()
		listMsg, ok := result.(StateListMsg)
		if !ok {
			t.Fatalf("expected StateListMsg on error, got %T", result)
		}
		if listMsg.Err == nil {
			t.Error("expected non-nil error")
		}
	})

	t.Run("ShouldReturnNilOnDecline", func(t *testing.T) {
		svc := &trackingMockService{}
		p := newTrackingPlugin(svc, nil)
		cmd := p.batchDelete(addresses)
		msg := cmd()
		reqMsg := msg.(sdk.RequestInputMsg)
		result := reqMsg.Request.Callback("n")
		if result != nil {
			t.Error("expected nil cmd on decline")
		}
	})
}

func TestActionTargets_WhenPinsExist_ShouldReturnPinnedAddresses(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
		{Address: "aws_instance.b", Type: "aws_instance"},
		{Address: "aws_instance.c", Type: "aws_instance"},
	}
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, resources)

	t.Run("ShouldReturnCursorWhenNoPins", func(t *testing.T) {
		targets := p.actionTargets()
		if len(targets) != 1 || targets[0] != "aws_instance.a" {
			t.Errorf("expected [aws_instance.a], got %v", targets)
		}
	})

	t.Run("ShouldReturnPinnedWhenPinsExist", func(t *testing.T) {
		p.pins.Toggle("aws_instance.b")
		p.pins.Toggle("aws_instance.c")
		targets := p.actionTargets()
		if len(targets) != 2 {
			t.Errorf("expected 2 pinned targets, got %d", len(targets))
		}
	})

	t.Run("ShouldReturnNilWhenNoSelectionAndNoPins", func(t *testing.T) {
		svc2 := &trackingMockService{}
		p2 := newTrackingPlugin(svc2, []sdk.Resource{})
		targets := p2.actionTargets()
		if targets != nil {
			t.Errorf("expected nil targets for empty list, got %v", targets)
		}
	})
}

func TestBuildActionFrame_WhenSingleTarget_ShouldHaveAllActionsEnabled(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, resources)

	t.Run("ShouldUseAddressAsTitle", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		if frame.ID() != "actions" {
			t.Errorf("expected frame ID 'actions', got %q", frame.ID())
		}
	})

	t.Run("ShouldHaveMoveEnabled", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		hints := frame.Hints()
		found := false
		for _, h := range hints {
			if h.Key == "m" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'm' (move) in hints for single target")
		}
	})

	t.Run("ShouldHaveImportEnabled", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		hints := frame.Hints()
		found := false
		for _, h := range hints {
			if h.Key == "n" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'n' (import) in hints for single target")
		}
	})

	t.Run("ShouldExecuteDeleteHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from delete handler")
		}
	})

	t.Run("ShouldExecuteMoveHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from move handler")
		}
	})

	t.Run("ShouldExecuteTaintHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from taint handler")
		}
	})

	t.Run("ShouldExecuteUntaintHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from untaint handler")
		}
	})

	t.Run("ShouldExecuteImportHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from import handler")
		}
	})

	t.Run("ShouldExecuteEditHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.web", false)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from edit handler")
		}
	})
}

func TestBuildActionFrame_WhenMultiplePinned_ShouldDisableMoveAndImport(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
		{Address: "aws_instance.b", Type: "aws_instance"},
	}
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, resources)
	p.pins.Toggle("aws_instance.a")
	p.pins.Toggle("aws_instance.b")
	p.syncPinnedToTree()

	t.Run("ShouldUsePinnedCountInTitle", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		view := frame.View(80, 24)
		if view == "" {
			t.Error("expected non-empty view")
		}
	})

	t.Run("ShouldDisableMoveForMultiTarget", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		hints := frame.Hints()
		for _, h := range hints {
			if h.Key == "m" {
				t.Error("'m' (move) should be disabled for multi-target")
			}
		}
	})

	t.Run("ShouldDisableImportForMultiTarget", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		hints := frame.Hints()
		for _, h := range hints {
			if h.Key == "n" {
				t.Error("'n' (import) should be disabled for multi-target")
			}
		}
	})

	t.Run("ShouldKeepTaintEnabledForMultiTarget", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		hints := frame.Hints()
		found := false
		for _, h := range hints {
			if h.Key == "t" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 't' (taint) in hints for multi-target")
		}
	})

	t.Run("ShouldExecuteBatchDeleteHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from batch delete handler")
		}
	})

	t.Run("ShouldExecuteBatchTaintHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from batch taint handler")
		}
	})

	t.Run("ShouldExecuteBatchUntaintHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
		if result != nil {
			t.Error("expected frame to pop after action key")
		}
		if cmd == nil {
			t.Error("expected non-nil cmd from batch untaint handler")
		}
	})

	t.Run("ShouldNotExecuteDisabledMoveHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		if result == nil {
			t.Error("expected frame to stay (disabled action should not pop)")
		}
		if cmd != nil {
			t.Error("expected nil cmd for disabled action")
		}
	})

	t.Run("ShouldNotExecuteDisabledImportHandler", func(t *testing.T) {
		frame := p.buildActionFrame("aws_instance.a", true)
		result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		if result == nil {
			t.Error("expected frame to stay (disabled action should not pop)")
		}
		if cmd != nil {
			t.Error("expected nil cmd for disabled action")
		}
	})
}

func TestListFrame_WhenActionKeyPressed_ShouldPushActionFrame(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.rebuildTree()

	t.Run("ShouldPushActionFrameOnBang", func(t *testing.T) {
		f := &listFrame{plugin: p}
		f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
		if p.stack.Peek().ID() != "actions" {
			t.Errorf("expected 'actions' frame on stack, got %q", p.stack.Peek().ID())
		}
		p.stack.Pop()
	})

	t.Run("ShouldReturnMoveCmdOnM", func(t *testing.T) {
		f := &listFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 'm' key")
		}
	})

	t.Run("ShouldReturnTaintCmdOnT", func(t *testing.T) {
		f := &listFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 't' key")
		}
	})

	t.Run("ShouldReturnUntaintCmdOnShiftT", func(t *testing.T) {
		f := &listFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 'T' key")
		}
	})

	t.Run("ShouldReturnImportCmdOnN", func(t *testing.T) {
		f := &listFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 'n' key")
		}
	})

	t.Run("ShouldDoNothingWhenNoResourceSelected", func(t *testing.T) {
		emptyP := newTestPlugin([]sdk.Resource{})
		f := &listFrame{plugin: emptyP}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		if cmd != nil {
			t.Error("expected nil cmd when no resource selected")
		}
	})

	t.Run("ShouldPushActionFrameOnBang_WithPinnedOnBranchNode", func(t *testing.T) {
		treeResources := []sdk.Resource{
			{Address: "module.a.aws_instance.one", Type: "aws_instance"},
			{Address: "module.a.aws_instance.two", Type: "aws_instance"},
		}
		tp := newTestPlugin(treeResources)
		tp.pins = sdk.NewPinService()
		tp.treeMode = true
		tp.rebuildTree()
		// Pin a resource
		tp.pins.Toggle("module.a.aws_instance.one")
		tp.syncPinnedToTree()
		// Cursor is on branch node "module.a" — SelectedResource() returns empty
		f := &listFrame{plugin: tp}
		f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
		if tp.stack.Peek().ID() != "actions" {
			t.Errorf("expected 'actions' frame on stack when pinned targets exist, got %q", tp.stack.Peek().ID())
		}
		tp.stack.Pop()
	})
}

func TestDetailFrame_WhenActionKeyPressed_ShouldTriggerAction(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.web", Type: "aws_instance"},
	}
	p := newTestPlugin(resources)
	p.pins = sdk.NewPinService()
	p.status = StatusShowingDetail
	p.detailAddr = "aws_instance.web"
	p.detail = `{"id": "i-123"}`
	p.rebuildTree()

	t.Run("ShouldReturnMoveCmdOnM", func(t *testing.T) {
		f := &detailFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 'm' in detail")
		}
	})

	t.Run("ShouldReturnTaintCmdOnT", func(t *testing.T) {
		f := &detailFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 't' in detail")
		}
	})

	t.Run("ShouldReturnUntaintCmdOnShiftT", func(t *testing.T) {
		f := &detailFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 'T' in detail")
		}
	})

	t.Run("ShouldReturnImportCmdOnN", func(t *testing.T) {
		f := &detailFrame{plugin: p}
		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		if cmd == nil {
			t.Error("expected non-nil cmd for 'n' in detail")
		}
	})
}

func TestUpdate_WhenStateMovedMsg_ShouldTriggerRefresh(t *testing.T) {
	svc := &trackingMockService{mockService: mockService{stateListResult: []sdk.Resource{}}}
	p := newTrackingPlugin(svc, []sdk.Resource{{Address: "a"}})

	_, cmd := p.Update(StateMovedMsg{Source: "a", Dest: "b"})
	if cmd == nil {
		t.Error("expected non-nil cmd (refresh) after StateMovedMsg")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("expected sdk.StatusLoading after refresh, got %v", p.status)
	}
}

func TestUpdate_WhenStateTaintedMsg_ShouldTriggerRefresh(t *testing.T) {
	svc := &trackingMockService{mockService: mockService{stateListResult: []sdk.Resource{}}}
	p := newTrackingPlugin(svc, []sdk.Resource{{Address: "a"}})

	_, cmd := p.Update(StateTaintedMsg{Addresses: []string{"a"}})
	if cmd == nil {
		t.Error("expected non-nil cmd (refresh) after StateTaintedMsg")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("expected sdk.StatusLoading after refresh, got %v", p.status)
	}
}

func TestUpdate_WhenStateUntaintedMsg_ShouldTriggerRefresh(t *testing.T) {
	svc := &trackingMockService{mockService: mockService{stateListResult: []sdk.Resource{}}}
	p := newTrackingPlugin(svc, []sdk.Resource{{Address: "a"}})

	_, cmd := p.Update(StateUntaintedMsg{Addresses: []string{"a"}})
	if cmd == nil {
		t.Error("expected non-nil cmd (refresh) after StateUntaintedMsg")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("expected sdk.StatusLoading after refresh, got %v", p.status)
	}
}

func TestUpdate_WhenStateImportedMsg_ShouldTriggerRefresh(t *testing.T) {
	svc := &trackingMockService{mockService: mockService{stateListResult: []sdk.Resource{}}}
	p := newTrackingPlugin(svc, []sdk.Resource{{Address: "a"}})

	_, cmd := p.Update(StateImportedMsg{Address: "a", ID: "i-123"})
	if cmd == nil {
		t.Error("expected non-nil cmd (refresh) after StateImportedMsg")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("expected sdk.StatusLoading after refresh, got %v", p.status)
	}
}

func TestRequestEditMultiple_ShouldProduceStateEditMsgWithAddresses(t *testing.T) {
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, nil)

	addresses := []string{"aws_instance.a", "aws_instance.b", "aws_instance.c"}
	cmd := p.requestEditMultiple(addresses)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	editMsg, ok := msg.(StateEditMsg)
	if !ok {
		t.Fatalf("expected StateEditMsg, got %T", msg)
	}
	if len(editMsg.Addresses) != 3 {
		t.Errorf("expected 3 addresses, got %d", len(editMsg.Addresses))
	}
	if editMsg.Addresses[0] != "aws_instance.a" {
		t.Errorf("expected first address 'aws_instance.a', got %q", editMsg.Addresses[0])
	}
}

func TestBuildActionFrame_EditHandler_MultiTarget(t *testing.T) {
	resources := []sdk.Resource{
		{Address: "aws_instance.a", Type: "aws_instance"},
		{Address: "aws_instance.b", Type: "aws_instance"},
	}
	svc := &trackingMockService{}
	p := newTrackingPlugin(svc, resources)
	p.pins.Toggle("aws_instance.a")
	p.pins.Toggle("aws_instance.b")
	p.syncPinnedToTree()

	frame := p.buildActionFrame("aws_instance.a", true)
	result, cmd := frame.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if result != nil {
		t.Error("expected frame to pop after 'e'")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from multi-target edit handler")
	}
	msg := cmd()
	editMsg, ok := msg.(StateEditMsg)
	if !ok {
		t.Fatalf("expected StateEditMsg, got %T", msg)
	}
	if len(editMsg.Addresses) != 2 {
		t.Errorf("expected 2 addresses in multi-target edit, got %d", len(editMsg.Addresses))
	}
}
