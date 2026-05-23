package sdktest

import (
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

func TestNewDeps_PinRecordsAddressAndEmitsRequest(t *testing.T) {
	h := NewDeps(&MockService{})

	cmd := h.Deps.Pin("aws_s3_bucket.x")
	if cmd == nil {
		t.Fatal("Pin returned nil cmd")
	}
	msg := cmd()

	got, ok := msg.(sdk.PinToggleRequestMsg)
	if !ok {
		t.Fatalf("expected PinToggleRequestMsg, got %T", msg)
	}
	if got.Address != "aws_s3_bucket.x" {
		t.Errorf("unexpected request: %+v", got)
	}
	if len(h.PinRequests) != 1 || h.PinRequests[0] != "aws_s3_bucket.x" {
		t.Errorf("PinRequests = %v", h.PinRequests)
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
