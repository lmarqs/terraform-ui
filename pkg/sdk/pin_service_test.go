package sdk

import "testing"

func TestPinService_Toggle(t *testing.T) {
	ps := NewPinService()

	// Pin an address
	if got := ps.Toggle("aws_instance.web"); got != true {
		t.Errorf("Toggle(new) = %v, want true (pinned)", got)
	}

	// Pin another
	ps.Toggle("aws_s3_bucket.data")

	// Verify both pinned
	if !ps.IsPinned("aws_instance.web") {
		t.Error("expected aws_instance.web to be pinned")
	}
	if !ps.IsPinned("aws_s3_bucket.data") {
		t.Error("expected aws_s3_bucket.data to be pinned")
	}

	// Unpin first
	if got := ps.Toggle("aws_instance.web"); got != false {
		t.Errorf("Toggle(existing) = %v, want false (unpinned)", got)
	}
	if ps.IsPinned("aws_instance.web") {
		t.Error("expected aws_instance.web to be unpinned")
	}
	if !ps.IsPinned("aws_s3_bucket.data") {
		t.Error("expected aws_s3_bucket.data to still be pinned")
	}
}

func TestPinService_All(t *testing.T) {
	ps := NewPinService()

	if got := ps.All(); len(got) != 0 {
		t.Errorf("All() on empty = %v, want empty", got)
	}

	ps.Toggle("a")
	ps.Toggle("b")
	ps.Toggle("c")

	got := ps.All()
	if len(got) != 3 {
		t.Fatalf("All() len = %d, want 3", len(got))
	}
	want := map[string]bool{"a": true, "b": true, "c": true}
	for _, addr := range got {
		if !want[addr] {
			t.Errorf("unexpected address in All(): %q", addr)
		}
	}
}

func TestPinService_Set(t *testing.T) {
	ps := NewPinService()

	ps.Toggle("old_address")

	ps.Set([]string{"new_a", "new_b"})

	if ps.IsPinned("old_address") {
		t.Error("expected old_address to be cleared after Set")
	}
	if !ps.IsPinned("new_a") {
		t.Error("expected new_a to be pinned after Set")
	}
	if !ps.IsPinned("new_b") {
		t.Error("expected new_b to be pinned after Set")
	}
	if ps.Count() != 2 {
		t.Errorf("Count() = %d, want 2", ps.Count())
	}
}

func TestPinService_Count(t *testing.T) {
	ps := NewPinService()

	if ps.Count() != 0 {
		t.Errorf("Count() empty = %d, want 0", ps.Count())
	}

	ps.Toggle("a")
	ps.Toggle("b")
	if ps.Count() != 2 {
		t.Errorf("Count() = %d, want 2", ps.Count())
	}

	ps.Toggle("a") // unpin
	if ps.Count() != 1 {
		t.Errorf("Count() after unpin = %d, want 1", ps.Count())
	}
}

func TestPinService_SharedInstance(t *testing.T) {
	ps := NewPinService()
	ps.Toggle("shared_resource")

	// Same instance is shared — verify state is visible
	if !ps.IsPinned("shared_resource") {
		t.Error("expected shared_resource to be pinned")
	}
	if ps.Count() != 1 {
		t.Errorf("Count() = %d, want 1", ps.Count())
	}
}
