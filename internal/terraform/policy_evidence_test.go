package terraform

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func policyEvidenceJSON(t *testing.T, source string) map[string]any {
	t.Helper()
	summary, err := LoadPlan([]byte(source))
	if err != nil {
		t.Fatalf("Plan review must remain available: %v", err)
	}
	field := reflect.ValueOf(summary).Elem().FieldByName("PolicyEvidence")
	if !field.IsValid() {
		t.Fatal("reviewed plan does not retain PolicyEvidence separately from display diffs")
	}
	data, err := json.Marshal(field.Interface())
	if err != nil {
		t.Fatal(err)
	}
	var evidence map[string]any
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence == nil {
		t.Fatal("reviewed plan has no policy evidence")
	}
	return evidence
}

func policyPlan(resource string) string {
	return `{"format_version":"1.2","resource_changes":[` + resource + `]}`
}

func policyResource(metadata, change string) string {
	return `{"address":"aws_lambda_function.worker","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws"` + metadata + `,"change":` + change + `}`
}

func policyRows(t *testing.T, evidence map[string]any) []any {
	t.Helper()
	if code, _ := evidence["error_code"].(string); code != "" {
		t.Fatalf("unexpected unavailable evidence: %s", code)
	}
	rows, ok := evidence["resources"].([]any)
	if !ok {
		t.Fatalf("resources = %#v, want explicit resource list", evidence["resources"])
	}
	return rows
}

func assertPolicyJSON(t *testing.T, got any, want string) {
	t.Helper()
	var expected any
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("policy evidence = %#v, want %#v", got, expected)
	}
}

func TestLoadPlan_GivenPreciseResourceEvidence_ShouldRetainIndependentRedactedInput(t *testing.T) {
	resource := policyResource(`,"previous_address":"aws_lambda_function.old","index":9007199254740993`, `{"actions":["create","delete"],"before":{"big":9007199254740993,"ordered":[null,2,1],"null":null,"secret":{"token":"do-not-expose"},"flag":false,"text":"old"},"after":{"big":1e3,"ordered":[2,null,1],"secret":{"token":"do-not-expose"},"flag":true,"text":"new"},"after_unknown":{"new_unknown":true},"before_sensitive":{"secret":true},"after_sensitive":{"secret":true},"replace_paths":[["ordered",0],[]]}`)
	source := policyPlan(resource)
	evidence := policyEvidenceJSON(t, source)
	if want := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(source))); evidence["plan_id"] != want {
		t.Errorf("plan_id = %v, want digest of exact reviewed bytes %s", evidence["plan_id"], want)
	}
	rows := policyRows(t, evidence)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	input := rows[0].(map[string]any)["input"]
	assertPolicyJSON(t, input, `{"schema_version":"tfui.risk.input/v1","resource":{"address":"aws_lambda_function.worker","previous_address":"aws_lambda_function.old","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws","index":{"kind":"number","value":"9007199254740993"}},"change":{"actions":["create","delete"],"importing":null,"replace_paths":[["ordered",0],[]],"before":{"kind":"object","entries":{"big":{"kind":"number","value":"9007199254740993"},"ordered":{"kind":"array","items":[{"kind":"null"},{"kind":"number","value":"2"},{"kind":"number","value":"1"}]},"null":{"kind":"null"},"secret":{"kind":"sensitive"},"flag":{"kind":"boolean","value":false},"text":{"kind":"string","value":"old"}}},"after":{"kind":"object","entries":{"big":{"kind":"number","value":"1e3"},"ordered":{"kind":"array","items":[{"kind":"number","value":"2"},{"kind":"null"},{"kind":"number","value":"1"}]},"secret":{"kind":"sensitive"},"flag":{"kind":"boolean","value":true},"text":{"kind":"string","value":"new"},"new_unknown":{"kind":"unknown"}}}}}`)
	neighbor := `{"address":"aws_s3_bucket.neighbor","change":{"actions":["no-op"]}}`
	other := policyEvidenceJSON(t, policyPlan(resource+`,`+neighbor))
	if !reflect.DeepEqual(policyRows(t, other)[0].(map[string]any)["input"], input) {
		t.Error("neighboring resource changed independent policy input")
	}
}

