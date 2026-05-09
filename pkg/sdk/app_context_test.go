package sdk

import (
	"log/slog"
	"os"
	"sync"
	"testing"
)

func TestNewAppContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx := NewAppContext("/tmp/project", "default", nil, logger)

	if ctx.Project.Dir != "/tmp/project" {
		t.Fatalf("expected dir /tmp/project, got %s", ctx.Project.Dir)
	}
	if ctx.Terraform == nil {
		t.Fatal("expected Terraform context to be initialized")
	}
	if ctx.Terraform.WorkingDir != "/tmp/project" {
		t.Fatalf("expected WorkingDir /tmp/project, got %s", ctx.Terraform.WorkingDir)
	}
	if ctx.Terraform.Workspace != "default" {
		t.Fatalf("expected workspace default, got %s", ctx.Terraform.Workspace)
	}
	if ctx.Terraform.Service != nil {
		t.Fatal("expected Service nil when passed nil")
	}
	if ctx.UI == nil {
		t.Fatal("expected UI context to be initialized")
	}
	if ctx.Config == nil {
		t.Fatal("expected Config context to be initialized")
	}
	if ctx.Cache == nil {
		t.Fatal("expected Cache context to be initialized")
	}
	if ctx.Logger != logger {
		t.Fatal("expected Logger to be the passed logger")
	}
}

func TestTerraformContext_Pin(t *testing.T) {
	tc := &TerraformContext{}

	tc.Pin("aws_instance.web")
	if !tc.IsPinned("aws_instance.web") {
		t.Fatal("expected aws_instance.web to be pinned")
	}
	if tc.PinnedCount() != 1 {
		t.Fatalf("expected 1 pinned, got %d", tc.PinnedCount())
	}
}

func TestTerraformContext_PinIdempotent(t *testing.T) {
	tc := &TerraformContext{}

	tc.Pin("aws_instance.web")
	tc.Pin("aws_instance.web")
	tc.Pin("aws_instance.web")

	if tc.PinnedCount() != 1 {
		t.Fatalf("expected 1 pinned (idempotent), got %d", tc.PinnedCount())
	}
}

func TestTerraformContext_Unpin(t *testing.T) {
	tc := &TerraformContext{}

	tc.Pin("aws_instance.web")
	tc.Pin("aws_s3_bucket.data")

	tc.Unpin("aws_instance.web")
	if tc.IsPinned("aws_instance.web") {
		t.Fatal("expected aws_instance.web to be unpinned")
	}
	if !tc.IsPinned("aws_s3_bucket.data") {
		t.Fatal("expected aws_s3_bucket.data to still be pinned")
	}
	if tc.PinnedCount() != 1 {
		t.Fatalf("expected 1 pinned, got %d", tc.PinnedCount())
	}
}

func TestTerraformContext_UnpinNonExistent(t *testing.T) {
	tc := &TerraformContext{}
	tc.Pin("aws_instance.web")

	// Unpin something not pinned — should be no-op
	tc.Unpin("aws_lambda_function.handler")

	if tc.PinnedCount() != 1 {
		t.Fatalf("expected 1 pinned after no-op unpin, got %d", tc.PinnedCount())
	}
	if !tc.IsPinned("aws_instance.web") {
		t.Fatal("expected aws_instance.web to still be pinned")
	}
}

func TestTerraformContext_IsPinned_NotPinned(t *testing.T) {
	tc := &TerraformContext{}
	if tc.IsPinned("anything") {
		t.Fatal("expected nothing to be pinned initially")
	}
}

func TestTerraformContext_ClearPins(t *testing.T) {
	tc := &TerraformContext{}
	tc.Pin("a")
	tc.Pin("b")
	tc.Pin("c")

	tc.ClearPins()
	if tc.PinnedCount() != 0 {
		t.Fatalf("expected 0 pinned after clear, got %d", tc.PinnedCount())
	}
	if tc.IsPinned("a") {
		t.Fatal("expected a to not be pinned after clear")
	}
}

func TestTerraformContext_GetPinned(t *testing.T) {
	tc := &TerraformContext{}
	tc.Pin("alpha")
	tc.Pin("beta")

	pinned := tc.GetPinned()
	if len(pinned) != 2 {
		t.Fatalf("expected 2 pinned, got %d", len(pinned))
	}

	// Verify it's a copy (mutating returned slice doesn't affect internal state)
	pinned[0] = "modified"
	original := tc.GetPinned()
	if original[0] == "modified" {
		t.Fatal("expected GetPinned to return a copy, not a reference")
	}
}

func TestTerraformContext_SetState(t *testing.T) {
	tc := &TerraformContext{}

	resources := []Resource{
		{Address: "aws_instance.web", Type: "aws_instance", Name: "web"},
		{Address: "aws_s3_bucket.data", Type: "aws_s3_bucket", Name: "data"},
	}
	tc.SetState(resources)

	if tc.State == nil {
		t.Fatal("expected State to be set")
	}
	if len(tc.State.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(tc.State.Resources))
	}
	if tc.State.LastRefreshed.IsZero() {
		t.Fatal("expected LastRefreshed to be set")
	}
	if tc.State.Loading {
		t.Fatal("expected Loading to be false")
	}
	if tc.State.Error != nil {
		t.Fatal("expected Error to be nil")
	}
}

func TestTerraformContext_InvalidateState(t *testing.T) {
	tc := &TerraformContext{}
	tc.SetState([]Resource{{Address: "test"}})

	tc.InvalidateState()
	if tc.State != nil {
		t.Fatal("expected State to be nil after invalidation")
	}
}

