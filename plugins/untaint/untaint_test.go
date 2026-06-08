package untaint

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

func TestPlugin_Lifecycle(t *testing.T) {
	svc := &sdktest.MockService{}
	p := New(svc)

	if p.ID() != "untaint" {
		t.Errorf("ID() = %q, want %q", p.ID(), "untaint")
	}
	if p.Name() != "Untaint" {
		t.Errorf("Name() = %q, want %q", p.Name(), "Untaint")
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

func TestPlugin_ImplementsCancellableAndBusy(t *testing.T) {
	p := New(&sdktest.MockService{})
	if _, ok := p.(sdk.Cancellable); !ok {
		t.Error("untaint must implement sdk.Cancellable (promoted from ActionRunner)")
	}
	if _, ok := p.(sdk.Busy); !ok {
		t.Error("untaint must implement sdk.Busy (promoted from ActionRunner)")
	}
}

func TestPlugin_DoesNotImplementStdoutEmitter(t *testing.T) {
	p := New(&sdktest.MockService{})
	if _, ok := p.(sdk.StdoutEmitter); ok {
		t.Error("untaint must not implement sdk.StdoutEmitter (no stdout content)")
	}
}

func TestPlugin_Activate(t *testing.T) {
	t.Run("stores input and addresses, requests confirmation", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		cmd := p.Activate(Input{Addrs: []string{"aws_instance.web", "aws_instance.db"}, JSON: true})
		if cmd == nil {
			t.Fatal("Activate() should return a confirm cmd")
		}
		if len(p.addresses) != 2 || p.addresses[0] != "aws_instance.web" {
			t.Errorf("addresses = %v, want the two supplied addrs", p.addresses)
		}
		if !p.input.JSON {
			t.Error("Input should be stored on plugin state")
		}
		req, ok := cmd().(sdk.RequestInputMsg)
		if !ok {
			t.Fatalf("Activate cmd = %T, want sdk.RequestInputMsg", cmd())
		}
		if req.Request.Mode != sdk.InputRequestBool {
			t.Errorf("request mode = %v, want InputRequestBool", req.Request.Mode)
		}
	})

	t.Run("no addresses deactivates", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		cmd := p.Activate(Input{Addrs: nil})
		if cmd == nil {
			t.Fatal("Activate() with no addresses should return a cmd")
		}
		if _, ok := cmd().(sdk.DeactivateMsg); !ok {
			t.Errorf("Activate cmd = %T, want sdk.DeactivateMsg", cmd())
		}
	})

	t.Run("while busy returns nil", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{Addrs: []string{"a"}})().(sdk.RequestInputMsg)
		req.Request.Callback("y") // → Start → Loading
		if !p.Busy() {
			t.Fatal("precondition: plugin should be Busy after confirm")
		}
		if cmd := p.Activate(Input{Addrs: []string{"b"}}); cmd != nil {
			t.Error("Activate() while busy should return nil")
		}
	})
}

func TestPlugin_Confirm(t *testing.T) {
	t.Run("single-address prompt and accept starts the run", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{Addrs: []string{"aws_instance.web"}})().(sdk.RequestInputMsg)
		if !strings.Contains(req.Request.Prompt, "aws_instance.web") {
			t.Errorf("prompt = %q, want the address", req.Request.Prompt)
		}
		if start := req.Request.Callback("y"); start == nil {
			t.Error("accepting confirmation should return a start cmd")
		}
		if !p.Busy() {
			t.Error("accepting confirmation should put the runner in Loading")
		}
	})

	t.Run("multi-address prompt lists count", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{Addrs: []string{"a", "b", "c"}})().(sdk.RequestInputMsg)
		if !strings.Contains(req.Request.Prompt, "3 resources") {
			t.Errorf("prompt = %q, want '3 resources'", req.Request.Prompt)
		}
	})

	t.Run("declining returns nil", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		req := p.Activate(Input{Addrs: []string{"a"}})().(sdk.RequestInputMsg)
		if req.Request.Callback("n") != nil {
			t.Error("declining confirmation should return nil")
		}
	})
}