func TestLoadPlan_GivenResourceActions_ShouldDistinguishMutationsAndExclusions(t *testing.T) {
	tests := []struct{ name, metadata, change, exclusion string }{
		{"create", "", `{"actions":["create"],"before":null,"after":{}}`, ""},
		{"update phantom", "", `{"actions":["update"],"before":{"x":[1,2]},"after":{"x":[2,1]}}`, ""},
		{"delete", "", `{"actions":["delete"],"before":{},"after":null}`, ""},
		{"delete create", "", `{"actions":["delete","create"],"before":{},"after":{}}`, ""},
		{"create delete", "", `{"actions":["create","delete"],"before":{},"after":{}}`, ""},
		{"read", "", `{"actions":["read"]}`, "read"},
		{"no op", "", `{"actions":["no-op"]}`, "no_op"},
		{"pure import", "", `{"actions":["no-op"],"importing":{"id":"private"}}`, "pure_import"},
		{"pure move", `,"previous_address":"aws_lambda_function.old"`, `{"actions":["no-op"]}`, "pure_move"},
		{"combined exclusions", `,"previous_address":"aws_lambda_function.old"`, `{"actions":["read"],"importing":{"id":"private"}}`, "pure_move"},
		{"move with update", `,"previous_address":"aws_lambda_function.old"`, `{"actions":["update"],"before":{},"after":{}}`, ""},
		{"import with update", "", `{"actions":["update"],"importing":{"unknown":true},"before":{},"after":{}}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := policyRows(t, policyEvidenceJSON(t, policyPlan(policyResource(tt.metadata, tt.change))))
			if len(rows) != 1 {
				t.Fatalf("rows = %d, want one eligible or excluded change", len(rows))
			}
			row := rows[0].(map[string]any)
			exclusion, _ := row["exclusion"].(string)
			if exclusion != tt.exclusion {
				t.Errorf("exclusion = %q, want %q", exclusion, tt.exclusion)
			}
			if (row["input"] != nil) != (tt.exclusion == "") {
				t.Errorf("input = %#v for exclusion %q", row["input"], tt.exclusion)
			}
			var change map[string]any
			_ = json.Unmarshal([]byte(tt.change), &change)
			if !reflect.DeepEqual(row["actions"], change["actions"]) {
				t.Errorf("actions = %v, want original order %v", row["actions"], change["actions"])
			}
		})
	}
}

func TestLoadPlan_GivenOutputsAndNoMutations_ShouldRetainExcludedChanges(t *testing.T) {
	evidence := policyEvidenceJSON(t, `{"format_version":"1.2","resource_changes":[],"output_changes":{"endpoint":{"actions":["update"],"before":"a","after":"b"}}}`)
	rows := policyRows(t, evidence)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want output exclusion", len(rows))
	}
	row := rows[0].(map[string]any)
	if row["address"] != "output.endpoint" || row["exclusion"] != "output_only" || row["input"] != nil {
		t.Errorf("output exclusion = %#v", row)
	}
	assertPolicyJSON(t, row["actions"], `["update"]`)
	for _, source := range []string{`{"format_version":"1.2"}`, `{"format_version":"1.2","resource_changes":null}`, `{"format_version":"1.2","resource_changes":[]}`, `{"format_version":"1.2","deferred_changes":null}`, `{"format_version":"1.2","deferred_changes":[]}`, `{"format_version":"1.2","complete":false}`} {
		if rows := policyRows(t, policyEvidenceJSON(t, source)); len(rows) != 0 {
			t.Errorf("empty plan evidence has %d rows", len(rows))
		}
	}
}

func TestLoadPlan_GivenOpaqueOrAbsentValues_ShouldPreserveTaggedDistinctions(t *testing.T) {
	tests := []struct{ name, change, side, expected string }{
		{"unknown concrete array", `{"actions":["update"],"before":{},"after":[1,2],"after_unknown":true,"after_sensitive":[false,true]}`, "after", `{"kind":"sensitive_unknown"}`},
		{"unknown concrete object", `{"actions":["update"],"before":{},"after":{"x":1},"after_unknown":true,"after_sensitive":{"x":false}}`, "after", `{"kind":"unknown"}`},
		{"empty containers", `{"actions":["update"],"before":{},"after":{"x":[]}}`, "after", `{"kind":"object","entries":{"x":{"kind":"array","items":[]}}}`},
		{"unknown sensitive array", `{"actions":["update"],"before":{},"after":null,"after_unknown":true,"after_sensitive":[false,true]}`, "after", `{"kind":"sensitive_unknown"}`},
		{"unknown unmarked array", `{"actions":["update"],"before":{},"after":null,"after_unknown":true,"after_sensitive":[false]}`, "after", `{"kind":"unknown"}`},
		{"unknown unmarked object", `{"actions":["update"],"before":{},"after":null,"after_unknown":true,"after_sensitive":{"x":false}}`, "after", `{"kind":"unknown"}`},
		{"unknown wholly sensitive", `{"actions":["update"],"before":{},"after":null,"after_unknown":true,"after_sensitive":true}`, "after", `{"kind":"sensitive_unknown"}`},
		{"before masking independent", `{"actions":["update"],"before":{"x":"secret"},"after":{"x":"public"},"before_sensitive":{"x":true}}`, "before", `{"kind":"object","entries":{"x":{"kind":"sensitive"}}}`},
		{"after unmasked independently", `{"actions":["update"],"before":{"x":"secret"},"after":{"x":"public"},"before_sensitive":{"x":true}}`, "after", `{"kind":"object","entries":{"x":{"kind":"string","value":"public"}}}`},
		{"create absent", `{"actions":["create"],"before":null,"after":{}}`, "before", `{"kind":"absent"}`},
		{"delete absent", `{"actions":["delete"],"before":{},"after":null}`, "after", `{"kind":"absent"}`},
		{"update null", `{"actions":["update"],"before":null,"after":null}`, "after", `{"kind":"null"}`},
		{"whole unknown", `{"actions":["update"],"before":{},"after":null,"after_unknown":true}`, "after", `{"kind":"unknown"}`},
		{"unknown sensitive descendant", `{"actions":["update"],"before":{},"after":null,"after_unknown":true,"after_sensitive":{"nested":true}}`, "after", `{"kind":"sensitive_unknown"}`},
		{"known sensitive ancestor", `{"actions":["update"],"before":{},"after":{},"after_unknown":{"child":true},"after_sensitive":true}`, "after", `{"kind":"sensitive"}`},
		{"array masks", `{"actions":["update"],"before":{},"after":{"list":[1,null]},"after_unknown":{"list":[false,true]},"after_sensitive":{"list":[true,false]}}`, "after", `{"kind":"object","entries":{"list":{"kind":"array","items":[{"kind":"sensitive"},{"kind":"unknown"}]}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := policyRows(t, policyEvidenceJSON(t, policyPlan(policyResource("", tt.change))))
			input := rows[0].(map[string]any)["input"].(map[string]any)
			assertPolicyJSON(t, input["change"].(map[string]any)[tt.side], tt.expected)
		})
	}
}

func TestLoadPlan_GivenInvalidPolicyEvidence_ShouldKeepReviewAndMarkUnavailable(t *testing.T) {
	tests := []struct{ name, source, code string }{
		{"unknown ancestor array child shape mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":[1],"after_unknown":true,"after_sensitive":[{"x":true}]}`)), "input_invalid"},
		{"unrepresentable replacement index", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"replace_paths":[[1e999999999999999999999999]]}`)), "input_invalid"},
		{"object deferred changes", `{"format_version":"1.2","deferred_changes":{}}`, "input_invalid"},
		{"string deferred changes", `{"format_version":"1.2","deferred_changes":"invalid"}`, "input_invalid"},
		{"unknown nested missing parent shape", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"after_unknown":{"nested":{"x":true}}}`)), "input_invalid"},
		{"unknown ancestor established array length mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{"list":[1,2]},"after_unknown":true,"after_sensitive":{"list":[true]}}`)), "input_invalid"},
		{"unknown ancestor established scalar shape mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{"list":1},"after_unknown":true,"after_sensitive":{"list":{"child":true}}}`)), "input_invalid"},
		{"deferred changes", `{"format_version":"1.2","resource_changes":[],"deferred_changes":[{"reason":"instance_count_unknown","resource_change":{"address":"aws_lambda_function.worker","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["create"],"before":null,"after":{}}}}]}`, "input_incomplete"},
		{"null resource row", `{"format_version":"1.2","resource_changes":[null]}`, "input_incomplete"},
		{"missing before", policyPlan(policyResource("", `{"actions":["update"],"after":{}}`)), "input_incomplete"},
		{"missing after", policyPlan(policyResource("", `{"actions":["update"],"before":{}}`)), "input_incomplete"},
		{"empty address", policyPlan(`{"address":"","change":{"actions":["no-op"]}}`), "input_incomplete"},
		{"invalid mode", policyPlan(`{"address":"aws_x.y","mode":"other","type":"aws_x","name":"y","provider_name":"aws","change":{"actions":["update"],"before":{},"after":{}}}`), "input_invalid"},
		{"null replace paths", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"replace_paths":null}`)), "input_invalid"},
		{"nonarray replace paths", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"replace_paths":{}}`)), "input_invalid"},
		{"nonarray path", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"replace_paths":["x"]}`)), "input_invalid"},
		{"negative replacement index", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"replace_paths":[[-1]]}`)), "input_invalid"},
		{"fractional replacement index", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"replace_paths":[[1.5]]}`)), "input_invalid"},
		{"array invalid leaf", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":[1],"after_sensitive":["invalid"]}`)), "input_invalid"},
		{"array child shape mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":[1],"after_sensitive":[{"x":true}]}`)), "input_invalid"},
		{"object mask array mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"after_sensitive":[]}`)), "input_invalid"},
		{"unknown scalar shape mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":1,"after_unknown":{"x":true}}`)), "input_invalid"},
		{"sensitive ancestor array length mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":[1],"after_sensitive":true,"after_unknown":[false,false]}`)), "input_invalid"},
		{"sensitive ancestor object shape mismatch", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"after_sensitive":true,"after_unknown":[]}`)), "input_invalid"},
		{"missing mode", policyPlan(`{"address":"aws_x.y","type":"aws_x","name":"y","provider_name":"aws","change":{"actions":["update"],"before":{},"after":{}}}`), "input_incomplete"},
		{"missing provider", policyPlan(`{"address":"aws_x.y","mode":"managed","type":"aws_x","name":"y","change":{"actions":["update"],"before":{},"after":{}}}`), "input_incomplete"},
		{"missing output actions", `{"format_version":"1.2","output_changes":{"x":{}}}`, "input_incomplete"},
		{"missing actions", policyPlan(policyResource("", `{"before":{},"after":{}}`)), "input_incomplete"},
		{"unsupported actions", policyPlan(policyResource("", `{"actions":["forget"],"before":{},"after":{}}`)), "input_invalid"},
		{"invalid unknown descendant", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":null,"after_unknown":true,"after_sensitive":{"child":"invalid"}}`)), "input_invalid"},
		{"invalid sensitive descendant", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{"child":{}},"after_sensitive":true,"after_unknown":{"child":{"nested":"invalid"}}}`)), "input_invalid"},
		{"mask leaf", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{"x":1},"after_sensitive":{"x":"secret"}}`)), "input_invalid"},
		{"array length", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{"x":[1,2]},"after_sensitive":{"x":[true]}}`)), "input_invalid"},
		{"mask shape", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{"x":1},"after_sensitive":{"x":{"child":true}}}`)), "input_invalid"},
		{"mask absent known key", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"after_sensitive":{"missing":true}}`)), "input_invalid"},
		{"negative index", policyPlan(policyResource(`,"index":-1`, `{"actions":["update"],"before":{},"after":{}}`)), "input_invalid"},
		{"fractional index", policyPlan(policyResource(`,"index":1.5`, `{"actions":["update"],"before":{},"after":{}}`)), "input_invalid"},
		{"boolean replacement path", policyPlan(policyResource("", `{"actions":["update"],"before":{},"after":{},"replace_paths":[[true]]}`)), "input_invalid"},
		{"unmasked import id", policyPlan(policyResource("", `{"actions":["update"],"importing":{"id":"secret"},"before":{},"after":{}}`)), "input_incomplete"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := policyEvidenceJSON(t, tt.source)
			if evidence["error_code"] != tt.code {
				t.Errorf("error_code = %v, want %s", evidence["error_code"], tt.code)
			}
			if rows, _ := evidence["resources"].([]any); len(rows) != 0 {
				t.Errorf("global invalid input must not salvage %d rows", len(rows))
			}
		})
	}
}

func TestLoadPlan_GivenDeleteChange_ShouldProduceAbsentAfterInput(t *testing.T) {
	source := policyPlan(policyResource("", `{"actions":["delete"],"before":{"id":"i-abc123"},"after":null,"after_unknown":{},"before_sensitive":false,"after_sensitive":false}`))
	summary, err := LoadPlan([]byte(source))
	if err != nil {
		t.Fatalf("Plan review must remain available: %v", err)
	}
	if summary.PolicyEvidence.ErrorCode != "" {
		t.Fatalf("PolicyEvidence.ErrorCode = %q, want empty", summary.PolicyEvidence.ErrorCode)
	}
	if len(summary.PolicyEvidence.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(summary.PolicyEvidence.Resources))
	}
	input := summary.PolicyEvidence.Resources[0].Input
	if input == nil {
		t.Fatal("deleted resource has no policy Input")
	}
	if input.Change.After.Kind != "absent" {
		t.Errorf("After.Kind = %q, want absent", input.Change.After.Kind)
	}
}

func TestLoadPlan_GivenEmptyMaskOnNullOrScalar_ShouldTreatValueAsNotMarked(t *testing.T) {
	tests := []struct{ name, change, expected string }{
		{"null empty object unknown mask", `{"actions":["update"],"before":{},"after":null,"after_unknown":{}}`, `{"kind":"null"}`},
		{"null empty array unknown mask", `{"actions":["update"],"before":{},"after":null,"after_unknown":[]}`, `{"kind":"null"}`},
		{"null empty object sensitive mask", `{"actions":["update"],"before":{},"after":null,"after_sensitive":{}}`, `{"kind":"null"}`},
		{"null empty array sensitive mask", `{"actions":["update"],"before":{},"after":null,"after_sensitive":[]}`, `{"kind":"null"}`},
		{"string empty object unknown mask", `{"actions":["update"],"before":{},"after":"x","after_unknown":{}}`, `{"kind":"string","value":"x"}`},
		{"string empty array unknown mask", `{"actions":["update"],"before":{},"after":"x","after_unknown":[]}`, `{"kind":"string","value":"x"}`},
		{"number empty object sensitive mask", `{"actions":["update"],"before":{},"after":1,"after_sensitive":{}}`, `{"kind":"number","value":"1"}`},
		{"boolean empty array sensitive mask", `{"actions":["update"],"before":{},"after":true,"after_sensitive":[]}`, `{"kind":"boolean","value":true}`},
		{"delete absent empty unknown mask", `{"actions":["delete"],"before":{},"after":null,"after_unknown":{}}`, `{"kind":"absent"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := policyRows(t, policyEvidenceJSON(t, policyPlan(policyResource("", tt.change))))
			input := rows[0].(map[string]any)["input"].(map[string]any)
			assertPolicyJSON(t, input["change"].(map[string]any)["after"], tt.expected)
		})
	}
}

