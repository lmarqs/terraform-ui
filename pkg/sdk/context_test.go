package sdk

import (
	"reflect"
	"testing"
)

func TestContext_PlanOptions_ShouldIncludeAllExecFields(t *testing.T) {
	ctx := &Context{
		Pins:        []string{"aws_instance.web"},
		VarFiles:    []string{"prod.tfvars"},
		Vars:        map[string]string{"env": "prod"},
		ExtraArgs:   []string{"-no-color"},
		Parallelism: 5,
		Lock:        LockEnabled,
		LockTimeout: "30s",
	}

	opts := ctx.PlanOptions()

	if !reflect.DeepEqual(opts.Targets, []string{"aws_instance.web"}) {
		t.Errorf("Targets = %v, want [aws_instance.web]", opts.Targets)
	}
	if !reflect.DeepEqual(opts.VarFiles, []string{"prod.tfvars"}) {
		t.Errorf("VarFiles = %v, want [prod.tfvars]", opts.VarFiles)
	}
	if opts.Vars["env"] != "prod" {
		t.Errorf("Vars[env] = %q, want prod", opts.Vars["env"])
	}
	if !reflect.DeepEqual(opts.ExtraArgs, []string{"-no-color"}) {
		t.Errorf("ExtraArgs = %v, want [-no-color]", opts.ExtraArgs)
	}
	if opts.Parallelism != 5 {
		t.Errorf("Parallelism = %d, want 5", opts.Parallelism)
	}
	if opts.Lock != LockEnabled {
		t.Errorf("Lock = %v, want LockEnabled", opts.Lock)
	}
	if opts.LockTimeout != "30s" {
		t.Errorf("LockTimeout = %q, want 30s", opts.LockTimeout)
	}
}

func TestContext_ApplyOptions_ShouldIncludeAllExecFields(t *testing.T) {
	ctx := &Context{
		Pins:        []string{"aws_instance.web"},
		VarFiles:    []string{"staging.tfvars"},
		Vars:        map[string]string{"region": "us-west-2"},
		ExtraArgs:   []string{"-compact-warnings"},
		Parallelism: 10,
		Lock:        LockDisabled,
		LockTimeout: "1m",
	}

	opts := ctx.ApplyOptions()

	if !reflect.DeepEqual(opts.VarFiles, []string{"staging.tfvars"}) {
		t.Errorf("VarFiles = %v, want [staging.tfvars]", opts.VarFiles)
	}
	if opts.Vars["region"] != "us-west-2" {
		t.Errorf("Vars[region] = %q, want us-west-2", opts.Vars["region"])
	}
	if !reflect.DeepEqual(opts.ExtraArgs, []string{"-compact-warnings"}) {
		t.Errorf("ExtraArgs = %v, want [-compact-warnings]", opts.ExtraArgs)
	}
	if opts.Parallelism != 10 {
		t.Errorf("Parallelism = %d, want 10", opts.Parallelism)
	}
	if opts.Lock != LockDisabled {
		t.Errorf("Lock = %v, want LockDisabled", opts.Lock)
	}
	if opts.LockTimeout != "1m" {
		t.Errorf("LockTimeout = %q, want 1m", opts.LockTimeout)
	}
}

func TestContext_WithPins_ShouldReturnFreshSnapshotWithoutMutatingOriginal(t *testing.T) {
	original := &Context{
		WorkingDir:  "/tmp/project",
		Workspace:   WorkspaceDefault,
		VarFiles:    []string{"common.tfvars"},
		Vars:        map[string]string{"env": "prod"},
		Parallelism: 5,
		Pins:        Pins{"aws_instance.original"},
	}

	next := original.WithPins(Pins{"aws_instance.new"})

	if next == original {
		t.Fatal("WithPins returned the same pointer; should be a fresh snapshot")
	}
	if !reflect.DeepEqual(original.Pins, Pins{"aws_instance.original"}) {
		t.Errorf("original mutated: Pins = %v", original.Pins)
	}
	if !reflect.DeepEqual(next.Pins, Pins{"aws_instance.new"}) {
		t.Errorf("next.Pins = %v, want [aws_instance.new]", next.Pins)
	}
	if next.WorkingDir != original.WorkingDir {
		t.Errorf("next.WorkingDir = %q, want %q", next.WorkingDir, original.WorkingDir)
	}
	if next.Workspace != original.Workspace {
		t.Errorf("next.Workspace = %v, want %v", next.Workspace, original.Workspace)
	}
	if next.Parallelism != original.Parallelism {
		t.Errorf("next.Parallelism = %d, want %d", next.Parallelism, original.Parallelism)
	}
}

