package sdk

import (
	"sync"
	"testing"
)

func TestSession_SetGet(t *testing.T) {
	s := NewSession()

	s.Set("key1", "value1")
	s.Set("key2", 42)

	v, ok := s.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if v != "value1" {
		t.Fatalf("expected value1, got %v", v)
	}

	v, ok = s.Get("key2")
	if !ok {
		t.Fatal("expected key2 to exist")
	}
	if v != 42 {
		t.Fatalf("expected 42, got %v", v)
	}
}

func TestSession_GetMissing(t *testing.T) {
	s := NewSession()

	_, ok := s.Get("nonexistent")
	if ok {
		t.Fatal("expected nonexistent key to return false")
	}
}

func TestSession_Overwrite(t *testing.T) {
	s := NewSession()

	s.Set("key", "first")
	s.Set("key", "second")

	v, ok := s.Get("key")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if v != "second" {
		t.Fatalf("expected second, got %v", v)
	}
}

func TestGetTyped_CorrectType(t *testing.T) {
	s := NewSession()
	s.Set("count", 10)

	v, ok := GetTyped[int](s, "count")
	if !ok {
		t.Fatal("expected typed get to succeed")
	}
	if v != 10 {
		t.Fatalf("expected 10, got %d", v)
	}
}

func TestGetTyped_WrongType(t *testing.T) {
	s := NewSession()
	s.Set("count", "not-an-int")

	v, ok := GetTyped[int](s, "count")
	if ok {
		t.Fatal("expected typed get to fail for wrong type")
	}
	if v != 0 {
		t.Fatalf("expected zero value, got %d", v)
	}
}

func TestGetTyped_MissingKey(t *testing.T) {
	s := NewSession()

	v, ok := GetTyped[string](s, "missing")
	if ok {
		t.Fatal("expected typed get to fail for missing key")
	}
	if v != "" {
		t.Fatalf("expected zero value, got %q", v)
	}
}

func TestGetTyped_PointerType(t *testing.T) {
	s := NewSession()
	summary := &PlanSummary{ToCreate: 3, ToDelete: 1}
	s.Set(SessionKeyPlanSummary, summary)

	v, ok := GetTyped[*PlanSummary](s, SessionKeyPlanSummary)
	if !ok {
		t.Fatal("expected typed get to succeed for pointer type")
	}
	if v.ToCreate != 3 {
		t.Fatalf("expected ToCreate=3, got %d", v.ToCreate)
	}
	if v.ToDelete != 1 {
		t.Fatalf("expected ToDelete=1, got %d", v.ToDelete)
	}
}

func TestSession_ConcurrentAccess(t *testing.T) {
	s := NewSession()
	var wg sync.WaitGroup
	const goroutines = 100

	// Concurrent writers
	wg.Add(goroutines)
	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			s.Set("key", n)
		}(i)
	}

	// Concurrent readers
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			s.Get("key")
		}()
	}

	wg.Wait()

	// Verify final state is valid (some int was written)
	v, ok := s.Get("key")
	if !ok {
		t.Fatal("expected key to exist after concurrent writes")
	}
	n, ok := v.(int)
	if !ok {
		t.Fatal("expected value to be int")
	}
	if n < 0 || n >= goroutines {
		t.Fatalf("unexpected value %d", n)
	}
}
