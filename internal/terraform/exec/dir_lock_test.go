package exec

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDirLock_WhenSameDir_ShouldSerialize(t *testing.T) {
	dl := NewDirLock()
	dir := "/dir/a"

	order := make([]string, 0, 2)
	var mu sync.Mutex
	aAcquired := make(chan struct{})
	bDone := make(chan struct{})

	dl.Acquire(dir)

	go func() {
		<-aAcquired
		dl.Acquire(dir)
		mu.Lock()
		order = append(order, "B")
		mu.Unlock()
		dl.Release(dir)
		close(bDone)
	}()

	close(aAcquired)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	order = append(order, "A")
	mu.Unlock()
	dl.Release(dir)

	<-bDone

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(order))
	}
	if order[0] != "A" || order[1] != "B" {
		t.Errorf("order = %v, want [A B]", order)
	}
}

func TestDirLock_WhenDifferentDirs_ShouldNotBlock(t *testing.T) {
	dl := NewDirLock()

	dl.Acquire("/dir/a")
	defer dl.Release("/dir/a")

	done := make(chan struct{})
	go func() {
		dl.Acquire("/dir/b")
		dl.Release("/dir/b")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("goroutine B blocked on a different dir")
	}
}

func TestDirLock_WhenManyGoroutines_ShouldNotRace(t *testing.T) {
	dl := NewDirLock()
	dir := "/dir/shared"
	const goroutines = 50

	var counter atomic.Int64
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			dl.Acquire(dir)
			counter.Add(1)
			dl.Release(dir)
		}()
	}

	wg.Wait()

	if got := counter.Load(); got != goroutines {
		t.Errorf("counter = %d, want %d", got, goroutines)
	}
}

func TestDirLock_WhenReleased_ShouldAllowNextAcquire(t *testing.T) {
	dl := NewDirLock()
	dir := "/dir/reuse"

	dl.Acquire(dir)
	dl.Release(dir)

	done := make(chan struct{})
	go func() {
		dl.Acquire(dir)
		dl.Release(dir)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("second acquire deadlocked after release")
	}
}

func TestExecService_WhenWithDir_ShouldShareDirLock(t *testing.T) {
	parent := NewExecService("/parent", "terraform", nil)
	child := parent.WithDir("/child").(*ExecService)

	if child.dirLock != parent.dirLock {
		t.Fatal("WithDir child does not share parent's DirLock")
	}
}
