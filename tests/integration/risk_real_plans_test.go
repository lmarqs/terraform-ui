//go:build integration

package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

type riskReportRow struct {
	Address   string  `json:"address"`
	Deposed   *string `json:"deposed"`
	Status    string  `json:"status"`
	Level     string  `json:"level"`
	Exclusion string  `json:"exclusion"`
	Reasons   []struct {
		Code string `json:"code"`
	} `json:"reasons"`
}

func TestRisk_RealTerraformPlans_ShouldReportExpectedRowsWithoutFailing(t *testing.T) {
	deposed := "00000001"
	tests := []struct {
		fixture string
		want    []riskReportRow
	}{
		{"risk-delete", []riskReportRow{{Address: "terraform_data.doomed", Status: "classified", Level: "high", Reasons: reasonCodes("delete-after-absent")}}},
		{"risk-forget", []riskReportRow{{Address: "terraform_data.kept", Status: "not_assessed", Exclusion: "forget"}}},
		{"risk-import", []riskReportRow{{Address: "terraform_data.adopted", Status: "classified", Level: "low", Reasons: reasonCodes("import-id-hidden")}}},
		{"risk-deposed", []riskReportRow{
			{Address: "aws_instance.web", Status: "classified", Level: "low", Reasons: reasonCodes("current-object")},
			{Address: "aws_instance.web", Deposed: &deposed, Status: "classified", Level: "high", Reasons: reasonCodes("deposed-object")},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.fixture, func(t *testing.T) {
			dir := fixtureDir(tc.fixture)

			stdout, stderr, err := runTfui("risk", "-ci", "-json", "-plan", filepath.Join(dir, "plan.json"), "-project", dir)

			if err != nil && !isExitCode(err, 3) {
				t.Fatalf("tfui risk failed (want exit 0 or 3): %v\nstderr: %s\nstdout: %s", err, stderr, stdout)
			}
			var report struct {
				Resources []riskReportRow `json:"resources"`
			}
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				t.Fatalf("report is not JSON: %v\n%s", err, stdout)
			}
			if len(report.Resources) != len(tc.want) {
				t.Fatalf("rows = %d, want %d: %s", len(report.Resources), len(tc.want), stdout)
			}
			for i, want := range tc.want {
				assertRiskRow(t, i, report.Resources[i], want)
			}
			if strings.Contains(stdout, "known-import-id") {
				t.Errorf("import ID leaked into the report: %s", stdout)
			}
		})
	}
}

func reasonCodes(codes ...string) []struct {
	Code string `json:"code"`
} {
	reasons := make([]struct {
		Code string `json:"code"`
	}, len(codes))
	for i, code := range codes {
		reasons[i].Code = code
	}
	return reasons
}

func assertRiskRow(t *testing.T, i int, got, want riskReportRow) {
	t.Helper()
	if got.Address != want.Address || got.Status != want.Status || got.Level != want.Level || got.Exclusion != want.Exclusion {
		t.Errorf("row %d = %+v, want address %q status %q level %q exclusion %q", i, got, want.Address, want.Status, want.Level, want.Exclusion)
	}
	if (got.Deposed == nil) != (want.Deposed == nil) || (got.Deposed != nil && *got.Deposed != *want.Deposed) {
		t.Errorf("row %d deposed = %v, want %v", i, got.Deposed, want.Deposed)
	}
	if len(got.Reasons) != len(want.Reasons) {
		t.Fatalf("row %d reasons = %+v, want %+v", i, got.Reasons, want.Reasons)
	}
	for j := range want.Reasons {
		if got.Reasons[j].Code != want.Reasons[j].Code {
			t.Errorf("row %d reason %d = %q, want %q", i, j, got.Reasons[j].Code, want.Reasons[j].Code)
		}
	}
}