func TestPlugin_Spec(t *testing.T) {
	t.Run("metadata wires terraform untaint semantics", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.addresses = []string{"a"}
		spec := p.spec()
		if spec.Name != "untaint" {
			t.Errorf("Name = %q, want untaint", spec.Name)
		}
		if !spec.OfferPlan {
			t.Error("untaint should offer the plan shortcut on success")
		}
		if spec.ErrorLabel != "Untaint failed" {
			t.Errorf("ErrorLabel = %q", spec.ErrorLabel)
		}
		if len(spec.OnSuccess) != 1 {
			t.Fatalf("OnSuccess len = %d, want 1", len(spec.OnSuccess))
		}
		if _, ok := spec.OnSuccess[0].(sdk.PlanInvalidatedEvent); !ok {
			t.Errorf("OnSuccess[0] = %T, want PlanInvalidatedEvent", spec.OnSuccess[0])
		}
	})

	t.Run("Run untaints every address in order", func(t *testing.T) {
		var calls []string
		svc := &sdktest.MockService{UntaintFn: func(_ context.Context, addr string) error {
			calls = append(calls, addr)
			return nil
		}}
		p, _ := newTestPlugin(svc)
		p.addresses = []string{"a", "b"}
		done, err := p.spec().Run(context.Background())
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if strings.Join(done, ",") != "a,b" || strings.Join(calls, ",") != "a,b" {
			t.Errorf("done=%v calls=%v, want both [a b]", done, calls)
		}
	})

	t.Run("Run stops at the first failure, reporting prior successes", func(t *testing.T) {
		n := 0
		svc := &sdktest.MockService{UntaintFn: func(_ context.Context, addr string) error {
			n++
			if n >= 2 {
				return errors.New("denied")
			}
			return nil
		}}
		p, _ := newTestPlugin(svc)
		p.addresses = []string{"a", "b", "c"}
		done, err := p.spec().Run(context.Background())
		if err == nil || !strings.Contains(err.Error(), "b") {
			t.Errorf("err = %v, want failure mentioning 'b'", err)
		}
		if len(done) != 1 || done[0] != "a" {
			t.Errorf("done = %v, want [a] (succeeded before failure)", done)
		}
	})

	t.Run("labels reflect single vs multiple addresses", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.addresses = []string{"aws_instance.web"}
		if got := p.spec().Running(); got != "Untainting aws_instance.web" {
			t.Errorf("Running() = %q", got)
		}
		if got := p.spec().Done([]string{"aws_instance.web"}); got != "✓ Untainted aws_instance.web" {
			t.Errorf("Done(single) = %q", got)
		}
		p.addresses = []string{"a", "b", "c"}
		if got := p.spec().Running(); got != "Untainting 3 resources" {
			t.Errorf("Running(multi) = %q", got)
		}
		if got := p.spec().Done([]string{"a", "b", "c"}); got != "✓ Untainted 3 resources" {
			t.Errorf("Done(multi) = %q", got)
		}
		if p.spec().Idle == "" {
			t.Error("Idle text should be set")
		}
	})
}

func TestPlugin_Update(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})

	if _, cmd := p.Update(ui.TimerTickMsg{}); cmd != nil {
		t.Error("idle timer tick should be handled with no cmd")
	}

	_, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("esc should return a cmd")
	}
	if _, ok := cmd().(sdk.DeactivateMsg); !ok {
		t.Errorf("esc cmd = %T, want DeactivateMsg", cmd())
	}

	if self, cmd := p.Update(struct{}{}); self.(*Plugin) != p || cmd != nil {
		t.Error("unknown msg should return same plugin and nil cmd")
	}
}

func TestPlugin_View_DelegatesToRunner(t *testing.T) {
	p, _ := newTestPlugin(&sdktest.MockService{})
	p.Activate(Input{Addrs: []string{"a"}})
	if !strings.Contains(p.View(80, 24), "Waiting for confirmation") {
		t.Errorf("View() = %q, want the armed idle text", p.View(80, 24))
	}
}

func TestPlugin_HandleContextChanged(t *testing.T) {
	t.Run("resets and clears addresses", func(t *testing.T) {
		svc := &sdktest.MockService{}
		p, _ := newTestPlugin(svc)
		p.Activate(Input{Addrs: []string{"a", "b"}})
		cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: &sdk.Context{Service: svc}})
		if cmd != nil {
			t.Error("HandleContextChanged should return nil cmd")
		}
		if p.addresses != nil {
			t.Errorf("addresses = %v, want nil", p.addresses)
		}
		if p.CurrentStatus() != sdk.StatusIdle {
			t.Errorf("status = %v, want StatusIdle", p.CurrentStatus())
		}
	})

	t.Run("nil Next is a no-op", func(t *testing.T) {
		p, _ := newTestPlugin(&sdktest.MockService{})
		p.addresses = []string{"keep"}
		if cmd := p.HandleContextChanged(sdk.ContextChangedEvent{Next: nil}); cmd != nil {
			t.Error("nil Next should return nil cmd")
		}
		if len(p.addresses) != 1 {
			t.Errorf("addresses mutated on no-op: %v", p.addresses)
		}
	})
}
