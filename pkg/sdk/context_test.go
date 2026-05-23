package sdk

import (
	"reflect"
	"testing"
)

func TestContext_PlanOptions_ShouldIncludeAllExecFields(t *testing.T) {
	lock := true
	ctx := &Context{
		Pins:        []string{"aws_instance.web"},
		VarFiles:    []string{"prod.tfvars"},
		Vars:        map[string]string{"env": "prod"},
		ExtraArgs:   []string{"-no-color"},
		Parallelism: 5,
		Lock:        &lock,
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
	if opts.Lock == nil || *opts.Lock != true {
		t.Errorf("Lock = %v, want true", opts.Lock)
	}
	if opts.LockTimeout != "30s" {
		t.Errorf("LockTimeout = %q, want 30s", opts.LockTimeout)
	}
}

func TestContext_ApplyOptions_ShouldIncludeAllExecFields(t *testing.T) {
	lock := false
	ctx := &Context{
		Pins:        []string{"aws_instance.web"},
		VarFiles:    []string{"staging.tfvars"},
		Vars:        map[string]string{"region": "us-west-2"},
		ExtraArgs:   []string{"-compact-warnings"},
		Parallelism: 10,
		Lock:        &lock,
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
	if opts.Lock == nil || *opts.Lock != false {
		t.Errorf("Lock = %v, want false", opts.Lock)
	}
	if opts.LockTimeout != "1m" {
		t.Errorf("LockTimeout = %q, want 1m", opts.LockTimeout)
	}
}

func TestContext_WithPins_ShouldReturnFreshSnapshotWithoutMutatingOriginal(t *testing.T) {
	original := &Context{
		WorkingDir:  "/tmp/project",
		Workspace:   "default",
		VarFiles:    []string{"common.tfvars"},
		Vars:        map[string]string{"env": "prod"},
		Parallelism: 5,
		Pins:        []string{"aws_instance.original"},
	}

	next := original.WithPins([]string{"aws_instance.new"})

	if next == original {
		t.Fatal("WithPins returned the same pointer; should be a fresh snapshot")
	}
	if !reflect.DeepEqual(original.Pins, []string{"aws_instance.original"}) {
		t.Errorf("original mutated: Pins = %v", original.Pins)
	}
	if !reflect.DeepEqual(next.Pins, []string{"aws_instance.new"}) {
		t.Errorf("next.Pins = %v, want [aws_instance.new]", next.Pins)
	}
	if next.WorkingDir != original.WorkingDir {
		t.Errorf("next.WorkingDir = %q, want %q", next.WorkingDir, original.WorkingDir)
	}
	if next.Workspace != original.Workspace {
		t.Errorf("next.Workspace = %q, want %q", next.Workspace, original.Workspace)
	}
	if next.Parallelism != original.Parallelism {
		t.Errorf("next.Parallelism = %d, want %d", next.Parallelism, original.Parallelism)
	}
}

func TestContext_TogglePin_ShouldAddWhenAbsent(t *testing.T) {
	ctx := &Context{WorkingDir: "/p", Pins: []string{"a"}}
	next := ctx.TogglePin("b")
	if !reflect.DeepEqual(next.Pins, []string{"a", "b"}) {
		t.Errorf("TogglePin add: Pins = %v, want [a b]", next.Pins)
	}
	if !reflect.DeepEqual(ctx.Pins, []string{"a"}) {
		t.Errorf("original mutated: Pins = %v", ctx.Pins)
	}
}

func TestContext_TogglePin_ShouldRemoveWhenPresent(t *testing.T) {
	ctx := &Context{WorkingDir: "/p", Pins: []string{"a", "b", "c"}}
	next := ctx.TogglePin("b")
	if !reflect.DeepEqual(next.Pins, []string{"a", "c"}) {
		t.Errorf("TogglePin remove: Pins = %v, want [a c]", next.Pins)
	}
	if !reflect.DeepEqual(ctx.Pins, []string{"a", "b", "c"}) {
		t.Errorf("original mutated: Pins = %v", ctx.Pins)
	}
}

func TestContext_TogglePin_ShouldHandleEmptyPins(t *testing.T) {
	ctx := &Context{WorkingDir: "/p"}
	next := ctx.TogglePin("x")
	if !reflect.DeepEqual(next.Pins, []string{"x"}) {
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
	if opts.Parallelism != 0 || opts.PlanFile != "" || len(opts.VarFiles) != 0 {
		t.Errorf("nil receiver ApplyOptions = %+v, want zero ApplyOptions", opts)
	}
}

func TestContextChangedEvent_OnlyPinsChanged_ShouldReportTrueWhenOnlyPinsDiffer(t *testing.T) {
	prev := &Context{WorkingDir: "/p", Workspace: "ws", Pins: []string{"a"}}
	next := &Context{WorkingDir: "/p", Workspace: "ws", Pins: []string{"a", "b"}}

	ev := ContextChangedEvent{Prev: prev, Next: next}
	if !ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = false; want true when only Pins differ")
	}
}

func TestContextChangedEvent_OnlyPinsChanged_ShouldReportFalseWhenChdirChanges(t *testing.T) {
	prev := &Context{WorkingDir: "/old", Workspace: "ws"}
	next := &Context{WorkingDir: "/new", Workspace: "ws"}

	ev := ContextChangedEvent{Prev: prev, Next: next}
	if ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = true; want false when WorkingDir differs")
	}
}

func TestContextChangedEvent_OnlyPinsChanged_ShouldReportFalseWhenWorkspaceChanges(t *testing.T) {
	prev := &Context{WorkingDir: "/p", Workspace: "old"}
	next := &Context{WorkingDir: "/p", Workspace: "new"}

	ev := ContextChangedEvent{Prev: prev, Next: next}
	if ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = true; want false when Workspace differs")
	}
}

func TestContextChangedEvent_OnlyPinsChanged_ShouldReportFalseWhenPrevIsNil(t *testing.T) {
	next := &Context{WorkingDir: "/p", Workspace: "ws"}
	ev := ContextChangedEvent{Prev: nil, Next: next}
	if ev.OnlyPinsChanged() {
		t.Error("OnlyPinsChanged() = true with nil Prev; want false (initial context build)")
	}
}

func TestPinnedAddresses_WhenPinsExist_ShouldReturnThem(t *testing.T) {
	ctx := &Context{Pins: []string{"a", "b"}}
	getCtx := func() *Context { return ctx }
	got := PinnedAddresses(getCtx)
	if !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("PinnedAddresses = %v, want [a b]", got)
	}
}

func TestPinnedAddresses_WhenNoPins_ShouldReturnNil(t *testing.T) {
	ctx := &Context{}
	getCtx := func() *Context { return ctx }
	got := PinnedAddresses(getCtx)
	if len(got) != 0 {
		t.Errorf("PinnedAddresses(empty ctx) = %v, want empty", got)
	}
}
