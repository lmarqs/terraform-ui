package terraform

import (
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
)

func TestParsePlan_NilPlan(t *testing.T) {
	summary := ParsePlan(nil)
	if summary == nil {
		t.Fatal("ParsePlan(nil) should return non-nil summary")
	}
	if len(summary.Changes) != 0 {
		t.Errorf("ParsePlan(nil).Changes length = %d, want 0", len(summary.Changes))
	}
}

func TestParsePlan_NilResourceChanges(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: nil,
	}

	summary := ParsePlan(plan)
	if len(summary.Changes) != 0 {
		t.Errorf("parsePlan with nil ResourceChanges: Changes length = %d, want 0", len(summary.Changes))
	}
}

func TestParsePlan_SkipsNoOp(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionNoop},
				},
			},
		},
	}

	summary := ParsePlan(plan)
	if len(summary.Changes) != 0 {
		t.Errorf("parsePlan should skip no-op: Changes length = %d, want 0", len(summary.Changes))
	}
}

func TestParsePlan_SkipsRead(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "data.aws_ami.latest",
				Type:         "aws_ami",
				Name:         "latest",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionRead},
				},
			},
		},
	}

	summary := ParsePlan(plan)
	if len(summary.Changes) != 0 {
		t.Errorf("parsePlan should skip read: Changes length = %d, want 0", len(summary.Changes))
	}
	if summary.ToRead != 1 {
		t.Errorf("parsePlan.ToRead = %d, want 1", summary.ToRead)
	}
}

func TestParsePlan_CountsCreate(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionCreate},
				},
			},
		},
	}

	summary := ParsePlan(plan)
	if summary.ToCreate != 1 {
		t.Errorf("parsePlan.ToCreate = %d, want 1", summary.ToCreate)
	}
	if len(summary.Changes) != 1 {
		t.Fatalf("parsePlan.Changes length = %d, want 1", len(summary.Changes))
	}
	if summary.Changes[0].Action != ActionCreate {
		t.Errorf("parsePlan.Changes[0].Action = %q, want %q", summary.Changes[0].Action, ActionCreate)
	}
}

func TestParsePlan_CountsUpdate(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionUpdate},
				},
			},
		},
	}

	summary := ParsePlan(plan)
	if summary.ToUpdate != 1 {
		t.Errorf("parsePlan.ToUpdate = %d, want 1", summary.ToUpdate)
	}
}

func TestParsePlan_CountsDelete(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionDelete},
				},
			},
		},
	}

	summary := ParsePlan(plan)
	if summary.ToDelete != 1 {
		t.Errorf("parsePlan.ToDelete = %d, want 1", summary.ToDelete)
	}
}

func TestParsePlan_CountsReplace(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate},
				},
			},
			{
				Address:      "aws_instance.api",
				Type:         "aws_instance",
				Name:         "api",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionCreate, tfjson.ActionDelete},
				},
			},
		},
	}

	summary := ParsePlan(plan)
	if summary.ToReplace != 2 {
		t.Errorf("parsePlan.ToReplace = %d, want 2", summary.ToReplace)
	}
}

func TestParsePlan_NilChange(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change:       nil, // nil change should be skipped
			},
		},
	}

	summary := ParsePlan(plan)
	if len(summary.Changes) != 0 {
		t.Errorf("parsePlan with nil Change: Changes length = %d, want 0", len(summary.Changes))
	}
}

func TestParsePlan_ExtractsResourceMetadata(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address:      "module.vpc.aws_subnet.private",
				Type:         "aws_subnet",
				Name:         "private",
				ProviderName: "registry.terraform.io/hashicorp/aws",
				Change: &tfjson.Change{
					Actions: tfjson.Actions{tfjson.ActionCreate},
				},
			},
		},
	}

	summary := ParsePlan(plan)
	if len(summary.Changes) != 1 {
		t.Fatalf("parsePlan.Changes length = %d, want 1", len(summary.Changes))
	}

	change := summary.Changes[0]
	if change.Resource.Address != "module.vpc.aws_subnet.private" {
		t.Errorf("Resource.Address = %q, want %q", change.Resource.Address, "module.vpc.aws_subnet.private")
	}
	if change.Resource.Type != "aws_subnet" {
		t.Errorf("Resource.Type = %q, want %q", change.Resource.Type, "aws_subnet")
	}
	if change.Resource.Name != "private" {
		t.Errorf("Resource.Name = %q, want %q", change.Resource.Name, "private")
	}
	if change.Resource.Module != "module.vpc" {
		t.Errorf("Resource.Module = %q, want %q", change.Resource.Module, "module.vpc")
	}
	if change.Resource.ProviderName != "registry.terraform.io/hashicorp/aws" {
		t.Errorf("Resource.ProviderName = %q, want %q", change.Resource.ProviderName, "registry.terraform.io/hashicorp/aws")
	}
}

