package tfimport

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	"github.com/lmarqs/terraform-ui/pkg/sdk/ui"
)

func newTestPlugin(svc sdk.Service) (*Plugin, *sdktest.PluginDepsHarness) {
	h := sdktest.NewDeps(svc)
	p := New(svc).(*Plugin)
	p.Init(h.Deps)
	return p, h
}

// driveToTerminal arms the spec, runs the work synchronously, and feeds the
// result back so the runner reaches Done/Error. Returns the event cmd.
func driveToTerminal(t *testing.T, p *Plugin) tea.Cmd {
	t.Helper()
	p.Arm(p.spec())
	batch, ok := p.Start()().(tea.BatchMsg)
	if !ok {
		t.Fatal("Start() should return a batch")
	}
	_, event := p.Update(batch[0]()) // batch[0] is the work cmd → actionResultMsg
	return event
}

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "import" {
		t.Errorf("ID() = %q, want %q", p.ID(), "import")
	}
	if p.Name() != "Import" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Import")
	}
	if p.Description() == "" {
		t.Error("Description() should not be empty")
	}
	if err := p.Configure(map[string]interface{}{}); err != nil {
		t.Errorf("Configure() = %v, want nil", err)
	}
	if cmd := p.Init(sdktest.NewDeps(svc).Deps); cmd != nil {
		t.Error("Init() should return nil cmd")
	}
	if p.Ready() {
		t.Error("Ready() should be false before activation")
	}
}

func TestPlugin_OptionalInterfaces(t *testing.T) {
	p := New(&sdktest.MockService{})
	if _, ok := p.(sdk.Cancellable); !ok {
		t.Error("import must implement sdk.Cancellable (promoted from ActionRunner)")
	}
	if _, ok := p.(sdk.Busy); !ok {
		t.Error("import must implement sdk.Busy (promoted from ActionRunner)")
	}
	if _, ok := p.(sdk.StdoutEmitter); ok {
		t.Error("import must not implement sdk.StdoutEmitter (no stdout content)")
	}
}

func TestPlugin_Activate(t *testing.T) {
	t.Run("both address and ID skip the form and confirm", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		cmd := p.Activate(Input{Addr: "aws_instance.web", ID: "i-abc", JSON: true})
		if cmd == nil {
			t.Fatal("Activate() should return a confirm cmd")
		}
		if p.address != "aws_instance.web" || p.id != "i-abc" {
			t.Errorf("address/id = %q/%q", p.address, p.id)
		}
		if !p.input.JSON {
			t.Error("Input should be stored on plugin state")
		}
		req := cmd().(sdk.RequestInputMsg)
		if req.Request.Mode != sdk.InputRequestBool {
			t.Errorf("mode = %v, want InputRequestBool (confirm)", req.Request.Mode)
		}
	})

	t.Run("only address pre-fills the form", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		cmd := p.Activate(Input{Addr: "aws_instance.web"})
		req := cmd().(sdk.RequestInputMsg)
		if req.Request.Mode != sdk.InputRequestText {
			t.Errorf("mode = %v, want InputRequestText (form)", req.Request.Mode)
		}
		if req.Request.Default != "aws_instance.web" {
			t.Errorf("default = %q, want the address", req.Request.Default)
		}
		if p.address != "aws_instance.web" {
			t.Errorf("address = %q, want pre-filled", p.address)
		}
	})

	t.Run("no input opens an empty form", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{})().(sdk.RequestInputMsg)
		if req.Request.Default != "" {
			t.Errorf("default = %q, want empty", req.Request.Default)
		}
	})

	t.Run("while busy returns nil", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{Addr: "a", ID: "b"})().(sdk.RequestInputMsg)
		req.Request.Callback("y") // → Start → Loading
		if !p.Busy() {
			t.Fatal("precondition: should be Busy after confirm")
		}
		if cmd := p.Activate(Input{Addr: "x", ID: "y"}); cmd != nil {
			t.Error("Activate() while busy should return nil")
		}
	})
}