func TestContext_TogglePin_ShouldAddWhenAbsent(t *testing.T) {
	ctx := &Context{WorkingDir: "/p", Pins: Pins{"a"}}
	next := ctx.TogglePin("b")
	if !reflect.DeepEqual(next.Pins, Pins{"a", "b"}) {
		t.Errorf("TogglePin add: Pins = %v, want [a b]", next.Pins)
	}
	if !reflect.DeepEqual(ctx.Pins, Pins{"a"}) {
		t.Errorf("original mutated: Pins = %v", ctx.Pins)
	}
}

func TestContext_TogglePin_ShouldRemoveWhenPresent(t *testing.T) {
	ctx := &Context{WorkingDir: "/p", Pins: Pins{"a", "b", "c"}}
	next := ctx.TogglePin("b")
	if !reflect.DeepEqual(next.Pins, Pins{"a", "c"}) {
		t.Errorf("TogglePin remove: Pins = %v, want [a c]", next.Pins)
	}
	if !reflect.DeepEqual(ctx.Pins, Pins{"a", "b", "c"}) {
		t.Errorf("original mutated: Pins = %v", ctx.Pins)
	}
}

func TestContext_TogglePin_ShouldHandleEmptyPins(t *testing.T) {
	ctx := &Context{WorkingDir: "/p"}
	next := ctx.TogglePin("x")
	if !reflect.DeepEqual(next.Pins, Pins{"x"}) {
		t.Errorf("TogglePin on empty: Pins = %v, want [x]", next.Pins)
	}
}

func TestContext_PlanOptions_GivenEmptyContext_ShouldReturnEmptyOptions(t *testing.T) {
	ctx := &Context{}
	opts := ctx.PlanOptions()
	if len(opts.Targets) != 0 || len(opts.VarFiles) != 0 || len(opts.Vars) != 0 || opts.Parallelism != 0 {
		t.Errorf("empty Context PlanOptions = %+v, want zero values", opts)
	}
}

func TestContext_PlanOptions_GivenNilReceiver_ShouldReturnZeroValue(t *testing.T) {
	var ctx *Context
	opts := ctx.PlanOptions()
	if opts.Parallelism != 0 || len(opts.Targets) != 0 {
		t.Errorf("nil receiver PlanOptions = %+v, want zero PlanOptions", opts)
	}
}

func TestContext_ApplyOptions_GivenNilReceiver_ShouldReturnZeroValue(t *testing.T) {
	var ctx *Context
	opts := ctx.ApplyOptions()
	if opts.Parallelism != 0 || len(opts.VarFiles) != 0 {
		t.Errorf("nil receiver ApplyOptions = %+v, want zero ApplyOptions", opts)
	}
}

func TestContextChangedEvent_OnlyPinsChanged_ShouldReportTrueWhenReasonIsPins(t *testing.T) {
	prev := &Context{WorkingDir: "/p", Workspace: NewWorkspace("ws"), Pins: []string{"a"}}
	next := &Context{WorkingDir: "/p", Workspace: NewWorkspace("ws"), Pins: []string{"a", "b"}}

	ev := ContextChangedEvent{Prev: prev, Next: next, Reason: ContextPinsChanged}
	if !ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = false; want true when Reason is ContextPinsChanged")
	}
}

func TestContextChangedEvent_OnlyPinsChanged_ShouldReportFalseWhenReasonIsSwitched(t *testing.T) {
	prev := &Context{WorkingDir: "/old", Workspace: NewWorkspace("ws")}
	next := &Context{WorkingDir: "/new", Workspace: NewWorkspace("ws")}

	ev := ContextChangedEvent{Prev: prev, Next: next, Reason: ContextSwitched}
	if ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = true; want false for a chdir/workspace switch")
	}
}

// Regression for #37: a same-chdir re-selection goes through the switch path
// (which also nulls pins), so Prev and Next share WorkingDir+Workspace while
// pins differ. The old diff-based check misread this as a pins-only change and
// skipped the full reset. Provenance (Reason=ContextSwitched) fixes it.
func TestContextChangedEvent_OnlyPinsChanged_ShouldReportFalseOnSameChdirReselection(t *testing.T) {
	prev := &Context{WorkingDir: "/p", Workspace: NewWorkspace("ws"), Pins: []string{"a"}}
	next := &Context{WorkingDir: "/p", Workspace: NewWorkspace("ws")} // pins dropped by rebuild

	ev := ContextChangedEvent{Prev: prev, Next: next, Reason: ContextSwitched}
	if ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = true; want false — a same-chdir re-selection is a switch, not a pin change")
	}
}

func TestContextChangedEvent_OnlyPinsChanged_ShouldReportFalseForZeroReason(t *testing.T) {
	// The zero value defaults to ContextSwitched: any event built without an
	// explicit reason is treated as a full reset.
	ev := ContextChangedEvent{Next: &Context{WorkingDir: "/p"}}
	if ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = true for zero-value Reason; want false (defaults to switch)")
	}
}
