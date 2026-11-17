package risk

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/sdktest"
)

func reportAssessment(level sdk.RiskLevel) sdk.PolicyAssessment {
	return sdk.PolicyAssessment{
		Status: "complete", PlanID: "sha256:reviewed",
		Policy:       sdk.PolicyIdentity{Kind: "custom", ID: "custom/sha256:policy", Digest: "sha256:policy"},
		OverallLevel: &level,
		Resources: []sdk.PolicyResourceAssessment{
			{Address: "aws_instance.z", Actions: []string{"delete", "create"}, Status: "classified", Level: &level, Reasons: []sdk.PolicyReason{{Code: "replacement", Message: "Replaces this resource.", Paths: [][]any{{"after", "size"}}}}},
			{Address: "aws_instance.a", Actions: []string{"update"}, Status: "classified", Level: &level, Reasons: []sdk.PolicyReason{{Code: "default", Message: "Explicit project classification."}}},
			{Address: "output.name", Actions: []string{"update"}, Status: "not_assessed", Exclusion: "output_only"},
		},
	}
}

func newReportPlugin(t *testing.T, assessment sdk.PolicyAssessment) *Plugin {
	t.Helper()
	p, _, _ := newAssessedPlugin(t, lifecycleSummary("aws_instance.a"), assessment)
	return p
}

// configureAsHost applies settings to p and makes p's host answer as the App
// does for those settings: unavailable with the settings' code and hint.
func configureAsHost(t *testing.T, p *Plugin, settings map[string]interface{}) error {
	t.Helper()
	err := p.Configure(settings)
	if err != nil {
		h := sdktest.NewDeps(&sdktest.MockService{})
		h.AssessmentEvent = sdk.RiskAssessedEvent{Summary: p.summary, Assessment: sdk.PolicyAssessment{Status: sdk.AssessmentUnavailable, ErrorCode: sdk.ErrorConfigurationInvalid}, Hint: sdk.DiagnosticHint(err)}
		runLifecycleCmd(p, p.Init(h.Deps))
	}
	return err
}

func reportStdout(t *testing.T, p *Plugin) ([]byte, error) {
	t.Helper()
	emitter, ok := any(p).(sdk.StdoutEmitter)
	if !ok {
		t.Fatal("Risk must emit its completed assessment through Stdout")
	}
	return emitter.Stdout()
}

func TestReport_GivenCompleteAssessment_ShouldEmitDeterministicPlainText(t *testing.T) {
	p := newReportPlugin(t, reportAssessment(sdk.RiskLow))
	runLifecycleCmd(p, p.Activate())
	got, err := reportStdout(t, p)
	if err != nil {
		t.Fatal(err)
	}
	want := "Risk: low\nPolicy: custom/sha256:policy\nMutations: 2 eligible, 2 classified; 1 changes not assessed\nThreshold: off (disabled)\nupdate aws_instance.a: low\n  default: Explicit project classification.\ndelete/create aws_instance.z: low\n  replacement: Replaces this resource. [paths: [[\"after\",\"size\"]]]\nupdate output.name: not assessed (output_only)\n"
	if string(got) != want {
		t.Fatalf("Risk text = %q, want %q", got, want)
	}
}