func TestMapActions(t *testing.T) {
	tests := []struct {
		name     string
		actions  tfjson.Actions
		expected Action
	}{
		{"noop", tfjson.Actions{tfjson.ActionNoop}, ActionNoOp},
		{"read", tfjson.Actions{tfjson.ActionRead}, ActionRead},
		{"create", tfjson.Actions{tfjson.ActionCreate}, ActionCreate},
		{"update", tfjson.Actions{tfjson.ActionUpdate}, ActionUpdate},
		{"delete", tfjson.Actions{tfjson.ActionDelete}, ActionDelete},
		{"delete then create", tfjson.Actions{tfjson.ActionDelete, tfjson.ActionCreate}, ActionDeleteThenCreate},
		{"create then delete", tfjson.Actions{tfjson.ActionCreate, tfjson.ActionDelete}, ActionCreateThenDelete},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapActions(tt.actions)
			if result != tt.expected {
				t.Errorf("mapActions(%v) = %q, want %q", tt.actions, result, tt.expected)
			}
		})
	}
}

func TestParseAttributeDiffs_NilBeforeAndAfter(t *testing.T) {
	change := &tfjson.Change{
		Before: nil,
		After:  nil,
	}

	diffs := parseAttributeDiffs(change)
	if len(diffs) != 0 {
		t.Errorf("parseAttributeDiffs with nil Before/After: length = %d, want 0", len(diffs))
	}
}

func TestParseAttributeDiffs_CreateResource(t *testing.T) {
	change := &tfjson.Change{
		Before: nil,
		After: map[string]interface{}{
			"name": "my-instance",
			"size": "t3.micro",
		},
	}

	diffs := parseAttributeDiffs(change)
	if len(diffs) != 2 {
		t.Fatalf("parseAttributeDiffs length = %d, want 2", len(diffs))
	}

	// Find the "name" diff
	var nameDiff *AttributeDiff
	for i := range diffs {
		if diffs[i].Key == "name" {
			nameDiff = &diffs[i]
			break
		}
	}
	if nameDiff == nil {
		t.Fatal("expected 'name' attribute diff")
	}
	if nameDiff.OldValue != "null" {
		t.Errorf("name OldValue = %q, want %q", nameDiff.OldValue, "null")
	}
	if nameDiff.NewValue != `"my-instance"` {
		t.Errorf("name NewValue = %q, want %q", nameDiff.NewValue, `"my-instance"`)
	}
}

func TestParseAttributeDiffs_UpdateResource(t *testing.T) {
	change := &tfjson.Change{
		Before: map[string]interface{}{
			"name": "old-name",
			"size": "t3.micro",
			"tags": map[string]interface{}{"env": "dev"},
		},
		After: map[string]interface{}{
			"name": "new-name",
			"size": "t3.micro", // unchanged
			"tags": map[string]interface{}{"env": "prod"},
		},
	}

	diffs := parseAttributeDiffs(change)

	// "size" is unchanged, so should not appear
	for _, d := range diffs {
		if d.Key == "size" {
			t.Error("unchanged attribute 'size' should not appear in diffs")
		}
	}

	// "name" should be in diffs
	found := false
	for _, d := range diffs {
		if d.Key == "name" {
			found = true
			if d.OldValue != `"old-name"` {
				t.Errorf("name OldValue = %q, want %q", d.OldValue, `"old-name"`)
			}
			if d.NewValue != `"new-name"` {
				t.Errorf("name NewValue = %q, want %q", d.NewValue, `"new-name"`)
			}
		}
	}
	if !found {
		t.Error("expected 'name' attribute diff")
	}
}

