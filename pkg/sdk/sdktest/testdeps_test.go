package sdktest

import (
	"reflect"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestNewDeps_SeedsContextAndDeps(t *testing.T) {
	svc := &MockService{}

	h := NewDeps(svc)

	if h.Ctx == nil || h.Ctx.Service != svc {
		t.Fatalf("expected Ctx seeded with svc, got %+v", h.Ctx)
	}
	if h.Deps == nil || h.Deps.Service != svc {
		t.Fatalf("expected Deps.Service to be svc")
	}
	if h.Deps.Logger == nil {
		t.Fatal("expected Logger non-nil")
	}
	if got := h.Deps.Context(); got != h.Ctx {
		t.Errorf("Context() = %p, want %p", got, h.Ctx)
	}
}

func TestNewDeps_ContextReflectsLiveSwap(t *testing.T) {
	h := NewDeps(&MockService{})

	next := &sdk.Context{WorkingDir: "/elsewhere"}
	h.Ctx = next

	if got := h.Deps.Context(); got != next {
		t.Errorf("Context() did not return live pointer; got %p want %p", got, next)
	}
}

func TestNewDeps_PinRecordsAddressesAndEmitsRequest(t *testing.T) {
	group := []string{"module.repos.github_branch.main", "module.repos.github_repository.this"}

	tests := []struct {
		name      string
		addresses []string
	}{
		{name: "single address", addresses: []string{"aws_s3_bucket.x"}},
		{name: "group from one gesture", addresses: group},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDeps(&MockService{})

			cmd := h.Deps.Pin(tt.addresses...)
			if cmd == nil {
				t.Fatal("Pin returned nil cmd")
			}
			msg := cmd()

			got, ok := msg.(sdk.PinToggleRequestMsg)
			if !ok {
				t.Fatalf("expected PinToggleRequestMsg, got %T", msg)
			}
			if !reflect.DeepEqual(got.Addresses, tt.addresses) {
				t.Errorf("request Addresses = %v, want %v", got.Addresses, tt.addresses)
			}
			if !reflect.DeepEqual(h.PinRequests, tt.addresses) {
				t.Errorf("PinRequests = %v, want %v", h.PinRequests, tt.addresses)
			}
		})
	}
}

func TestNewDeps_ClearPinsRecordsAndEmitsRequest(t *testing.T) {
	h := NewDeps(&MockService{})

	cmd := h.Deps.ClearPins()
	if cmd == nil {
		t.Fatal("ClearPins returned nil cmd")
	}
	msg := cmd()

	_, ok := msg.(sdk.PinClearRequestMsg)
	if !ok {
		t.Fatalf("expected PinClearRequestMsg, got %T", msg)
	}
	if h.ClearPinsCount != 1 {
		t.Errorf("ClearPinsCount = %d, want 1", h.ClearPinsCount)
	}
}
