package exec

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/terraform"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
	tfuirisk "github.com/lmarqs/terraform-ui/plugins/risk"
)

func TestPlan_GivenLiveAcquisition_ShouldRetainExactShowEvidenceWithoutReplanning(t *testing.T) {
	dir := t.TempDir()
	source := `{"format_version":"1.2","resource_changes":[{"address":"aws_lambda_function.worker","mode":"managed","type":"aws_lambda_function","name":"worker","provider_name":"registry.terraform.io/hashicorp/aws","change":{"actions":["update"],"before":{"n":9007199254740993},"after":{"n":1e3}}}]}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "show.json"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "terraform")
	script := `#!/bin/sh
printf '%s\n' "$*" >> calls
case "$1" in
version) printf '%s\n' '{"terraform_version":"1.14.0","platform":"linux_amd64","provider_selections":{}}' ;;
plan) exit 2 ;;
show) cat show.json ;;
*) exit 1 ;;
esac
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	svc := NewExecService(dir, binary, nil)
	live, err := svc.Plan(context.Background(), sdk.PlanOptions{PlanFile: filepath.Join(dir, "reviewed.tfplan")})
	if err != nil {
		t.Fatalf("live Plan: %v", err)
	}
	harness := sdktest.NewDeps(svc)
	risk := tfuirisk.New(svc).(*tfuirisk.Plugin)
	if cmd := risk.Init(harness.Deps); cmd != nil {
		cmd()
	}
	risk.Analyze(live)
	if !risk.Ready() || !strings.Contains(risk.View(100, 30), "aws_lambda_function.worker") {
		t.Fatalf("Risk did not review the acquired plan: %q", risk.View(100, 30))
	}
	field := reflect.ValueOf(live).Elem().FieldByName("PolicyEvidence")
	if !field.IsValid() {
		t.Fatal("live reviewed plan does not retain policy evidence from its original show JSON")
	}
	data, err := json.Marshal(field.Interface())
	if err != nil {
		t.Fatal(err)
	}
	var evidence map[string]any
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence["plan_id"] != fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(source))) {
		t.Errorf("plan identity must hash original show bytes: %#v", evidence)
	}
	seeded, err := terraform.LoadPlan([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	seededField := reflect.ValueOf(seeded).Elem().FieldByName("PolicyEvidence")
	if !reflect.DeepEqual(field.Interface(), seededField.Interface()) {
		t.Error("live and seeded evidence differ for the same reviewed JSON")
	}
	calls, err := os.ReadFile(filepath.Join(dir, "calls"))
	if err != nil {
		t.Fatal(err)
	}
	var plans, shows int
	for _, call := range strings.Split(strings.TrimSpace(string(calls)), "\n") {
		if strings.HasPrefix(call, "plan ") {
			plans++
		}
		if strings.HasPrefix(call, "show ") {
			shows++
			if !strings.Contains(call, "-json") {
				t.Errorf("show must preserve JSON evidence: %s", call)
			}
		}
	}
	if plans != 1 || shows != 1 {
		t.Errorf("plan/show calls = %d/%d, want one acquisition each; calls: %s", plans, shows, calls)
	}
}
