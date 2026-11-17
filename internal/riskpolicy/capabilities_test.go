package riskpolicy

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestCapabilities_GivenPublishedProfile_ShouldMatchFrozenDeclarationsAndFingerprint(t *testing.T) {
	frozen, err := os.ReadFile("testdata/capabilities.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]any
	if err := json.Unmarshal(frozen, &expected); err != nil {
		t.Fatal(err)
	}
	var allowed []any
	for _, item := range expected["builtins"].([]any) {
		name := item.(map[string]any)["name"]
		if name != "plus" && name != "minus" && name != "mul" && name != "div" {
			allowed = append(allowed, item)
		}
	}
	expected["builtins"] = allowed
	profile := capabilities()
	if profile.AllowNet == nil || len(profile.AllowNet) != 0 {
		t.Fatalf("network permissions = %v; want explicit empty allowlist", profile.AllowNet)
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		t.Fatal(err)
	}
	var actual map[string]any
	if err := json.Unmarshal(encoded, &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("embedded capability profile differs from accepted filtered artifact")
	}
	var canonical bytes.Buffer
	encoder := json.NewEncoder(&canonical)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(expected); err != nil {
		t.Fatal(err)
	}
	wantDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(bytes.TrimSuffix(canonical.Bytes(), []byte("\n"))))
	if evaluatorVersion != "v1.19.1" || capabilityDigest != wantDigest {
		t.Fatalf("runtime identity = %s / %s; want v1.19.1 / %s", evaluatorVersion, capabilityDigest, wantDigest)
	}
}