func TestLoadPlan_GivenEmptyMaskOnBefore_ShouldTreatValueAsNotMarked(t *testing.T) {
	rows := policyRows(t, policyEvidenceJSON(t, policyPlan(policyResource("", `{"actions":["update"],"before":"old","after":"new","before_sensitive":{}}`))))
	input := rows[0].(map[string]any)["input"].(map[string]any)
	assertPolicyJSON(t, input["change"].(map[string]any)["before"], `{"kind":"string","value":"old"}`)
}

func TestLoadPlan_GivenNonEmptyMaskOnNullOrScalar_ShouldRejectAsInputInvalid(t *testing.T) {
	tests := []struct{ name, change string }{
		{"null non-empty object unknown mask", `{"actions":["update"],"before":{},"after":null,"after_unknown":{"x":false}}`},
		{"null non-empty array sensitive mask", `{"actions":["update"],"before":{},"after":null,"after_sensitive":[false]}`},
		{"string non-empty object sensitive mask", `{"actions":["update"],"before":{},"after":"x","after_sensitive":{"x":true}}`},
		{"string non-empty array unknown mask", `{"actions":["update"],"before":{},"after":"x","after_unknown":[true]}`},
		{"number non-empty object unknown mask", `{"actions":["update"],"before":{},"after":1,"after_unknown":{"x":false}}`},
		{"before scalar non-empty object mask", `{"actions":["update"],"before":"old","after":{},"before_sensitive":{"x":true}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := policyEvidenceJSON(t, policyPlan(policyResource("", tt.change)))
			if evidence["error_code"] != "input_invalid" {
				t.Errorf("error_code = %v, want input_invalid", evidence["error_code"])
			}
		})
	}
}

func TestLoadPlan_GivenStringIndexAndUnknownImport_ShouldRetainCompleteIdentity(t *testing.T) {
	rows := policyRows(t, policyEvidenceJSON(t, policyPlan(policyResource(`,"index":"worker-key"`, `{"actions":["update"],"importing":{"unknown":true},"before":{},"after":{}}`))))
	input := rows[0].(map[string]any)["input"].(map[string]any)
	assertPolicyJSON(t, input["resource"].(map[string]any)["index"], `{"kind":"string","value":"worker-key"}`)
	assertPolicyJSON(t, input["change"].(map[string]any)["importing"], `{"id":{"kind":"unknown"},"unknown":true}`)
}

func TestLoadPlan_GivenMalformedJSON_ShouldRejectAcquisition(t *testing.T) {
	for _, source := range []string{`{"format_version":"1.2",`, `{"format_version":"1.2","resource_changes":[}`, `{"format_version":"1.2"} {}`, `{"format_version":"1.2"} invalid`, `{"format_version":"9.0"}`} {
		if _, err := LoadPlan([]byte(source)); err == nil {
			t.Errorf("LoadPlan(%q) must reject malformed acquisition JSON", source)
		}
	}
}

func TestLoadPlan_GivenInvalidPolicyOnlyMetadata_ShouldPreserveDisplayChanges(t *testing.T) {
	source := policyPlan(policyResource("", `{"actions":["update"],"before":{"memory_size":128},"after":{"memory_size":256},"replace_paths":{}}`))
	summary, err := LoadPlan([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if summary.PolicyEvidence.ErrorCode != "input_invalid" || summary.ToUpdate != 1 || len(summary.Changes) != 1 {
		t.Fatalf("policy failure lost reviewed mutation: %#v", summary)
	}
	diffs := summary.Changes[0].AttributeDiffs
	if len(diffs) != 1 || diffs[0].Key != "memory_size" || diffs[0].OldValue != "128" || diffs[0].NewValue != "256" {
		t.Errorf("policy failure changed display evidence: %#v", diffs)
	}
}