func TestTerraformContext_SetPlan(t *testing.T) {
	tc := &TerraformContext{}

	summary := &PlanSummary{
		ToCreate: 3,
		ToUpdate: 1,
	}
	tc.SetPlan(summary, "/tmp/plan.tfplan")

	if tc.Plan == nil {
		t.Fatal("expected Plan to be set")
	}
	if tc.Plan.Summary != summary {
		t.Fatal("expected Plan.Summary to match")
	}
	if tc.Plan.FilePath != "/tmp/plan.tfplan" {
		t.Fatalf("expected filePath /tmp/plan.tfplan, got %s", tc.Plan.FilePath)
	}
	if tc.Plan.LastRefreshed.IsZero() {
		t.Fatal("expected LastRefreshed to be set")
	}
}

func TestTerraformContext_InvalidatePlan(t *testing.T) {
	tc := &TerraformContext{}
	tc.SetPlan(&PlanSummary{}, "/tmp/plan.tfplan")

	tc.InvalidatePlan()
	if tc.Plan != nil {
		t.Fatal("expected Plan to be nil after invalidation")
	}
}

func TestUIContext_SetGetSize(t *testing.T) {
	ui := &UIContext{}

	ui.SetSize(120, 40)
	w, h := ui.GetSize()
	if w != 120 {
		t.Fatalf("expected width 120, got %d", w)
	}
	if h != 40 {
		t.Fatalf("expected height 40, got %d", h)
	}
}

func TestUIContext_SetGetActivePlugin(t *testing.T) {
	ui := &UIContext{}

	if ui.GetActivePlugin() != "" {
		t.Fatal("expected empty active plugin initially")
	}

	ui.SetActivePlugin("state")
	if ui.GetActivePlugin() != "state" {
		t.Fatalf("expected state, got %s", ui.GetActivePlugin())
	}

	ui.SetActivePlugin("plan")
	if ui.GetActivePlugin() != "plan" {
		t.Fatalf("expected plan, got %s", ui.GetActivePlugin())
	}
}

func TestUIContext_SetGetInputMode(t *testing.T) {
	ui := &UIContext{}

	if ui.GetInputMode() != InputModeNormal {
		t.Fatalf("expected InputModeNormal initially, got %d", ui.GetInputMode())
	}

	ui.SetInputMode(InputModeCommand)
	if ui.GetInputMode() != InputModeCommand {
		t.Fatalf("expected InputModeCommand, got %d", ui.GetInputMode())
	}

	ui.SetInputMode(InputModeFilter)
	if ui.GetInputMode() != InputModeFilter {
		t.Fatalf("expected InputModeFilter, got %d", ui.GetInputMode())
	}

	ui.SetInputMode(InputModePrompt)
	if ui.GetInputMode() != InputModePrompt {
		t.Fatalf("expected InputModePrompt, got %d", ui.GetInputMode())
	}

	ui.SetInputMode(InputModeREPL)
	if ui.GetInputMode() != InputModeREPL {
		t.Fatalf("expected InputModeREPL, got %d", ui.GetInputMode())
	}
}

func TestTerraformContext_ConcurrentPinUnpin(t *testing.T) {
	tc := &TerraformContext{}

	var wg sync.WaitGroup
	addresses := []string{
		"aws_instance.a",
		"aws_instance.b",
		"aws_instance.c",
		"aws_instance.d",
		"aws_instance.e",
	}

	// Concurrently pin all addresses
	for _, addr := range addresses {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			tc.Pin(a)
		}(addr)
	}
	wg.Wait()

	if tc.PinnedCount() != 5 {
		t.Fatalf("expected 5 pinned after concurrent pins, got %d", tc.PinnedCount())
	}

	// Concurrently unpin all addresses
	for _, addr := range addresses {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			tc.Unpin(a)
		}(addr)
	}
	wg.Wait()

	if tc.PinnedCount() != 0 {
		t.Fatalf("expected 0 pinned after concurrent unpins, got %d", tc.PinnedCount())
	}
}

func TestTerraformContext_ConcurrentPinSameAddress(t *testing.T) {
	tc := &TerraformContext{}

	var wg sync.WaitGroup
	// Pin the same address from 100 goroutines
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tc.Pin("aws_instance.shared")
		}()
	}
	wg.Wait()

	// Idempotency: should still be just 1
	if tc.PinnedCount() != 1 {
		t.Fatalf("expected 1 pinned (idempotent), got %d", tc.PinnedCount())
	}
}

func TestTerraformContext_ConcurrentMixedOperations(t *testing.T) {
	tc := &TerraformContext{}

	var wg sync.WaitGroup

	// Mix of pins, unpins, reads concurrently
	for i := 0; i < 50; i++ {
		wg.Add(3)
		addr := "aws_instance.resource"
		go func() {
			defer wg.Done()
			tc.Pin(addr)
		}()
		go func() {
			defer wg.Done()
			tc.IsPinned(addr)
		}()
		go func() {
			defer wg.Done()
			tc.GetPinned()
		}()
	}
	wg.Wait()

	// No panics or data races is the success criterion here.
	// Just verify state is consistent.
	count := tc.PinnedCount()
	if count > 1 {
		t.Fatalf("expected at most 1 pinned, got %d", count)
	}
}

func TestUIContext_ConcurrentSizeUpdates(t *testing.T) {
	ui := &UIContext{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			ui.SetSize(n, n*2)
		}(i)
		go func() {
			defer wg.Done()
			ui.GetSize()
		}()
	}
	wg.Wait()

	// No panics or data races is the success criterion.
	w, h := ui.GetSize()
	if w < 0 || h < 0 {
		t.Fatalf("unexpected negative dimensions: %d x %d", w, h)
	}
}
