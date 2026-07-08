package sdk_test

import (
	"bytes"
	"log/slog"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func TestNewPluginBase_PopulatesMetadataAndDiscardLogger(t *testing.T) {
	b := sdk.NewPluginBase("plan", "Plan", "Review terraform plan changes")

	if got := b.ID(); got != "plan" {
		t.Errorf("ID() = %q, want %q", got, "plan")
	}
	if got := b.Name(); got != "Plan" {
		t.Errorf("Name() = %q, want %q", got, "Plan")
	}
	if got := b.Description(); got != "Review terraform plan changes" {
		t.Errorf("Description() = %q, want %q", got, "Review terraform plan changes")
	}
	if b.Log == nil {
		t.Fatal("Log was nil; expected discard logger")
	}
	b.Log.Info("smoke") // must not panic
}

func TestPluginBase_InitBase_AssignsAllDeps(t *testing.T) {
	b := sdk.NewPluginBase("p", "P", "")
	svc := &sdktest.MockService{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	ctx := &sdk.Context{WorkingDir: "/tmp"}
	pinCalls := 0
	clearCalls := 0

	deps := &sdk.PluginDeps{
		Logger:    logger,
		Service:   svc,
		Context:   func() *sdk.Context { return ctx },
		Pin:       func(string) tea.Cmd { pinCalls++; return nil },
		ClearPins: func() tea.Cmd { clearCalls++; return nil },
	}

	b.InitBase(deps)

	if b.Svc != svc {
		t.Errorf("Svc not assigned; got %#v", b.Svc)
	}
	if b.Log != logger {
		t.Error("Log not replaced by deps.Logger")
	}
	if got := b.GetCtx(); got != ctx {
		t.Errorf("GetCtx() returned %p, want %p", got, ctx)
	}
	b.PinFn("addr")
	b.ClearPinsFn()
	if pinCalls != 1 || clearCalls != 1 {
		t.Errorf("expected 1 pin / 1 clear, got %d / %d", pinCalls, clearCalls)
	}
}

func TestPluginBase_InitBase_KeepsDiscardLoggerWhenDepsLoggerNil(t *testing.T) {
	b := sdk.NewPluginBase("p", "P", "")
	prev := b.Log

	b.InitBase(&sdk.PluginDeps{Logger: nil})

	if b.Log != prev {
		t.Error("Log replaced even though deps.Logger was nil")
	}
}

func TestPluginBase_HandleContextChangedDefault(t *testing.T) {
	original := &sdktest.MockService{}
	replacement := &sdktest.MockService{}

	tests := []struct {
		name    string
		ev      sdk.ContextChangedEvent
		wantOK  bool
		wantSvc sdk.Service
	}{
		{
			name:    "next nil → no-op, returns false",
			ev:      sdk.ContextChangedEvent{Next: nil},
			wantOK:  false,
			wantSvc: original,
		},
		{
			name:    "next without service keeps current Svc",
			ev:      sdk.ContextChangedEvent{Next: &sdk.Context{}},
			wantOK:  true,
			wantSvc: original,
		},
		{
			name:    "next with service rebinds Svc",
			ev:      sdk.ContextChangedEvent{Next: &sdk.Context{Service: replacement}},
			wantOK:  true,
			wantSvc: replacement,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := sdk.NewPluginBase("p", "P", "")
			b.Svc = original

			got := b.HandleContextChangedDefault(tt.ev)

			if got != tt.wantOK {
				t.Errorf("HandleContextChangedDefault() = %v, want %v", got, tt.wantOK)
			}
			if b.Svc != tt.wantSvc {
				t.Errorf("Svc = %p, want %p", b.Svc, tt.wantSvc)
			}
		})
	}
}

func TestPluginBase_ReactToContext(t *testing.T) {
	original := &sdktest.MockService{}
	replacement := &sdktest.MockService{}

	t.Run("next nil → neither callback fires, svc unchanged", func(t *testing.T) {
		b := sdk.NewPluginBase("p", "P", "")
		b.Svc = original
		var pins, reset bool
		cmd := b.ReactToContext(sdk.ContextChangedEvent{Next: nil},
			func() { pins = true }, func() { reset = true })
		if cmd != nil || pins || reset {
			t.Errorf("nil Next should be a no-op; cmd=%v pins=%v reset=%v", cmd, pins, reset)
		}
		if b.Svc != original {
			t.Error("svc should be unchanged on nil Next")
		}
	})

	t.Run("pins-only → onPins fires, no reset, svc not rebound", func(t *testing.T) {
		b := sdk.NewPluginBase("p", "P", "")
		b.Svc = original
		var pins, reset bool
		b.ReactToContext(sdk.ContextChangedEvent{
			Next:   &sdk.Context{Service: replacement},
			Reason: sdk.ContextPinsChanged,
		}, func() { pins = true }, func() { reset = true })
		if !pins || reset {
			t.Errorf("pins-only should run onPins only; pins=%v reset=%v", pins, reset)
		}
		if b.Svc != original {
			t.Error("svc must not rebind on a pins-only change")
		}
	})

	t.Run("pins-only with nil onPins → safe no-op", func(t *testing.T) {
		b := sdk.NewPluginBase("p", "P", "")
		var reset bool
		b.ReactToContext(sdk.ContextChangedEvent{
			Next:   &sdk.Context{},
			Reason: sdk.ContextPinsChanged,
		}, nil, func() { reset = true })
		if reset {
			t.Error("reset should not fire on pins-only change")
		}
	})

	t.Run("switch → svc rebound then onReset fires", func(t *testing.T) {
		b := sdk.NewPluginBase("p", "P", "")
		b.Svc = original
		var pins, reset bool
		b.ReactToContext(sdk.ContextChangedEvent{
			Next:   &sdk.Context{Service: replacement},
			Reason: sdk.ContextSwitched,
		}, func() { pins = true }, func() { reset = true })
		if pins || !reset {
			t.Errorf("switch should run onReset only; pins=%v reset=%v", pins, reset)
		}
		if b.Svc != replacement {
			t.Error("svc should rebind to next.Service on a switch")
		}
	})
}

func TestPluginBase_PinnedAddresses_ReturnsContextPins(t *testing.T) {
	ctx := &sdk.Context{Pins: []string{"a", "b"}}
	b := sdk.NewPluginBase("p", "P", "")
	b.GetCtx = func() *sdk.Context { return ctx }

	got := b.PinnedAddresses()

	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("PinnedAddresses() = %v, want [a b]", got)
	}
	if c := b.PinnedCount(); c != 2 {
		t.Errorf("PinnedCount() = %d, want 2", c)
	}
}

func TestPluginBase_PinnedAddresses_BeforeInit_IsNilSafe(t *testing.T) {
	b := sdk.NewPluginBase("p", "P", "")

	if got := b.PinnedAddresses(); got != nil {
		t.Errorf("PinnedAddresses() before Init = %v, want nil", got)
	}
	if got := b.PinnedCount(); got != 0 {
		t.Errorf("PinnedCount() before Init = %d, want 0", got)
	}
	if b.HasPins() {
		t.Error("HasPins() before Init = true, want false")
	}
	if b.IsPinned("anything") {
		t.Error("IsPinned() before Init = true, want false")
	}
}

func TestPluginBase_HasPinsAndIsPinned(t *testing.T) {
	ctx := &sdk.Context{Pins: []string{"aws_instance.a", "aws_instance.b"}}
	b := sdk.NewPluginBase("p", "P", "")
	b.GetCtx = func() *sdk.Context { return ctx }

	if !b.HasPins() {
		t.Error("HasPins() = false, want true")
	}
	if !b.IsPinned("aws_instance.a") {
		t.Error("IsPinned(a) = false, want true")
	}
	if b.IsPinned("aws_instance.missing") {
		t.Error("IsPinned(missing) = true, want false")
	}
}