func TestParseAttributeDiffs_SensitiveAttribute(t *testing.T) {
	change := &tfjson.Change{
		Before: map[string]interface{}{
			"password": "old-secret",
			"name":     "app",
		},
		After: map[string]interface{}{
			"password": "new-secret",
			"name":     "app",
		},
		BeforeSensitive: map[string]interface{}{
			"password": true,
		},
		AfterSensitive: map[string]interface{}{
			"password": true,
		},
	}

	diffs := parseAttributeDiffs(change)

	for _, d := range diffs {
		if d.Key == "password" {
			if !d.Sensitive {
				t.Error("password attribute should be marked sensitive")
			}
			return
		}
	}
	t.Error("expected 'password' attribute diff")
}

func TestParseAttributeDiffs_DeleteResource(t *testing.T) {
	change := &tfjson.Change{
		Before: map[string]interface{}{
			"name": "doomed-instance",
		},
		After: nil,
	}

	diffs := parseAttributeDiffs(change)
	if len(diffs) != 1 {
		t.Fatalf("parseAttributeDiffs length = %d, want 1", len(diffs))
	}
	if diffs[0].Key != "name" {
		t.Errorf("diff Key = %q, want %q", diffs[0].Key, "name")
	}
	if diffs[0].NewValue != "null" {
		t.Errorf("diff NewValue = %q, want %q", diffs[0].NewValue, "null")
	}
}

func TestJsonToMap_Nil(t *testing.T) {
	result := jsonToMap(nil)
	if result == nil {
		t.Fatal("jsonToMap(nil) should return empty map, not nil")
	}
	if len(result) != 0 {
		t.Errorf("jsonToMap(nil) length = %d, want 0", len(result))
	}
}

func TestJsonToMap_ValidMap(t *testing.T) {
	input := map[string]interface{}{"key": "value"}
	result := jsonToMap(input)
	if result["key"] != "value" {
		t.Errorf("jsonToMap result[key] = %v, want %q", result["key"], "value")
	}
}

func TestJsonToMap_NonMapValue(t *testing.T) {
	result := jsonToMap("string value")
	if result == nil {
		t.Fatal("jsonToMap(string) should return empty map")
	}
	if len(result) != 0 {
		t.Errorf("jsonToMap(string) length = %d, want 0", len(result))
	}
}

func TestMarshalValue_Nil(t *testing.T) {
	result := marshalValue(nil)
	if result != "null" {
		t.Errorf("marshalValue(nil) = %q, want %q", result, "null")
	}
}

func TestMarshalValue_String(t *testing.T) {
	result := marshalValue("hello")
	if result != `"hello"` {
		t.Errorf("marshalValue(\"hello\") = %q, want %q", result, `"hello"`)
	}
}

func TestMarshalValue_Number(t *testing.T) {
	result := marshalValue(42.0)
	if result != "42" {
		t.Errorf("marshalValue(42.0) = %q, want %q", result, "42")
	}
}

func TestMarshalValue_Bool(t *testing.T) {
	result := marshalValue(true)
	if result != "true" {
		t.Errorf("marshalValue(true) = %q, want %q", result, "true")
	}
}

func TestMarshalValue_Map(t *testing.T) {
	result := marshalValue(map[string]interface{}{"a": 1})
	// JSON output
	if result != `{"a":1}` {
		t.Errorf("marshalValue(map) = %q, want %q", result, `{"a":1}`)
	}
}

func TestIsKeySensitive_NilSensitive(t *testing.T) {
	result := isKeySensitive(nil, "anything")
	if result {
		t.Error("isKeySensitive(nil) should return false")
	}
}

func TestIsKeySensitive_BoolTrue(t *testing.T) {
	// When sensitive is just "true", all keys are sensitive
	result := isKeySensitive(true, "anything")
	if !result {
		t.Error("isKeySensitive(true) should return true for any key")
	}
}