func TestPlugin_Form(t *testing.T) {
	t.Run("empty address deactivates", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{})().(sdk.RequestInputMsg)
		cmd := req.Request.Callback("")
		if _, ok := cmd().(sdk.DeactivateMsg); !ok {
			t.Errorf("empty address → %T, want DeactivateMsg", cmd())
		}
	})

	t.Run("empty ID deactivates", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		addrReq := p.Activate(Input{})().(sdk.RequestInputMsg)
		idReq := addrReq.Request.Callback("aws_instance.web")().(sdk.RequestInputMsg)
		if idReq.Request.Mode != sdk.InputRequestText {
			t.Errorf("ID prompt mode = %v, want InputRequestText", idReq.Request.Mode)
		}
		cmd := idReq.Request.Callback("")
		if _, ok := cmd().(sdk.DeactivateMsg); !ok {
			t.Errorf("empty ID → %T, want DeactivateMsg", cmd())
		}
	})

	t.Run("completed form submits address and ID, then confirms", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		addrReq := p.Activate(Input{})().(sdk.RequestInputMsg)
		idReq := addrReq.Request.Callback("aws_instance.web")().(sdk.RequestInputMsg)
		submit := idReq.Request.Callback("i-123")().(importSubmitMsg)
		if submit.Address != "aws_instance.web" || submit.ID != "i-123" {
			t.Errorf("submit = %+v", submit)
		}
		_, confirmCmd := p.Update(submit)
		if confirmCmd == nil {
			t.Fatal("submit should produce a confirm cmd")
		}
		if p.address != "aws_instance.web" || p.id != "i-123" {
			t.Errorf("submit should store address/id, got %q/%q", p.address, p.id)
		}
		req := confirmCmd().(sdk.RequestInputMsg)
		if req.Request.Mode != sdk.InputRequestBool {
			t.Errorf("confirm mode = %v, want InputRequestBool", req.Request.Mode)
		}
	})

	t.Run("declining the confirm returns nil", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		_, confirmCmd := p.Update(importSubmitMsg{Address: "a", ID: "b"})
		req := confirmCmd().(sdk.RequestInputMsg)
		if req.Request.Callback("n") != nil {
			t.Error("declining should return nil")
		}
	})
}

func TestPlugin_Spec(t *testing.T) {
	t.Run("metadata wires terraform import semantics", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.address, p.id = "aws_instance.web", "i-1"
		spec := p.spec()
		if spec.Name != "import" || spec.ErrorLabel != "Import failed" || !spec.OfferPlan {
			t.Errorf("spec metadata = %+v", spec)
		}
		if len(spec.OnSuccess) != 2 {
			t.Fatalf("OnSuccess len = %d, want 2", len(spec.OnSuccess))
		}
		if _, ok := spec.OnSuccess[0].(sdk.StateRefreshedEvent); !ok {
			t.Errorf("OnSuccess[0] = %T, want StateRefreshedEvent", spec.OnSuccess[0])
		}
		if _, ok := spec.OnSuccess[1].(sdk.PlanInvalidatedEvent); !ok {
			t.Errorf("OnSuccess[1] = %T, want PlanInvalidatedEvent", spec.OnSuccess[1])
		}
		if spec.Running() != "Importing aws_instance.web" {
			t.Errorf("Running() = %q", spec.Running())
		}
		if got := spec.Done(nil); got != "✓ Imported i-1 as aws_instance.web" {
			t.Errorf("Done() = %q", got)
		}
	})

	t.Run("Run imports address with ID", func(t *testing.T) {
		svc := &sdktest.MockService{}
		p, _ := newTestPlugin(svc)
		p.address, p.id = "aws_instance.web", "i-1234"
		done, err := p.spec().Run(context.Background())
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if len(done) != 1 || done[0] != "aws_instance.web" {
			t.Errorf("done = %v, want [aws_instance.web]", done)
		}
		if len(svc.ImportCalls) != 1 || svc.ImportCalls[0][0] != "aws_instance.web" || svc.ImportCalls[0][1] != "i-1234" {
			t.Errorf("ImportCalls = %v, want one call with addr+id", svc.ImportCalls)
		}
	})

	t.Run("Run surfaces import errors", func(t *testing.T) {
		svc := &sdktest.MockService{ImportFn: func(context.Context, string, string) error {
			return errors.New("already managed")
		}}
		p, _ := newTestPlugin(svc)
		p.address, p.id = "a", "b"
		if _, err := p.spec().Run(context.Background()); err == nil {
			t.Error("Run() should surface the import error")
		}
	})
}

