package exec

import (
	"context"
	"sync"
)

// Runs tracks the terraform calls a session started, so shutdown can
// interrupt every one of them and wait for each to return.
type Runs struct {
	mu          sync.Mutex
	idle        *sync.Cond
	outstanding int
	nextID      int
	cancels     map[int]context.CancelFunc
	interrupted bool
}

// NewRuns creates an empty tracker.
func NewRuns() *Runs {
	r := &Runs{cancels: make(map[int]context.CancelFunc)}
	r.idle = sync.NewCond(&r.mu)
	return r
}

// Track registers a terraform call. The returned context is cancelled when ctx
// is, or when Interrupt runs (immediately, if it already has). The caller
// must call done when the call returns.
func (r *Runs) Track(ctx context.Context) (context.Context, func()) {
	tracked, cancel := context.WithCancel(ctx)

	r.mu.Lock()
	id := r.nextID
	r.nextID++
	r.outstanding++
	r.cancels[id] = cancel
	if r.interrupted {
		cancel()
	}
	r.mu.Unlock()

	var once sync.Once
	return tracked, func() {
		once.Do(func() {
			cancel()
			r.mu.Lock()
			delete(r.cancels, id)
			r.outstanding--
			if r.outstanding == 0 {
				r.idle.Broadcast()
			}
			r.mu.Unlock()
		})
	}
}

// Interrupt cancels every tracked context, including any tracked later.
func (r *Runs) Interrupt() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.interrupted = true
	for _, cancel := range r.cancels {
		cancel()
	}
}

// Wait blocks until every tracked call has called done.
func (r *Runs) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for r.outstanding > 0 {
		r.idle.Wait()
	}
}

// Running reports whether a tracked call has not yet called done.
func (r *Runs) Running() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.outstanding > 0
}