func TestIsKeySensitive_BoolFalse(t *testing.T) {
	result := isKeySensitive(false, "anything")
	if result {
		t.Error("isKeySensitive(false) should return false")
	}
}

func TestIsKeySensitive_MapWithKey(t *testing.T) {
	sensitive := map[string]interface{}{
		"password": true,
		"name":     false,
	}

	if !isKeySensitive(sensitive, "password") {
		t.Error("isKeySensitive should return true for 'password'")
	}
	if isKeySensitive(sensitive, "name") {
		t.Error("isKeySensitive should return false for 'name'")
	}
	if isKeySensitive(sensitive, "nonexistent") {
		t.Error("isKeySensitive should return false for missing key")
	}
}

func TestIsKeySensitive_MapWithNonBoolValue(t *testing.T) {
	sensitive := map[string]interface{}{
		"nested": map[string]interface{}{"a": true},
	}

	result := isKeySensitive(sensitive, "nested")
	if result {
		t.Error("isKeySensitive with non-bool value should return false")
	}
}

func TestParseStateResources_NilModule(t *testing.T) {
	resources := ParseStateResources(nil)
	if len(resources) != 0 {
		t.Errorf("ParseStateResources(nil) length = %d, want 0", len(resources))
	}
}

func TestParseStateResources_WithResources(t *testing.T) {
	module := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{
				Address:      "aws_instance.web",
				Type:         "aws_instance",
				Name:         "web",
				ProviderName: "registry.terraform.io/hashicorp/aws",
			},
			{
				Address:      "aws_s3_bucket.data",
				Type:         "aws_s3_bucket",
				Name:         "data",
				ProviderName: "registry.terraform.io/hashicorp/aws",
			},
		},
	}

	resources := ParseStateResources(module)
	if len(resources) != 2 {
		t.Fatalf("parseStateResources length = %d, want 2", len(resources))
	}
	if resources[0].Address != "aws_instance.web" {
		t.Errorf("resources[0].Address = %q, want %q", resources[0].Address, "aws_instance.web")
	}
	if resources[1].Address != "aws_s3_bucket.data" {
		t.Errorf("resources[1].Address = %q, want %q", resources[1].Address, "aws_s3_bucket.data")
	}
}

func TestParseStateResources_WithChildModules(t *testing.T) {
	module := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{
				Address:      "aws_instance.root",
				Type:         "aws_instance",
				Name:         "root",
				ProviderName: "registry.terraform.io/hashicorp/aws",
			},
		},
		ChildModules: []*tfjson.StateModule{
			{
				Resources: []*tfjson.StateResource{
					{
						Address:      "module.vpc.aws_subnet.private",
						Type:         "aws_subnet",
						Name:         "private",
						ProviderName: "registry.terraform.io/hashicorp/aws",
					},
				},
			},
		},
	}

	resources := ParseStateResources(module)
	if len(resources) != 2 {
		t.Fatalf("parseStateResources with children: length = %d, want 2", len(resources))
	}
}

func TestFindResourceInState_NilModule(t *testing.T) {
	result := FindResourceInState(nil, "aws_instance.web")
	if result != nil {
		t.Error("FindResourceInState(nil) should return nil")
	}
}

func TestFindResourceInState_Found(t *testing.T) {
	module := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_instance.web"},
			{Address: "aws_s3_bucket.data"},
		},
	}

	result := FindResourceInState(module, "aws_s3_bucket.data")
	if result == nil {
		t.Fatal("findResourceInState should find 'aws_s3_bucket.data'")
	}
	if result.Address != "aws_s3_bucket.data" {
		t.Errorf("found resource Address = %q, want %q", result.Address, "aws_s3_bucket.data")
	}
}

func TestFindResourceInState_NotFound(t *testing.T) {
	module := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_instance.web"},
		},
	}

	result := FindResourceInState(module, "aws_instance.nonexistent")
	if result != nil {
		t.Error("findResourceInState should return nil for non-existent address")
	}
}