func TestPlugin_DriveToDone_EmitsEvents(t *testing.T) {
	svc := &sdktest.MockService{}
	p, _ := newTestPlugin(svc)
	p.address, p.id = "aws_instance.web", "i-1"

	eventCmd := driveToTerminal(t, p)
	if !p.Ready() {
		t.Fatalf("status = %v, want Done", p.CurrentStatus())
	}
	if eventCmd == nil {
		t.Fatal("success should emit events")
	}
	batch, ok := eventCmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("events = %T, want batch", eventCmd())
	}
	var refreshed, invalidated bool
	for _, sub := range batch {
		switch sub().(type) {
		case sdk.StateRefreshedEvent:
			refreshed = true
		case sdk.PlanInvalidatedEvent:
			invalidated = true
		}
	}
	if !refreshed || !invalidated {
		t.Errorf("events: refreshed=%v invalidated=%v, want both", refreshed, invalidated)
	}
}

func TestPlugin_DriveToError_SetsMessage(t *testing.T) {
	svc := &sdktest.MockService{ImportFn: func(context.Context, string, string) error {
		return errors.New("boom")
	}}
	p, _ := newTestPlugin(svc)
	p.address, p.id = "a", "b"

	eventCmd := driveToTerminal(t, p)
	if p.CurrentStatus() != sdk.StatusError {
		t.Errorf("status = %v, want Error", p.CurrentStatus())
	}
	if p.ErrMessage() == "" {
		t.Error("ErrMessage should be set on failure")
	}
	if eventCmd != nil {
		t.Error("failure should not emit events")
	}
}

func TestPlugin_Update(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})

	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd != nil {
		t.Error("idle timer tick should be handled with no cmd")
	}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should route to StandardKeys and return a cmd")
	}
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Errorf("esc cmd = %T, want DeactivateMsg", cmd())
	}

	if self, cmd := p.Update(struct{}{}); self.(*Plugin) != p || cmd != nil {
		t.Error("unknown msg should return same plugin and nil cmd")
	}
}

func TestPlugin_View(t *testing.T) {
	t.Run("idle shows the form placeholder", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		if !strings.Contains(p.View(80, 24), "Import resource into terraform state") {
			t.Errorf("idle view = %q", p.View(80, 24))
		}
	})

	t.Run("done delegates to the runner", func(t *testing.T) {
		svc := &sdktest.MockService{}
		p, _ := newTestPlugin(svc)
		p.address, p.id = "aws_instance.web", "i-1"
		driveToTerminal(t, p)
		if !strings.Contains(p.View(80, 24), "Imported") {
			t.Errorf("done view = %q, want imported message", p.View(80, 24))
		}
	})
}

func TestPlugin_Hints(t *testing.T) {
	t.Run("idle offers quit", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		hints := p.Hints()
		if len(hints) == 0 || hints[len(hints)-1].Description != "quit" {
			t.Errorf("idle hints = %+v, want quit", hints)
		}
	})

	t.Run("done offers plan and cancel", func(t *testing.T) {
		svc := &sdktest.MockService{}
		p, _ := newTestPlugin(svc)
		p.address, p.id = "a", "b"
		driveToTerminal(t, p)
		hints := p.Hints()
		if len(hints) != 2 || hints[0].Key != "p" {
			t.Errorf("done hints = %+v, want [{p plan} {Esc cancel}]", hints)
		}
	})

	t.Run("error offers retry and quit", func(t *testing.T) {
		svc := &sdktest.MockService{ImportFn: func(context.Context, string, string) error {
			return errors.New("x")
		}}
		p, _ := newTestPlugin(svc)
		p.address, p.id = "a", "b"
		driveToTerminal(t, p)
		var retry, quit bool
		for _, h := range p.Hints() {
			if h.Description == "retry" {
				retry = true
			}
			if h.Description == "quit" {
				quit = true
			}
		}
		if !retry || !quit {
			t.Errorf("error hints: retry=%v quit=%v, want both", retry, quit)
		}
	})
}

func TestPlugin_HandleContextChanged(t *testing.T) {
	t.Run("resets and clears address/id", func(t *testing.T) {
		svc := &sdktest.MockService{}
		p, _ := newTestPlugin(svc)
		p.address, p.id = "old.addr", "old-id"
		cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})
		if cmd != nil {
			t.Error("HandleContextChanged should return nil cmd")
		}
		if p.address != "" || p.id != "" || p.CurrentStatus() != sdk.StatusIdle {
			t.Errorf("not reset: addr=%q id=%q status=%v", p.address, p.id, p.CurrentStatus())
		}
	})

	t.Run("nil Next is a no-op", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.address = "keep"
		if cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil}); cmd != nil {
			t.Error("nil Next should return nil cmd")
		}
		if p.address != "keep" {
			t.Errorf("address mutated on no-op: %q", p.address)
		}
	})
}
