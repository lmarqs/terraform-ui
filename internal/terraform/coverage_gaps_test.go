package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/lmarqs/terraform-ui/internal/editor"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// --- loader.go:49 parseShowState ---

func TestParseShowState_WhenStateHasInvalidJSON_ShouldReturnError(t *testing.T) {
	_, _, err := parseShowState([]byte(`{not valid json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseShowState_WhenStateHasNilValues_ShouldReturnEmptyResources(t *testing.T) {
	data := []byte(`{"format_version":"1.0","values":null}`)
	resources, state, err := parseShowState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("len(resources) = %d, want 0", len(resources))
	}
	if state == nil {
		t.Fatal("state should not be nil")
	}
}

func TestParseShowState_WhenStateHasResources_ShouldReturnParsedResources(t *testing.T) {
	data := []byte(`{
		"format_version":"1.0",
		"values": {
			"root_module": {
				"resources": [{
					"address": "aws_instance.web",
					"type": "aws_instance",
					"name": "web",
					"provider_name": "registry.terraform.io/hashicorp/aws",
					"values": {"id": "i-123"},
					"sensitive_values": {}
				}]
			}
		}
	}`)
	resources, state, err := parseShowState(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("len(resources) = %d, want 1", len(resources))
	}
	if resources[0].Address != "aws_instance.web" {
		t.Errorf("address = %q, want aws_instance.web", resources[0].Address)
	}
	if state == nil || state.Values == nil {
		t.Fatal("state.Values should not be nil")
	}
}

// --- macro_service.go:42 SetApplyError ---

func TestMacroService_WhenSetApplyError_ShouldReturnErrorOnApply(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	expectedErr := errors.New("apply failed")
	svc.SetApplyError(expectedErr)

	err := svc.Apply(context.Background(), sdk.ApplyOptions{})
	if err == nil {
		t.Fatal("expected error from Apply")
	}
	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

func TestMacroService_WhenSetApplyErrorNil_ShouldReturnNilOnApply(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	svc.SetApplyError(errors.New("initial error"))
	svc.SetApplyError(nil)

	err := svc.Apply(context.Background(), sdk.ApplyOptions{})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// --- macro_service.go:96 Show (with cached state) ---

func TestMacroService_WhenShowWithCachedState_ShouldReturnResourceJSON(t *testing.T) {
	cache := NewServiceCache()
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123", "ami": "ami-abc"},
						SensitiveValues: json.RawMessage(`{}`),
					},
				},
			},
		},
	}
	cache.SetState([]sdk.Resource{{Address: "aws_instance.web"}}, state)

	svc := NewMacroService("terraform", cache)
	result, err := svc.Show(context.Background(), "aws_instance.web")
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if result == "" || result == "{}" {
		t.Error("Show() returned empty/default when resource should exist")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if parsed["address"] != "aws_instance.web" {
		t.Errorf("address = %v, want aws_instance.web", parsed["address"])
	}
}

func TestMacroService_WhenShowWithCachedStateAndMissingResource_ShouldReturnError(t *testing.T) {
	cache := NewServiceCache()
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123"},
						SensitiveValues: json.RawMessage(`{}`),
					},
				},
			},
		},
	}
	cache.SetState([]sdk.Resource{{Address: "aws_instance.web"}}, state)

	svc := NewMacroService("terraform", cache)
	_, err := svc.Show(context.Background(), "aws_instance.nonexistent")
	if err == nil {
		t.Fatal("expected error for missing resource")
	}
}

// --- macro_service.go:170 buildInitFlags ---

func TestBuildInitFlags_WhenAllOptionsSet_ShouldProduceCorrectFlags(t *testing.T) {
	backendFalse := false
	opts := sdk.InitOptions{
		Upgrade:       true,
		Reconfigure:   true,
		Backend:       &backendFalse,
		BackendConfig: []string{"key=value", "region=us-east-1"},
		ExtraArgs:     []string{"-input=false"},
	}

	flags := buildInitFlags(opts)

	expected := []string{
		"-upgrade",
		"-reconfigure",
		"-backend=false",
		"-backend-config=key=value",
		"-backend-config=region=us-east-1",
		"-input=false",
	}

	if len(flags) != len(expected) {
		t.Fatalf("len(flags) = %d, want %d; flags = %v", len(flags), len(expected), flags)
	}
	for i, want := range expected {
		if flags[i] != want {
			t.Errorf("flags[%d] = %q, want %q", i, flags[i], want)
		}
	}
}

func TestBuildInitFlags_WhenBackendTrue_ShouldNotIncludeBackendFlag(t *testing.T) {
	backendTrue := true
	opts := sdk.InitOptions{
		Backend: &backendTrue,
	}

	flags := buildInitFlags(opts)
	for _, f := range flags {
		if f == "-backend=false" {
			t.Error("flags should not include -backend=false when backend is true")
		}
	}
}

func TestBuildInitFlags_WhenBackendNil_ShouldNotIncludeBackendFlag(t *testing.T) {
	opts := sdk.InitOptions{
		Backend: nil,
	}

	flags := buildInitFlags(opts)
	for _, f := range flags {
		if f == "-backend=false" {
			t.Error("flags should not include -backend=false when backend is nil")
		}
	}
}

func TestBuildInitFlags_WhenEmpty_ShouldReturnNil(t *testing.T) {
	flags := buildInitFlags(sdk.InitOptions{})
	if flags != nil {
		t.Errorf("expected nil, got %v", flags)
	}
}

func TestMacroService_WhenInitWithFlags_ShouldRecordCommand(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	backendFalse := false
	opts := sdk.InitOptions{
		Upgrade:       true,
		Reconfigure:   true,
		Backend:       &backendFalse,
		BackendConfig: []string{"key=val"},
		ExtraArgs:     []string{"-input=false"},
	}
	svc.Init(context.Background(), opts)

	cmds := svc.Commands()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	expected := "terraform init -upgrade -reconfigure -backend=false -backend-config=key=val -input=false"
	if cmds[0].String() != expected {
		t.Errorf("got %q, want %q", cmds[0].String(), expected)
	}
}

// --- macro_service.go:188 Version ---

func TestMacroService_WhenVersion_ShouldReturnDefaultVersion(t *testing.T) {
	svc := NewMacroService("terraform", nil)
	info, err := svc.Version(context.Background())
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if info == nil {
		t.Fatal("Version() returned nil")
	}
	if info.TerraformVersion != "0.0.0" {
		t.Errorf("TerraformVersion = %q, want %q", info.TerraformVersion, "0.0.0")
	}
}

// --- macro_service.go:206 showFromState ---

func TestShowFromState_WhenNilState_ShouldReturnError(t *testing.T) {
	_, err := showFromState(nil, "aws_instance.web")
	if err == nil {
		t.Fatal("expected error for nil state")
	}
}

func TestShowFromState_WhenNilValues_ShouldReturnError(t *testing.T) {
	state := &tfjson.State{FormatVersion: "1.0", Values: nil}
	_, err := showFromState(state, "aws_instance.web")
	if err == nil {
		t.Fatal("expected error for nil Values")
	}
}

func TestShowFromState_WhenResourceNotFound_ShouldReturnError(t *testing.T) {
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{Address: "aws_instance.web", Type: "aws_instance", Name: "web"},
				},
			},
		},
	}
	_, err := showFromState(state, "aws_instance.nonexistent")
	if err == nil {
		t.Fatal("expected error for missing resource")
	}
}

func TestShowFromState_WhenResourceFound_ShouldReturnJSON(t *testing.T) {
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123"},
						SensitiveValues: json.RawMessage(`{}`),
					},
				},
			},
		},
	}

	result, err := showFromState(state, "aws_instance.web")
	if err != nil {
		t.Fatalf("showFromState() error = %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["address"] != "aws_instance.web" {
		t.Errorf("address = %v, want aws_instance.web", parsed["address"])
	}
	if parsed["type"] != "aws_instance" {
		t.Errorf("type = %v, want aws_instance", parsed["type"])
	}
	if parsed["name"] != "web" {
		t.Errorf("name = %v, want web", parsed["name"])
	}
}

func TestShowFromState_WhenResourceHasSensitiveValues_ShouldProduceValidJSON(t *testing.T) {
	state := &tfjson.State{
		FormatVersion: "1.0",
		Values: &tfjson.StateValues{
			RootModule: &tfjson.StateModule{
				Resources: []*tfjson.StateResource{
					{
						Address:         "aws_instance.web",
						Type:            "aws_instance",
						Name:            "web",
						ProviderName:    "registry.terraform.io/hashicorp/aws",
						AttributeValues: map[string]interface{}{"id": "i-123", "password": "secret"},
						SensitiveValues: json.RawMessage(`{"password": true}`),
					},
				},
			},
		},
	}

	result, err := showFromState(state, "aws_instance.web")
	if err != nil {
		t.Fatalf("showFromState() error = %v", err)
	}

	var parsed struct {
		Values map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed.Values["id"] != "i-123" {
		t.Errorf("id = %v, want i-123", parsed.Values["id"])
	}
}

// --- phantom.go:52 normalizeJSON (marshal error path) ---

func TestNormalizeJSON_WhenInvalidJSON_ShouldReturnOriginalString(t *testing.T) {
	input := "not valid json {"
	result := normalizeJSON(input)
	if result != input {
		t.Errorf("normalizeJSON(%q) = %q, want original string", input, result)
	}
}

func TestNormalizeJSON_WhenValidJSON_ShouldNormalize(t *testing.T) {
	input := `{"b":2,"a":1}`
	result := normalizeJSON(input)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if parsed["a"] != float64(1) || parsed["b"] != float64(2) {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestNormalizeJSON_WhenScalarValue_ShouldReturnAsIs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"number", "42", "42"},
		{"string", `"hello"`, `"hello"`},
		{"boolean", "true", "true"},
		{"null", "null", "null"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeJSON(tt.input)
			if got != tt.want {
				t.Errorf("normalizeJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- plan_parser.go:62 mapActions ---

func TestMapActions_WhenDestroyBeforeCreate_ShouldReturnDeleteThenCreate(t *testing.T) {
	actions := tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}
	result := mapActions(actions)
	if result != ActionDeleteThenCreate {
		t.Errorf("mapActions([delete,create]) = %q, want %q", result, ActionDeleteThenCreate)
	}
}

func TestMapActions_WhenCreateBeforeDestroy_ShouldReturnCreateThenDelete(t *testing.T) {
	actions := tfjson.Actions{tfjson.ActionCreate, tfjson.ActionDelete}
	result := mapActions(actions)
	if result != ActionCreateThenDelete {
		t.Errorf("mapActions([create,delete]) = %q, want %q", result, ActionCreateThenDelete)
	}
}

func TestMapActions_WhenUnknownActions_ShouldReturnNoOp(t *testing.T) {
	actions := tfjson.Actions{"unknown-action"}
	result := mapActions(actions)
	if result != ActionNoOp {
		t.Errorf("mapActions([unknown]) = %q, want %q", result, ActionNoOp)
	}
}

// --- plan_parser.go:136 marshalValue (error path) ---

func TestMarshalValue_WhenNil_ShouldReturnNull(t *testing.T) {
	result := marshalValue(nil)
	if result != "null" {
		t.Errorf("marshalValue(nil) = %q, want %q", result, "null")
	}
}

func TestMarshalValue_WhenJsonSerializable_ShouldReturnJSON(t *testing.T) {
	result := marshalValue(map[string]interface{}{"key": "value"})
	if result != `{"key":"value"}` {
		t.Errorf("marshalValue(map) = %q, want %q", result, `{"key":"value"}`)
	}
}

func TestMarshalValue_WhenUnsupported_ShouldFallbackToSprintf(t *testing.T) {
	ch := make(chan int)
	result := marshalValue(ch)
	if result == "" {
		t.Error("marshalValue(chan) returned empty string")
	}
}

// --- service_cache.go:52 SeedPlan / service_cache.go:87 SeedState ---
// The gaps are for the "invalid JSON in file" path. When the file exists but
// contains invalid plan/state JSON, it should return an error.

func TestServiceCache_WhenSeedPlanFromFileWithInvalidJSON_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(`{not valid}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedPlan(planFile, nil)
	if err == nil {
		t.Error("SeedPlan() with invalid JSON file: want error")
	}
}

func TestServiceCache_WhenSeedStateFromFileWithInvalidJSON_ShouldReturnError(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(`{not valid}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	err := c.SeedState(stateFile, nil)
	if err == nil {
		t.Error("SeedState() with invalid JSON file: want error")
	}
}

// --- service_cache.go:200 invalidatePlan (sourceFile with corrupted file) ---

func TestServiceCache_WhenInvalidatePlanWithCorruptedFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	planFile := filepath.Join(dir, "plan.json")
	if err := os.WriteFile(planFile, []byte(minimalPlanJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedPlan(planFile, nil); err != nil {
		t.Fatal(err)
	}

	plan, ok := c.GetPlan()
	if !ok || plan == nil {
		t.Fatal("plan should be cached before corruption")
	}

	if err := os.WriteFile(planFile, []byte(`{corrupt data`), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	plan, ok = c.GetPlan()
	if ok {
		t.Error("GetPlan() ok = true after invalidation with corrupted file, want false")
	}
	if plan != nil {
		t.Errorf("GetPlan() = %v, want nil", plan)
	}
}

func TestServiceCache_WhenInvalidatePlanWithSourceNone_ShouldDoNothing(t *testing.T) {
	c := NewServiceCache()
	c.InvalidateAll()

	plan, ok := c.GetPlan()
	if ok {
		t.Error("GetPlan() ok = true, want false")
	}
	if plan != nil {
		t.Errorf("GetPlan() = %v, want nil", plan)
	}
}

func TestServiceCache_WhenInvalidatePlanWithSourceStdin_ShouldPreserveData(t *testing.T) {
	c := NewServiceCache()
	if err := c.SeedPlan("", []byte(minimalPlanJSON)); err != nil {
		t.Fatal(err)
	}

	c.InvalidateAll()

	plan, ok := c.GetPlan()
	if !ok {
		t.Fatal("GetPlan() ok = false, want true (stdin is immutable)")
	}
	if plan == nil {
		t.Fatal("GetPlan() = nil, want non-nil")
	}
}

// --- service_cache.go:226 invalidateState (sourceFile with corrupted file) ---

func TestServiceCache_WhenInvalidateStateWithCorruptedFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	resources, ok := c.GetResources()
	if !ok || resources == nil {
		t.Fatal("state should be cached before corruption")
	}

	if err := os.WriteFile(stateFile, []byte(`{corrupt data`), 0o644); err != nil {
		t.Fatal(err)
	}

	c.InvalidateState()

	resources, ok = c.GetResources()
	if ok {
		t.Error("GetResources() ok = true after invalidation with corrupted file, want false")
	}
	if resources != nil {
		t.Errorf("GetResources() = %v, want nil", resources)
	}
}

func TestServiceCache_WhenInvalidateStateWithSourceNone_ShouldDoNothing(t *testing.T) {
	c := NewServiceCache()
	c.InvalidateState()

	resources, ok := c.GetResources()
	if ok {
		t.Error("GetResources() ok = true, want false")
	}
	if resources != nil {
		t.Errorf("GetResources() = %v, want nil", resources)
	}
}

func TestServiceCache_WhenInvalidateStateWithDeletedFile_ShouldClearData(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.json")
	if err := os.WriteFile(stateFile, []byte(minimalStateJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewServiceCache()
	if err := c.SeedState(stateFile, nil); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(stateFile); err != nil {
		t.Fatal(err)
	}

	c.InvalidateState()

	resources, ok := c.GetResources()
	if ok {
		t.Error("GetResources() ok = true after invalidation with deleted file, want false")
	}
	if resources != nil {
		t.Errorf("GetResources() = %v, want nil", resources)
	}

	state, sok := c.GetState()
	if sok {
		t.Error("GetState() ok = true after invalidation with deleted file, want false")
	}
	if state != nil {
		t.Errorf("GetState() = %v, want nil", state)
	}
}

// --- source.go:21 NewSourceIndex (node_modules skip, .tofu file support) ---

func TestNewSourceIndex_WhenNodeModulesPresent_ShouldSkipDirectory(t *testing.T) {
	dir := t.TempDir()
	nodeModDir := filepath.Join(dir, "node_modules", "some-package")
	if err := os.MkdirAll(nodeModDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(nodeModDir, "something.tf"), `
resource "aws_s3_bucket" "hidden" {
  bucket = "hidden"
}
`)
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_s3_bucket" "visible" {
  bucket = "visible"
}
`)

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	if _, ok := idx.Lookup("aws_s3_bucket.hidden"); ok {
		t.Error("aws_s3_bucket.hidden should not be indexed (inside node_modules)")
	}
	if _, ok := idx.Lookup("aws_s3_bucket.visible"); !ok {
		t.Error("aws_s3_bucket.visible should be indexed")
	}
}

func TestNewSourceIndex_WhenTofuFilesPresent_ShouldIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tofu"), `
resource "aws_instance" "tofu_resource" {
  ami = "ami-123"
}
`)

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	if _, ok := idx.Lookup("aws_instance.tofu_resource"); !ok {
		t.Error("aws_instance.tofu_resource should be indexed from .tofu file")
	}
}

// --- source.go:84 lookupModuleCall (bare module prefix without dot after segment) ---

func TestLookupModuleCall_WhenModuleWithIndexedKey_ShouldResolveViaStrippedIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
module "users" {
  source   = "./modules/users"
  for_each = var.users
}
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	loc, ok := idx.Lookup(`module.users["admin"].aws_iam_user.this`)
	if !ok {
		t.Fatal("Lookup() should fall back to module.users declaration via lookupModuleCall")
	}
	if loc.Line != 2 {
		t.Errorf("Line = %d, want 2", loc.Line)
	}
}

func TestLookupModuleCall_WhenNoModulePrefix_ShouldReturnFalse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {}
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	_, ok := idx.lookupModuleCall("aws_instance.nonexistent")
	if ok {
		t.Error("lookupModuleCall() should return false for non-module address")
	}
}

func TestLookupModuleCall_WhenModuleNotDeclared_ShouldReturnFalse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {}
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	_, ok := idx.lookupModuleCall("module.unknown.aws_instance.web")
	if ok {
		t.Error("lookupModuleCall() should return false when module is not declared")
	}
}

func TestLookupModuleCall_WhenIndexedModuleWithBareNotFound_ShouldTryIndexedPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
module "users" {
  source = "./modules/users"
}
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	// Manually add an indexed module location to test the bare != modulePath branch
	// where bare is NOT found but modulePath IS found
	idx.locations[`module.teams["dev"]`] = editor.SourceLocation{File: "fake.tf", Line: 10}

	loc, ok := idx.lookupModuleCall(`module.teams["dev"].aws_iam_team.this`)
	if !ok {
		t.Fatal("lookupModuleCall() should find indexed module path")
	}
	if loc.Line != 10 {
		t.Errorf("Line = %d, want 10", loc.Line)
	}
}

func TestLookupModuleCall_WhenModuleHasNoDotsAfterName_ShouldBreak(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {}
`)
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	_, ok := idx.lookupModuleCall("module.standalone")
	if ok {
		t.Error("lookupModuleCall() should return false for module address with no dot after segment")
	}
}

// --- source.go:149 LookupFile (with .tofu fallback) ---

func TestLookupFile_WhenOnlyTofuFiles_ShouldReturnFirstTofuFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "config.tofu"), `resource "null_resource" "x" {}`)

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	loc, ok := idx.LookupFile(dir)
	if !ok {
		t.Fatal("LookupFile() returned false")
	}
	if loc.File != filepath.Join(dir, "config.tofu") {
		t.Errorf("File = %q, want %q", loc.File, filepath.Join(dir, "config.tofu"))
	}
}

