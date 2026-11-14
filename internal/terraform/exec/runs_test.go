package exec

import (
	"context"
	"runtime"
	"testing"
	"time"
)

const runsTimeout = 2 * time.Second

func isCancelled(t *testing.T, ctx context.Context) bool {
	t.Helper()
	select {
	case <-ctx.Done():
		return true
	case <-time.After(runsTimeout):
		return false
	}
}

func isLive(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(20 * time.Millisecond):
		return true
	}
}

func TestRuns_WhenInterrupted_ShouldCancelContextsTrackedBeforeAndAfter(t *testing.T) {
	tests := []struct {
		name  string
		track func(r *Runs) context.Context
	}{
		{
			name: "tracked before Interrupt",
			track: func(r *Runs) context.Context {
				ctx, done := r.Track(context.Background())
				t.Cleanup(done)
				r.Interrupt()
				return ctx
			},
		},
		{
			name: "tracked after Interrupt",
			track: func(r *Runs) context.Context {
				r.Interrupt()
				ctx, done := r.Track(context.Background())
				t.Cleanup(done)
				return ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.track(NewRuns())
			if !isCancelled(t, ctx) {
				t.Fatal("tracked context was not cancelled by Interrupt")
			}
		})
	}
}

func TestRuns_WhenInterruptedTwice_ShouldNotPanic(t *testing.T) {
	r := NewRuns()
	ctx, done := r.Track(context.Background())
	defer done()

	r.Interrupt()
	r.Interrupt()

	if !isCancelled(t, ctx) {
		t.Fatal("tracked context was not cancelled by Interrupt")
	}
}

func TestRuns_WhenCallerContextCancelled_ShouldCancelOnlyThatContext(t *testing.T) {
	r := NewRuns()
	callerCtx, cancelCaller := context.WithCancel(context.Background())
	cancelled, doneA := r.Track(callerCtx)
	defer doneA()
	other, doneB := r.Track(context.Background())
	defer doneB()

	cancelCaller()

	if !isCancelled(t, cancelled) {
		t.Fatal("tracked context was not cancelled with its caller's context")
	}
	if !isLive(other) {
		t.Error("cancelling one caller's context cancelled another tracked context")
	}
}

func TestRuns_WhenDoneOutstanding_ShouldBlockWaitUntilAllDone(t *testing.T) {
	r := NewRuns()
	_, doneA := r.Track(context.Background())
	_, doneB := r.Track(context.Background())

	waited := make(chan struct{})
	go func() {
		r.Wait()
		close(waited)
	}()

	doneA()
	select {
	case <-waited:
		t.Fatal("Wait returned while a tracked call was still outstanding")
	case <-time.After(50 * time.Millisecond):
	}

	doneB()
	select {
	case <-waited:
	case <-time.After(runsTimeout):
		t.Fatal("Wait did not return after every done was called")
	}
}

func TestRuns_Running_ShouldReflectOutstandingCalls(t *testing.T) {
	r := NewRuns()
	if r.Running() {
		t.Fatal("Running() = true on a fresh tracker")
	}

	_, doneA := r.Track(context.Background())
	_, doneB := r.Track(context.Background())
	if !r.Running() {
		t.Fatal("Running() = false with two calls outstanding")
	}

	doneA()
	if !r.Running() {
		t.Fatal("Running() = false with one call outstanding")
	}

	doneB()
	if r.Running() {
		t.Fatal("Running() = true after every done was called")
	}
}

func TestExecService_WhenWithDir_ShouldShareRunsTracker(t *testing.T) {
	svc := NewExecService("/dir/a", "terraform", nil)
	scoped, ok := svc.WithDir("/dir/b").(*ExecService)
	if !ok {
		t.Fatal("WithDir did not return an *ExecService")
	}

	if svc.Runs() == nil {
		t.Fatal("Runs() = nil")
	}
	if scoped.Runs() != svc.Runs() {
		t.Error("WithDir copy does not share the original's Runs tracker")
	}
}

func TestTerraformHasOwnProcessGroup_WhenLinux_ShouldBeTrue(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("terraform-exec sets Setpgid only on linux")
	}
	if !TerraformHasOwnProcessGroup {
		t.Error("TerraformHasOwnProcessGroup = false on linux")
	}
}

func TestRuns_WhenDoneCalledAfterInterruptOrTwice_ShouldCountEachCallOnce(t *testing.T) {
	tests := []struct {
		name        string
		act         func(r *Runs) (waitReturns bool)
		wantRunning bool
	}{
		{
			name: "done after Interrupt lets Wait return",
			act: func(r *Runs) bool {
				_, done := r.Track(context.Background())
				r.Interrupt()
				done()
				return waitReturns(r)
			},
			wantRunning: false,
		},
		{
			name: "done twice decrements once, so a second outstanding call keeps Wait blocked",
			act: func(r *Runs) bool {
				_, doneA := r.Track(context.Background())
				_, doneB := r.Track(context.Background())
				t.Cleanup(doneB)
				doneA()
				doneA()
				return waitReturns(r)
			},
			wantRunning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRuns()
			returned := tt.act(r)
			if returned == tt.wantRunning {
				t.Errorf("Wait returned = %v, want %v", returned, !tt.wantRunning)
			}
			if r.Running() != tt.wantRunning {
				t.Errorf("Running() = %v, want %v", r.Running(), tt.wantRunning)
			}
		})
	}
}

func waitReturns(r *Runs) bool {
	returned := make(chan struct{})
	go func() {
		r.Wait()
		close(returned)
	}()
	select {
	case <-returned:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}
