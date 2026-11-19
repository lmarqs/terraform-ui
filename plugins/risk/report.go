package risk

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Input carries command output options independently of policy evaluation.
type Input struct {
	JSON      bool
	Threshold *string
	Project   *string
	Chdir     string
}

func (e *Plugin) configureReport(cfg map[string]interface{}) error {
	e.configured, e.settingsInvalid = nil, false
	if err := sdk.RiskSettingsError(cfg); err != nil {
		e.settingsInvalid = true
		return err
	}
	// RiskSettingsError accepts only a valid threshold name.
	if name, ok := cfg["threshold"].(string); ok {
		e.configured, _ = parseThreshold(name)
	}
	return nil
}

// parseThreshold reads a threshold name; off, a nil level, disables the comparison.
func parseThreshold(name string) (*sdk.RiskLevel, bool) {
	if name == "off" {
		return nil, true
	}
	level, valid := sdk.ParseRiskLevel(name)
	return &level, valid
}

func thresholdName(threshold *sdk.RiskLevel) string {
	if threshold == nil {
		return "off"
	}
	return threshold.String()
}

// thresholdResult is how a usable assessment's overall level compares with the threshold.
type thresholdResult int

const (
	thresholdNotApplicable thresholdResult = iota
	thresholdDisabled
	thresholdBelow
	thresholdMet
)

// String is the result's v1 report value.
func (r thresholdResult) String() string {
	return [...]string{"not_applicable", "disabled", "below", "met"}[r]
}

func compareThreshold(a sdk.PolicyAssessment, threshold *sdk.RiskLevel) thresholdResult {
	switch {
	case a.Status == sdk.AssessmentEmpty:
		return thresholdNotApplicable
	case threshold == nil:
		return thresholdDisabled
	case *a.OverallLevel >= *threshold:
		return thresholdMet
	}
	return thresholdBelow
}

// sortedRows returns the assessment's rows ordered by instance object.
func sortedRows(a sdk.PolicyAssessment) []sdk.PolicyResourceAssessment {
	rows := slices.Clone(a.Resources)
	slices.SortFunc(rows, func(a, b sdk.PolicyResourceAssessment) int {
		return sdk.InstanceKey{Address: a.Address, Deposed: a.Deposed}.Compare(sdk.InstanceKey{Address: b.Address, Deposed: b.Deposed})
	})
	return rows
}

// failureDiagnostic names why an assessment is unusable: its global code with
// the host's hint, or else its first failed row by address.
func failureDiagnostic(a sdk.PolicyAssessment, hint string) string {
	if a.ErrorCode != "" {
		return sdk.DiagnosticError{Code: a.ErrorCode, Hint: hint}.Error()
	}
	for _, row := range sortedRows(a) {
		if row.ErrorCode != "" {
			return string(row.ErrorCode) + " for " + instanceLabel(row.Address, row.Deposed)
		}
	}
	return ""
}

type reportContext struct {
	Project   *string `json:"project"`
	Chdir     string  `json:"chdir"`
	Workspace string  `json:"workspace"`
}

// reportedContext is the Context the assessment was requested in, with the
// command's chdir in place of the selected one when it gave one.
func (e *Plugin) reportedContext() reportContext {
	c := reportContext{Project: e.input.Project, Chdir: e.input.Chdir, Workspace: e.requested.Workspace.String()}
	if c.Chdir == "" {
		c.Chdir = e.requested.Chdir.String()
	}
	if c.Chdir == "" {
		c.Chdir = "."
	}
	return c
}

type reportReason struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Paths   [][]any `json:"paths"`
}

type reportResource struct {
	Address   string             `json:"address"`
	Deposed   *string            `json:"deposed"`
	Actions   []string           `json:"actions"`
	Status    sdk.ResourceStatus `json:"status"`
	Level     *string            `json:"level,omitempty"`
	Reasons   *[]reportReason    `json:"reasons,omitempty"`
	Exclusion sdk.Exclusion      `json:"exclusion,omitempty"`
}

type riskReport struct {
	SchemaVersion      string `json:"schema_version"`
	InputSchemaVersion string `json:"input_schema_version"`
	Policy             struct {
		Kind   string `json:"kind"`
		ID     string `json:"id"`
		Digest string `json:"digest"`
	} `json:"policy"`
	Context      reportContext        `json:"context"`
	PlanID       string               `json:"plan_id"`
	Status       sdk.AssessmentStatus `json:"status"`
	OverallLevel *string              `json:"overall_level"`
	Counts       struct {
		Eligible    int `json:"eligible"`
		Classified  int `json:"classified"`
		NotAssessed int `json:"not_assessed"`
	} `json:"counts"`
	Levels    map[string]int `json:"levels"`
	Threshold struct {
		Level  *string `json:"level"`
		Result string  `json:"result"`
	} `json:"threshold"`
	Resources []reportResource `json:"resources"`
}