func TestLookupFile_WhenNonexistentDirectory_ShouldReturnFalse(t *testing.T) {
	dir := t.TempDir()
	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	_, ok := idx.LookupFile(filepath.Join(dir, "nonexistent"))
	if ok {
		t.Error("LookupFile() returned true for nonexistent directory")
	}
}

// --- source.go:179 scanFile (unreadable file path) ---

func TestScanFile_WhenFileUnreadable_ShouldNotPanic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {}
`)

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	idx.scanFile(filepath.Join(dir, "nonexistent.tf"))
	// Should not panic - just skip unreadable files
}

// --- state_helpers.go:55 RedactSensitiveValues ---

func TestRedactSensitiveValues_WhenNilValues_ShouldReturnNil(t *testing.T) {
	result := RedactSensitiveValues(nil, nil)
	if result != nil {
		t.Errorf("RedactSensitiveValues(nil, nil) = %v, want nil", result)
	}
}

func TestRedactSensitiveValues_WhenNoSensitive_ShouldReturnOriginalValues(t *testing.T) {
	values := map[string]interface{}{
		"id":   "i-123",
		"name": "web",
	}
	result := RedactSensitiveValues(values, nil)
	if result["id"] != "i-123" {
		t.Errorf("id = %v, want i-123", result["id"])
	}
	if result["name"] != "web" {
		t.Errorf("name = %v, want web", result["name"])
	}
}

func TestRedactSensitiveValues_WhenSensitiveMap_ShouldRedactMarkedKeys(t *testing.T) {
	values := map[string]interface{}{
		"id":       "i-123",
		"password": "secret123",
		"token":    "tok-abc",
	}
	sensitive := map[string]interface{}{
		"password": true,
		"token":    true,
	}
	result := RedactSensitiveValues(values, sensitive)
	if result["id"] != "i-123" {
		t.Errorf("id = %v, want i-123", result["id"])
	}
	if result["password"] != "(sensitive)" {
		t.Errorf("password = %v, want (sensitive)", result["password"])
	}
	if result["token"] != "(sensitive)" {
		t.Errorf("token = %v, want (sensitive)", result["token"])
	}
}

func TestRedactSensitiveValues_WhenSensitiveBoolTrue_ShouldRedactAllKeys(t *testing.T) {
	values := map[string]interface{}{
		"id":   "i-123",
		"name": "web",
	}
	result := RedactSensitiveValues(values, true)
	if result["id"] != "(sensitive)" {
		t.Errorf("id = %v, want (sensitive)", result["id"])
	}
	if result["name"] != "(sensitive)" {
		t.Errorf("name = %v, want (sensitive)", result["name"])
	}
}

func TestRedactSensitiveValues_WhenSensitiveBoolFalse_ShouldNotRedact(t *testing.T) {
	values := map[string]interface{}{
		"id":   "i-123",
		"name": "web",
	}
	result := RedactSensitiveValues(values, false)
	if result["id"] != "i-123" {
		t.Errorf("id = %v, want i-123", result["id"])
	}
	if result["name"] != "web" {
		t.Errorf("name = %v, want web", result["name"])
	}
}

// --- state_helpers.go:70 isSensitiveKey ---

func TestIsSensitiveKey_WhenNilSensitive_ShouldReturnFalse(t *testing.T) {
	result := isSensitiveKey(nil, "password")
	if result {
		t.Error("isSensitiveKey(nil, ...) = true, want false")
	}
}

func TestIsSensitiveKey_WhenBoolTrue_ShouldReturnTrue(t *testing.T) {
	result := isSensitiveKey(true, "anything")
	if !result {
		t.Error("isSensitiveKey(true, ...) = false, want true")
	}
}

func TestIsSensitiveKey_WhenBoolFalse_ShouldReturnFalse(t *testing.T) {
	result := isSensitiveKey(false, "anything")
	if result {
		t.Error("isSensitiveKey(false, ...) = true, want false")
	}
}

func TestIsSensitiveKey_WhenMapWithKeyTrue_ShouldReturnTrue(t *testing.T) {
	sensitive := map[string]interface{}{
		"password": true,
		"token":    true,
	}
	if !isSensitiveKey(sensitive, "password") {
		t.Error("isSensitiveKey(map, 'password') = false, want true")
	}
}

func TestIsSensitiveKey_WhenMapWithKeyFalse_ShouldReturnFalse(t *testing.T) {
	sensitive := map[string]interface{}{
		"id": false,
	}
	if isSensitiveKey(sensitive, "id") {
		t.Error("isSensitiveKey(map, 'id' => false) = true, want false")
	}
}

func TestIsSensitiveKey_WhenMapWithKeyAbsent_ShouldReturnFalse(t *testing.T) {
	sensitive := map[string]interface{}{
		"password": true,
	}
	if isSensitiveKey(sensitive, "id") {
		t.Error("isSensitiveKey(map, 'id') = true, want false")
	}
}

func TestIsSensitiveKey_WhenMapWithNonBoolValue_ShouldReturnTrue(t *testing.T) {
	sensitive := map[string]interface{}{
		"nested": map[string]interface{}{"inner": true},
	}
	if !isSensitiveKey(sensitive, "nested") {
		t.Error("isSensitiveKey(map, 'nested' => non-bool non-nil) = false, want true")
	}
}

func TestIsSensitiveKey_WhenUnsupportedType_ShouldReturnFalse(t *testing.T) {
	result := isSensitiveKey("string-value", "key")
	if result {
		t.Error("isSensitiveKey(string, ...) = true, want false")
	}
}

func TestIsSensitiveKey_WhenMapWithNilValue_ShouldReturnFalse(t *testing.T) {
	sensitive := map[string]interface{}{
		"key": nil,
	}
	if isSensitiveKey(sensitive, "key") {
		t.Error("isSensitiveKey(map, 'key' => nil) = true, want false")
	}
}

// --- source.go:21 NewSourceIndex (inaccessible path during walk) ---

func TestNewSourceIndex_WhenInaccessibleSubdirectory_ShouldSkipGracefully(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {}
`)
	restrictedDir := filepath.Join(dir, "restricted")
	if err := os.MkdirAll(restrictedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(restrictedDir, "secret.tf"), `
resource "aws_instance" "secret" {}
`)
	if err := os.Chmod(restrictedDir, 0o000); err != nil {
		t.Skip("cannot restrict directory permissions on this OS")
	}
	t.Cleanup(func() { os.Chmod(restrictedDir, 0o755) })

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() should not error on inaccessible paths: %v", err)
	}
	if _, ok := idx.Lookup("aws_instance.web"); !ok {
		t.Error("accessible resource should still be indexed")
	}
}

