package state

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func newTrackingPlugin(svc *sdktest.MockService, resources []sdk.Resource) *Plugin {
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
	svc := &sdktest.MockService{}
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
		svc2 := &sdktest.MockService{}
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
		if len(svc2.StateMoveCalls) == 0 || svc2.StateMoveCalls[0][0] != "aws_instance.web" || svc2.StateMoveCalls[0][1] != "aws_instance.new_name" {
			t.Errorf("service.StateMove not called correctly: %v", svc2.StateMoveCalls)
		}
	})

	t.Run("ShouldReturnErrorOnMoveFailure", func(t *testing.T) {
		svc2 := &sdktest.MockService{StateMoveFn: func(_ context.Context, _, _ string) error { return errors.New("move failed") }}
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

func TestBatchDelete_WhenCalledWithMultipleAddresses_ShouldConfirmThenDeleteAll(t *testing.T) {
	addresses := []string{"aws_instance.a", "aws_instance.b", "aws_instance.c"}

	t.Run("ShouldDeleteAllOnConfirmation", func(t *testing.T) {
		svc := &sdktest.MockService{}
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
		if len(svc.StateRmCalls) != 3 {
			t.Errorf("expected 3 stateRm calls, got %d", len(svc.StateRmCalls))
		}
	})

	t.Run("ShouldStopOnFirstError", func(t *testing.T) {
		svc := &sdktest.MockService{StateRmFn: func(_ context.Context, _ string) error { return errors.New("rm failed") }}
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
		svc := &sdktest.MockService{}
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
	svc := &sdktest.MockService{}
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
		svc2 := &sdktest.MockService{}
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
	svc := &sdktest.MockService{}
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
	svc := &sdktest.MockService{}
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
	svc := &sdktest.MockService{StateListFn: func(_ context.Context, _ ...sdk.StateListOption) ([]sdk.Resource, error) {
		return []sdk.Resource{}, nil
	}}
	p := newTrackingPlugin(svc, []sdk.Resource{{Address: "a"}})

	_, cmd := p.Update(StateMovedMsg{Source: "a", Dest: "b"})
	if cmd == nil {
		t.Error("expected non-nil cmd (refresh) after StateMovedMsg")
	}
	if p.status != sdk.StatusLoading {
		t.Errorf("expected sdk.StatusLoading after refresh, got %v", p.status)
	}
}

func TestRequestEditMultiple_ShouldProduceStateEditMsgWithAddresses(t *testing.T) {
	svc := &sdktest.MockService{}
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
	svc := &sdktest.MockService{}
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

func TestBuildActionFrame_WhenMultiTarget_ShouldDisableMoveAndImport(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	if frame == nil {
		t.Fatal("buildActionFrame returned nil")
	}
}

func TestBuildActionFrame_WhenSingleTarget_ShouldEnableAllActions(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	if frame == nil {
		t.Fatal("buildActionFrame returned nil")
	}
}

func TestBuildActionFrame_WhenMultiTargetEditHandler_ShouldEditMultiple(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	if frame == nil {
		t.Fatal("buildActionFrame returned nil")
	}

	// Push the frame and trigger the 'e' action (edit)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for edit action in multi-target mode")
	}
}

func TestBuildActionFrame_WhenMultiTargetDeleteHandler_ShouldBatchDelete(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for delete action in multi-target mode")
	}
}

func TestBuildActionFrame_WhenSingleTargetTaintHandler_ShouldEmitTaintRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for taint action")
	}
}

func TestBuildActionFrame_WhenSingleTargetUntaintHandler_ShouldEmitUntaintRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for untaint action")
	}
}

func TestBuildActionFrame_WhenSingleTargetImportHandler_ShouldEmitImportRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for import action")
	}
}

func TestBuildActionFrame_WhenSingleTargetMoveHandler_ShouldEmitMoveRequest(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("a", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for move action")
	}
}

func TestBuildActionFrame_WhenSingleTargetTaintHandlerExecuted_ShouldProduceTaintMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "aws_instance.web"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("aws_instance.web", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for taint action")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil for taint action")
	}
}

func TestBuildActionFrame_WhenSingleTargetUntaintHandlerExecuted_ShouldProduceUntaintMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "aws_instance.web"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("aws_instance.web", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for untaint action")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil for untaint action")
	}
}

func TestBuildActionFrame_WhenSingleTargetImportHandlerExecuted_ShouldProduceImportMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "aws_instance.web"}}
	p.filtered = p.resources
	p.rebuildTree()

	frame := p.buildActionFrame("aws_instance.web", false)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for import action")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil for import action")
	}
}

func TestBuildActionFrame_WhenMultiTargetTaintHandlerExecuted_ShouldProduceTaintMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for multi-target taint action")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil for multi-target taint action")
	}
}

func TestBuildActionFrame_WhenMultiTargetUntaintHandlerExecuted_ShouldProduceUntaintMsg(t *testing.T) {
	p := New(&sdktest.MockService{}).(*Plugin)
	p.log = slog.New(slog.NewTextHandler(io.Discard, nil))
	p.pins = sdk.NewPinService()
	p.resources = []sdk.Resource{{Address: "a"}, {Address: "b"}}
	p.filtered = p.resources
	p.rebuildTree()
	p.pins.Toggle("a")
	p.pins.Toggle("b")

	frame := p.buildActionFrame("a", true)
	p.stack.Push(frame)
	cmd := p.stack.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for multi-target untaint action")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("cmd() returned nil for multi-target untaint action")
	}
}