func TestReport_GivenThreshold_ShouldCompareInclusivelyWithoutTreatingEmptyAsNone(t *testing.T) {
	for _, tc := range []struct {
		name, threshold, status, result string
		level                           sdk.RiskLevel
		exit                            int
	}{
		{"below", "high", "complete", "below", sdk.RiskMedium, 0},
		{"equal", "high", "complete", "met", sdk.RiskHigh, 3},
		{"above", "high", "complete", "met", sdk.RiskCritical, 3},
		{"medium", "medium", "complete", "met", sdk.RiskMedium, 3},
		{"critical", "critical", "complete", "met", sdk.RiskCritical, 3},
		{"explicit none", "none", "complete", "met", sdk.RiskNone, 3},
		{"off", "off", "complete", "disabled", sdk.RiskCritical, 0},
		{"empty", "none", "empty", "not_applicable", sdk.RiskNone, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := reportAssessment(tc.level)
			a.Status = sdk.AssessmentStatus(tc.status)
			if tc.status == "empty" {
				a.OverallLevel = nil
				a.Resources = a.Resources[2:]
			}
			p := newReportPlugin(t, a)
			if err := p.Configure(map[string]interface{}{"threshold": tc.threshold}); err != nil {
				t.Fatal(err)
			}
			runLifecycleCmd(p, p.Activate())
			if got := p.ExitCode(); got != tc.exit {
				t.Fatalf("threshold %s at %s exit = %d, want %d", tc.threshold, tc.level, got, tc.exit)
			}
			out, err := reportStdout(t, p)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), "Threshold: "+tc.threshold+" ("+tc.result+")\n") {
				t.Fatalf("threshold result missing from %q", out)
			}
			if tc.status == "empty" && !strings.HasPrefix(string(out), "No eligible mutations (1 changes not assessed).\n") {
				t.Fatalf("empty assessment became implicit none: %q", out)
			}
		})
	}
}

func TestReport_GivenEmptyAssessment_ShouldEmitRequiredJSONFieldsAndEmptyArrays(t *testing.T) {
	for _, threshold := range []string{"off", "high"} {
		t.Run(threshold, func(t *testing.T) {
			a := reportAssessment(sdk.RiskNone)
			a.Status, a.OverallLevel, a.Resources = "empty", nil, nil
			p := newReportPlugin(t, a)
			runLifecycleCmd(p, p.ActivateInput(Input{JSON: true, Threshold: threshold}))
			out, err := reportStdout(t, p)
			if err != nil {
				t.Fatal(err)
			}
			var report map[string]any
			if err := json.Unmarshal(out, &report); err != nil {
				t.Fatal(err)
			}
			if report["overall_level"] != nil || report["status"] != "empty" || !reflect.DeepEqual(report["resources"], []any{}) {
				t.Fatalf("empty JSON lost distinction from explicit none: %s", out)
			}
			comparison := report["threshold"].(map[string]any)
			if comparison["result"] != "not_applicable" || (threshold == "off" && comparison["level"] != nil) || (threshold == "high" && comparison["level"] != "high") {
				t.Fatalf("empty threshold = %v", comparison)
			}
			if p.ExitCode() != 0 || len(p.Stderr()) != 0 {
				t.Fatalf("empty assessment exit %d stderr %q", p.ExitCode(), p.Stderr())
			}
		})
	}
}

func TestReport_GivenNoAssessment_ShouldEmitNoReport(t *testing.T) {
	p, _ := newLifecyclePlugin(t, &sdktest.MockService{})
	for _, done := range []bool{false, true} {
		if done {
			p.Analyze(nil)
		}
		out, err := reportStdout(t, p)
		if err != nil || len(out) != 0 {
			t.Fatalf("unassessed plugin stdout %q error %v", out, err)
		}
	}
}

func TestReport_GivenGlobalFailure_ShouldPreferGlobalDiagnosticAndEmitNoJSON(t *testing.T) {
	a := reportAssessment(sdk.RiskHigh)
	a.Status, a.OverallLevel, a.ErrorCode = "unavailable", nil, "input_incomplete"
	a.Resources = []sdk.PolicyResourceAssessment{{Address: "aws_instance.bad\x1b[31m", Status: "unavailable", ErrorCode: "result_invalid"}}
	p := newReportPlugin(t, a)
	runLifecycleCmd(p, p.ActivateInput(Input{JSON: true, Threshold: "off"}))
	out, err := reportStdout(t, p)
	if err != nil || len(out) != 0 || p.ExitCode() != 1 || string(p.Stderr()) != "risk unavailable: input_incomplete\n" {
		t.Fatalf("global failure stdout %q stderr %q exit %d error %v", out, p.Stderr(), p.ExitCode(), err)
	}
}