// --- source.go:179 scanFile with non-tf file content ---

func TestScanFile_WhenFileHasNoBlocks_ShouldNotAddLocations(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "variables.tf"), `
variable "name" {
  type    = string
  default = "hello"
}

locals {
  greeting = "world"
}
`)

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v", err)
	}

	if idx.Count() != 0 {
		t.Errorf("Count() = %d, want 0 (no resource/data/module blocks)", idx.Count())
	}
}

func TestNewSourceIndex_WhenBrokenSymlink_ShouldSkipGracefully(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.tf"), `
resource "aws_instance" "web" {}
`)
	brokenLink := filepath.Join(dir, "broken.tf")
	os.Symlink("/nonexistent/target.tf", brokenLink)

	idx, err := NewSourceIndex(dir)
	if err != nil {
		t.Fatalf("NewSourceIndex() should not error on broken symlinks: %v", err)
	}
	if _, ok := idx.Lookup("aws_instance.web"); !ok {
		t.Error("valid resource should still be indexed")
	}
}

func TestNewSourceIndex_WhenDirectoryDoesNotExist_ShouldReturnEmptyIndex(t *testing.T) {
	idx, err := NewSourceIndex("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("NewSourceIndex() error = %v (should gracefully handle missing dir)", err)
	}
	if idx.Count() != 0 {
		t.Errorf("Count() = %d, want 0", idx.Count())
	}
}