func TestFindResourceInState_InChildModule(t *testing.T) {
	module := &tfjson.StateModule{
		Resources: []*tfjson.StateResource{
			{Address: "aws_instance.root"},
		},
		ChildModules: []*tfjson.StateModule{
			{
				Resources: []*tfjson.StateResource{
					{Address: "module.vpc.aws_subnet.private"},
				},
			},
		},
	}

	result := FindResourceInState(module, "module.vpc.aws_subnet.private")
	if result == nil {
		t.Fatal("findResourceInState should find resource in child module")
	}
	if result.Address != "module.vpc.aws_subnet.private" {
		t.Errorf("found resource Address = %q, want %q", result.Address, "module.vpc.aws_subnet.private")
	}
}

func TestNewService(t *testing.T) {
	svc := NewService("/work/dir", "/usr/bin/terraform")
	if svc.workingDir != "/work/dir" {
		t.Errorf("NewService().workingDir = %q, want %q", svc.workingDir, "/work/dir")
	}
	if svc.binaryPath != "/usr/bin/terraform" {
		t.Errorf("NewService().binaryPath = %q, want %q", svc.binaryPath, "/usr/bin/terraform")
	}
}

func TestIsPhantomChange_WithNonJSONValues(t *testing.T) {
	// When attribute diffs have non-JSON values, normalizeJSON returns them as-is
	// If old and new are the same non-JSON string, it's still phantom
	change := &PlanChange{
		Action: ActionUpdate,
		AttributeDiffs: []AttributeDiff{
			{Key: "raw", OldValue: "not-json", NewValue: "not-json"},
		},
	}
	if !IsPhantomChange(change) {
		t.Error("IsPhantomChange should be true when non-JSON values are equal")
	}

	// Different non-JSON values = real change
	change2 := &PlanChange{
		Action: ActionUpdate,
		AttributeDiffs: []AttributeDiff{
			{Key: "raw", OldValue: "old-value", NewValue: "new-value"},
		},
	}
	if IsPhantomChange(change2) {
		t.Error("IsPhantomChange should be false when non-JSON values differ")
	}
}

func TestExtractModule_ModuleAtEnd(t *testing.T) {
	// Edge case: address is just "module" with no following name
	result := ExtractModule("module")
	if result != "module" {
		t.Errorf("ExtractModule(%q) = %q, want %q", "module", result, "module")
	}
}

func TestRiskLevel_String(t *testing.T) {
	tests := []struct {
		level    RiskLevel
		expected string
	}{
		{RiskNone, "none"},
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
		{RiskCritical, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("RiskLevel(%d).String() = %q, want %q", tt.level, result, tt.expected)
			}
		})
	}
}

func TestParsePlan_MixedActions(t *testing.T) {
	plan := &tfjson.Plan{
		ResourceChanges: []*tfjson.ResourceChange{
			{
				Address: "aws_instance.a", Type: "aws_instance", Name: "a",
				ProviderName: "aws",
				Change:       &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionCreate}},
			},
			{
				Address: "aws_instance.b", Type: "aws_instance", Name: "b",
				ProviderName: "aws",
				Change:       &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionUpdate}},
			},
			{
				Address: "aws_instance.c", Type: "aws_instance", Name: "c",
				ProviderName: "aws",
				Change:       &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionDelete}},
			},
			{
				Address: "aws_instance.d", Type: "aws_instance", Name: "d",
				ProviderName: "aws",
				Change:       &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionNoop}},
			},
			{
				Address: "data.aws_ami.e", Type: "aws_ami", Name: "e",
				ProviderName: "aws",
				Change:       &tfjson.Change{Actions: tfjson.Actions{tfjson.ActionRead}},
			},
		},
	}

	summary := ParsePlan(plan)
	if summary.ToCreate != 1 {
		t.Errorf("ToCreate = %d, want 1", summary.ToCreate)
	}
	if summary.ToUpdate != 1 {
		t.Errorf("ToUpdate = %d, want 1", summary.ToUpdate)
	}
	if summary.ToDelete != 1 {
		t.Errorf("ToDelete = %d, want 1", summary.ToDelete)
	}
	if summary.ToRead != 1 {
		t.Errorf("ToRead = %d, want 1", summary.ToRead)
	}
	// no-op and read are excluded from Changes
	if len(summary.Changes) != 3 {
		t.Errorf("Changes length = %d, want 3", len(summary.Changes))
	}
}