func TestReport_GivenUntrustedResourceAddress_ShouldEscapeTerminalControlsInDiagnostic(t *testing.T) {
	a := reportAssessment(sdk.RiskHigh)
	a.Status, a.OverallLevel = "partial", nil
	a.Resources = append(a.Resources, sdk.PolicyResourceAssessment{Address: "aws_instance.bad\x1b[31m", Status: "unavailable", ErrorCode: "result_invalid"})
	p := newReportPlugin(t, a)
	runLifecycleCmd(p, p.Activate())
	if got := string(p.Stderr()); got != "risk unavailable: result_invalid for \"aws_instance.bad\\x1b[31m\"\n" {
		t.Fatalf("unsafe diagnostic = %q", got)
	}
}

func TestReport_GivenUntrustedResourceAddress_ShouldEscapeTerminalControlsInText(t *testing.T) {
	a := reportAssessment(sdk.RiskLow)
	a.Resources[0].Address = "aws_instance.bad\x1b[31m"
	p := newReportPlugin(t, a)
	runLifecycleCmd(p, p.Activate())
	out, err := reportStdout(t, p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(string(out), '\x1b') || !strings.Contains(string(out), `"aws_instance.bad\x1b[31m"`) {
		t.Fatalf("text report contains unescaped terminal metadata: %q", out)
	}
}

func TestReport_GivenPartialAssessment_ShouldEmitNoStdoutAndFirstFailureByAddress(t *testing.T) {
	a := reportAssessment(sdk.RiskHigh)
	a.Status, a.OverallLevel = "partial", nil
	a.Resources = append(a.Resources,
		sdk.PolicyResourceAssessment{Address: "aws_instance.z_failed", Status: "unavailable", ErrorCode: "evaluation_failed"},
		sdk.PolicyResourceAssessment{Address: "aws_instance.a_failed", Status: "unavailable", ErrorCode: "result_invalid"})
	p := newReportPlugin(t, a)
	runLifecycleCmd(p, p.Activate())
	out, _ := reportStdout(t, p)
	if len(out) != 0 || p.ExitCode() != 1 {
		t.Fatalf("partial assessment emitted %q with exit %d", out, p.ExitCode())
	}
	if got := string(p.Stderr()); got != "risk unavailable: result_invalid for aws_instance.a_failed\n" {
		t.Fatalf("failure diagnostic = %q", got)
	}
}

func TestConfigure_GivenInvalidRiskSettings_ShouldRejectConfiguration(t *testing.T) {
	for _, settings := range []map[string]interface{}{{"threshold": "urgent"}, {"threshold": true}, {"level": "high"}} {
		p := newReportPlugin(t, reportAssessment(sdk.RiskLow))
		if err := configureAsHost(t, p, settings); err == nil {
			t.Errorf("Configure(%v) accepted unsupported risk settings", settings)
		}
		runLifecycleCmd(p, p.Activate())
		if p.ExitCode() != 1 || !strings.HasPrefix(string(p.Stderr()), "risk unavailable: configuration_invalid") {
			t.Errorf("invalid settings did not remain unavailable: exit %d stderr %q", p.ExitCode(), p.Stderr())
		}
		if _, legacy := settings["level"]; legacy && !strings.Contains(string(p.Stderr()), "\n\nhint: risk setting level is unsupported; use threshold") {
			t.Errorf("legacy setting omitted migration hint: %q", p.Stderr())
		}
	}
}

func TestActivateInput_GivenLegacySettingAndInvalidOverride_ShouldRetainMigrationHint(t *testing.T) {
	p := newReportPlugin(t, reportAssessment(sdk.RiskLow))
	if err := configureAsHost(t, p, map[string]interface{}{"level": "high"}); err == nil {
		t.Fatal("legacy level must be rejected")
	}
	runLifecycleCmd(p, p.ActivateInput(Input{Threshold: "urgent", ThresholdSet: true}))
	want := "risk unavailable: configuration_invalid\n\nhint: risk setting level is unsupported; use threshold\n"
	if got := string(p.Stderr()); got != want {
		t.Fatalf("invalid override replaced actionable legacy diagnostic: got %q, want %q", got, want)
	}
	out, err := reportStdout(t, p)
	if err != nil || len(out) != 0 || p.ExitCode() != 1 {
		t.Fatalf("invalid settings emitted report %q, error %v, exit %d", out, err, p.ExitCode())
	}
}

func TestReport_GivenJSONInput_ShouldPreserveProvenanceContextAndStatusSpecificRows(t *testing.T) {
	p := newReportPlugin(t, reportAssessment(sdk.RiskLow))
	current := *p.GetCtx()
	current.Workspace = "production"
	p.GetCtx = func() *sdk.Context { return &current }
	project := "../project"
	runLifecycleCmd(p, p.ActivateInput(Input{JSON: true, Threshold: "high", ThresholdSet: true, Project: &project, Chdir: "members/api"}))
	// The report must retain the acquisition Context even if the host has since replaced it.
	next := current
	next.Workspace = "later"
	p.GetCtx = func() *sdk.Context { return &next }
	out, err := reportStdout(t, p)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("Risk JSON = %q: %v", out, err)
	}
	var want map[string]any
	const expected = `{
		"schema_version":"tfui.risk.report/v1", "input_schema_version":"tfui.risk.input/v1",
		"policy":{"kind":"custom","id":"custom/sha256:policy","digest":"sha256:policy"},
		"context":{"project":"../project","chdir":"members/api","workspace":"production"},
		"plan_id":"sha256:reviewed","status":"complete","overall_level":"low",
		"counts":{"eligible":2,"classified":2,"not_assessed":1},
		"levels":{"none":0,"low":2,"medium":0,"high":0,"critical":0},
		"threshold":{"level":"high","result":"below"},
		"resources":[
			{"address":"aws_instance.a","deposed":null,"actions":["update"],"status":"classified","level":"low","reasons":[{"code":"default","message":"Explicit project classification.","paths":[]}]},
			{"address":"aws_instance.z","deposed":null,"actions":["delete","create"],"status":"classified","level":"low","reasons":[{"code":"replacement","message":"Replaces this resource.","paths":[["after","size"]]}]},
			{"address":"output.name","deposed":null,"actions":["update"],"status":"not_assessed","exclusion":"output_only"}
		]}`
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Risk report = %s, want %s", out, expected)
	}
}

func TestActivateInput_GivenThresholdOverride_ShouldOverrideSettingsAndRejectExplicitEmpty(t *testing.T) {
	for _, tc := range []struct {
		name, override string
		set            bool
		exit           int
		diagnostic     string
	}{
		{"omitted uses settings", "", false, 3, "risk threshold met: low >= low\n"},
		{"off overrides settings", "off", true, 0, ""},
		{"higher overrides settings", "high", true, 0, ""},
		{"explicit empty invalid", "", true, 1, "risk unavailable: configuration_invalid\n"},
		{"invalid override", "urgent", true, 1, "risk unavailable: configuration_invalid\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := newReportPlugin(t, reportAssessment(sdk.RiskLow))
			if err := p.Configure(map[string]interface{}{"threshold": "low"}); err != nil {
				t.Fatal(err)
			}
			runLifecycleCmd(p, p.ActivateInput(Input{Threshold: tc.override, ThresholdSet: tc.set}))
			if p.ExitCode() != tc.exit || string(p.Stderr()) != tc.diagnostic {
				t.Fatalf("override exit %d stderr %q, want exit %d stderr %q", p.ExitCode(), p.Stderr(), tc.exit, tc.diagnostic)
			}
		})
	}
}