func (e *Plugin) report() riskReport {
	a := e.assessment
	r := riskReport{SchemaVersion: "tfui.risk.report/v1", InputSchemaVersion: "tfui.risk.input/v1", Context: e.reportedContext(), PlanID: a.PlanID, Status: a.Status, Levels: map[string]int{"none": 0, "low": 0, "medium": 0, "high": 0, "critical": 0}, Resources: []reportResource{}}
	r.Policy.Kind, r.Policy.ID, r.Policy.Digest = a.Policy.Kind, a.Policy.ID, a.Policy.Digest
	if a.OverallLevel != nil {
		level := a.OverallLevel.String()
		r.OverallLevel = &level
	}
	if e.threshold != nil {
		level := e.threshold.String()
		r.Threshold.Level = &level
	}
	r.Threshold.Result = compareThreshold(*a, e.threshold).String()
	counts := countRows(a)
	r.Counts.Eligible, r.Counts.Classified, r.Counts.NotAssessed = counts.classified, counts.classified, counts.notAssessed
	for _, row := range sortedRows(*a) {
		resource := reportResource{Address: row.Address, Actions: row.Actions, Status: row.Status, Exclusion: row.Exclusion}
		if row.Deposed != "" {
			resource.Deposed = &row.Deposed
		}
		if row.Status == sdk.ResourceClassified {
			level := row.Level.String()
			resource.Level = &level
			r.Levels[level]++
			reasons := []reportReason{}
			for _, reason := range row.Reasons {
				paths := reason.Paths
				if paths == nil {
					paths = [][]any{}
				}
				reasons = append(reasons, reportReason{Code: reason.Code, Message: reason.Message, Paths: paths})
			}
			resource.Reasons = &reasons
		}
		r.Resources = append(r.Resources, resource)
	}
	return r
}

// Stdout emits only a complete or empty versioned assessment report.
func (e *Plugin) Stdout() ([]byte, error) {
	if e.outcome.kind != outcomeDone {
		return nil, nil
	}
	if e.input.JSON {
		data, err := json.MarshalIndent(e.report(), "", "  ")
		return append(data, '\n'), err
	}
	a := e.assessment
	counts := countRows(a)
	var out strings.Builder
	if a.Status == sdk.AssessmentEmpty {
		fmt.Fprintf(&out, "No eligible mutations (%d changes not assessed).\n", counts.notAssessed)
	} else {
		fmt.Fprintf(&out, "Risk: %s\n", a.OverallLevel)
	}
	fmt.Fprintf(&out, "Policy: %s\nMutations: %d eligible, %d classified; %d changes not assessed\nThreshold: %s (%s)\n", a.Policy.ID, counts.classified, counts.classified, counts.notAssessed, thresholdName(e.threshold), compareThreshold(*a, e.threshold))
	rows := sortedRows(*a)
	for _, row := range rows {
		if row.Status != sdk.ResourceClassified {
			continue
		}
		fmt.Fprintf(&out, "%s %s: %s\n", strings.Join(row.Actions, "/"), instanceLabel(row.Address, row.Deposed), row.Level)
		for _, reason := range row.Reasons {
			fmt.Fprintf(&out, "  %s: %s", reason.Code, reason.Message)
			if len(reason.Paths) > 0 {
				// Assessment paths contain only validated JSON strings and integer indexes.
				paths, _ := json.Marshal(reason.Paths)
				fmt.Fprintf(&out, " [paths: %s]", paths)
			}
			out.WriteByte('\n')
		}
	}
	for _, row := range rows {
		if row.Status == sdk.ResourceNotAssessed {
			fmt.Fprintf(&out, "%s %s: not assessed (%s)\n", strings.Join(row.Actions, "/"), instanceLabel(row.Address, row.Deposed), row.Exclusion)
		}
	}
	return []byte(out.String()), nil
}

func instanceLabel(address, deposed string) string {
	label := escapedAddress(address)
	if deposed != "" {
		label += " (deposed " + escapedAddress(deposed) + ")"
	}
	return label
}

func escapedAddress(address string) string {
	if strings.ContainsFunc(address, unicode.IsControl) {
		return strconv.QuoteToASCII(address)
	}
	return address
}

func (e *Plugin) Stderr() []byte {
	if e.thresholdMet() {
		return fmt.Appendf(nil, "risk threshold met: %s >= %s\n", e.assessment.OverallLevel, thresholdName(e.threshold))
	}
	if e.outcome.diagnostic == "" {
		return nil
	}
	return []byte(e.outcome.diagnostic + "\n")
}

func (e *Plugin) ExitCode() int {
	switch e.outcome.kind {
	case outcomeInvalidThreshold, outcomePlanFailed, outcomeUnusable:
		return 1
	case outcomeDone:
		if e.thresholdMet() {
			return 3
		}
	}
	return 0
}

// thresholdMet reports a done run whose overall level reached the threshold.
func (e *Plugin) thresholdMet() bool {
	return e.outcome.kind == outcomeDone && compareThreshold(*e.assessment, e.threshold) == thresholdMet
}
