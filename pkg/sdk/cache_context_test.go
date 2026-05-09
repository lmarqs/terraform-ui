package sdk

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheContext_SetAndGet(t *testing.T) {
	c := NewCacheContext()
	c.Set("key", "value", time.Minute)

	v, ok := c.Get("key")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if v != "value" {
		t.Fatalf("expected value, got %v", v)
	}
}

func TestCacheContext_GetNonExistent(t *testing.T) {
	c := NewCacheContext()
	_, ok := c.Get("missing")
	if ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestCacheContext_TTLExpiration(t *testing.T) {
	c := NewCacheContext()
	c.Set("key", "value", 10*time.Millisecond)

	// Should be available immediately
	v, ok := c.Get("key")
	if !ok {
		t.Fatal("expected key to be found before expiration")
	}
	if v != "value" {
		t.Fatalf("expected value, got %v", v)
	}

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	_, ok = c.Get("key")
	if ok {
		t.Fatal("expected expired key to return false")
	}
}

func TestCacheContext_ZeroTTLNoExpiration(t *testing.T) {
	c := NewCacheContext()
	c.Set("key", "forever", 0)

	// Should be available immediately
	v, ok := c.Get("key")
	if !ok {
		t.Fatal("expected key with zero TTL to be found")
	}
	if v != "forever" {
		t.Fatalf("expected forever, got %v", v)
	}

	// Wait a bit and verify it's still there
	time.Sleep(15 * time.Millisecond)
	v, ok = c.Get("key")
	if !ok {
		t.Fatal("expected key with zero TTL to never expire")
	}
	if v != "forever" {
		t.Fatalf("expected forever, got %v", v)
	}
}

func TestCacheGetTyped_CorrectType(t *testing.T) {
	c := NewCacheContext()
	c.Set("count", 42, time.Minute)

	v, ok := CacheGetTyped[int](c, "count")
	if !ok {
		t.Fatal("expected typed get to succeed")
	}
	if v != 42 {
		t.Fatalf("expected 42, got %d", v)
	}
}

func TestCacheGetTyped_WrongType(t *testing.T) {
	c := NewCacheContext()
	c.Set("count", "not-an-int", time.Minute)

	v, ok := CacheGetTyped[int](c, "count")
	if ok {
		t.Fatal("expected typed get to fail for wrong type")
	}
	if v != 0 {
		t.Fatalf("expected zero value, got %d", v)
	}
}

func TestCacheGetTyped_MissingKey(t *testing.T) {
	c := NewCacheContext()

	v, ok := CacheGetTyped[string](c, "missing")
	if ok {
		t.Fatal("expected typed get to fail for missing key")
	}
	if v != "" {
		t.Fatalf("expected zero value, got %q", v)
	}
}

func TestCacheGetTyped_Expired(t *testing.T) {
	c := NewCacheContext()
	c.Set("key", "value", 10*time.Millisecond)

	time.Sleep(20 * time.Millisecond)

	_, ok := CacheGetTyped[string](c, "key")
	if ok {
		t.Fatal("expected typed get to fail for expired key")
	}
}

func TestCacheContext_Invalidate(t *testing.T) {
	c := NewCacheContext()
	c.Set("key1", "value1", time.Minute)
	c.Set("key2", "value2", time.Minute)

	c.Invalidate("key1")

	_, ok := c.Get("key1")
	if ok {
		t.Fatal("expected invalidated key to be gone")
	}

	v, ok := c.Get("key2")
	if !ok {
		t.Fatal("expected key2 to still exist")
	}
	if v != "value2" {
		t.Fatalf("expected value2, got %v", v)
	}
}

func TestCacheContext_InvalidateNonExistent(t *testing.T) {
	c := NewCacheContext()
	// Should not panic
	c.Invalidate("nonexistent")
}

func TestCacheContext_InvalidatePrefix(t *testing.T) {
	c := NewCacheContext()
	c.Set("state:list", "data1", time.Minute)
	c.Set("state:show:addr1", "data2", time.Minute)
	c.Set("plan:result", "data3", time.Minute)

	c.InvalidatePrefix("state:")

	_, ok := c.Get("state:list")
	if ok {
		t.Fatal("expected state:list to be invalidated")
	}
	_, ok = c.Get("state:show:addr1")
	if ok {
		t.Fatal("expected state:show:addr1 to be invalidated")
	}

	v, ok := c.Get("plan:result")
	if !ok {
		t.Fatal("expected plan:result to still exist")
	}
	if v != "data3" {
		t.Fatalf("expected data3, got %v", v)
	}
}

func TestCacheContext_InvalidatePrefixNoMatch(t *testing.T) {
	c := NewCacheContext()
	c.Set("key1", "value1", time.Minute)
	c.Set("key2", "value2", time.Minute)

	c.InvalidatePrefix("nonexistent:")

	_, ok1 := c.Get("key1")
	_, ok2 := c.Get("key2")
	if !ok1 || !ok2 {
		t.Fatal("expected all keys to remain when prefix doesn't match")
	}
}

func TestCacheContext_Clear(t *testing.T) {
	c := NewCacheContext()
	c.Set("key1", "value1", time.Minute)
	c.Set("key2", "value2", time.Minute)
	c.Set("key3", "value3", time.Minute)

	c.Clear()

	_, ok1 := c.Get("key1")
	_, ok2 := c.Get("key2")
	_, ok3 := c.Get("key3")
	if ok1 || ok2 || ok3 {
		t.Fatal("expected all keys to be removed after Clear")
	}
}

func TestCacheContext_ClearEmpty(t *testing.T) {
	c := NewCacheContext()
	// Should not panic
	c.Clear()
}

func TestCacheContext_Cleanup(t *testing.T) {
	c := NewCacheContext()
	c.Set("expired", "old", 10*time.Millisecond)
	c.Set("alive", "fresh", time.Minute)
	c.Set("no-expiry", "forever", 0)

	time.Sleep(20 * time.Millisecond)
	c.Cleanup()

	_, ok := c.Get("expired")
	if ok {
		t.Fatal("expected expired key to be removed by Cleanup")
	}

	v, ok := c.Get("alive")
	if !ok {
		t.Fatal("expected alive key to remain after Cleanup")
	}
	if v != "fresh" {
		t.Fatalf("expected fresh, got %v", v)
	}

	v, ok = c.Get("no-expiry")
	if !ok {
		t.Fatal("expected no-expiry key to remain after Cleanup")
	}
	if v != "forever" {
		t.Fatalf("expected forever, got %v", v)
	}
}

func TestCacheContext_ConcurrentAccess(t *testing.T) {
	c := NewCacheContext()
	var wg sync.WaitGroup
	const goroutines = 100

	// Concurrent writers
	wg.Add(goroutines)
	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			c.Set(fmt.Sprintf("key%d", n), n, time.Minute)
		}(i)
	}

	// Concurrent readers
	wg.Add(goroutines)
	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			c.Get(fmt.Sprintf("key%d", n))
		}(i)
	}

	// Concurrent mixed operations
	wg.Add(goroutines)
	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			switch n % 4 {
			case 0:
				c.Set(fmt.Sprintf("shared%d", n%10), n, time.Minute)
			case 1:
				c.Get(fmt.Sprintf("shared%d", n%10))
			case 2:
				c.Invalidate(fmt.Sprintf("shared%d", n%10))
			case 3:
				c.InvalidatePrefix("shared")
			}
		}(i)
	}

	wg.Wait()

	// Should not have panicked — if we get here, concurrency is safe
}

func TestCacheContext_OverwriteValue(t *testing.T) {
	c := NewCacheContext()
	c.Set("key", "first", time.Minute)
	c.Set("key", "second", time.Minute)

	v, ok := c.Get("key")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if v != "second" {
		t.Fatalf("expected second, got %v", v)
	}
}

func TestCacheContext_OverwriteExtendsTTL(t *testing.T) {
	c := NewCacheContext()
	c.Set("key", "value", 10*time.Millisecond)

	// Overwrite with longer TTL before expiration
	c.Set("key", "updated", time.Minute)

	time.Sleep(20 * time.Millisecond)

	v, ok := c.Get("key")
	if !ok {
		t.Fatal("expected overwritten key with new TTL to still be alive")
	}
	if v != "updated" {
		t.Fatalf("expected updated, got %v", v)
	}
}
