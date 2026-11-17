package risk

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

// Input carries command output options independently of policy evaluation.
type Input struct {
	JSON      bool
	Threshold *string
	Project   *string
	Chdir     string
}

// ActivateInput starts Risk with command-specific output and comparison options.
func (e *Plugin) ActivateInput(input Input) tea.Cmd {
	e.input = input
	return e.Activate()
}

func (e *Plugin) configureReport(cfg map[string]interface{}) error {
	e.threshold, e.settingsInvalid = "off", false
	if err := sdk.RiskSettingsError(cfg); err != nil {
		e.settingsInvalid = true
		return err
	}
	if value, exists := cfg["threshold"]; exists {
		e.threshold = value.(string)
	}
	return nil
}

func (e *Plugin) selectedThreshold() string {
	if e.input.Threshold != nil {
		return *e.input.Threshold
	}
	if e.threshold == "" {
		return "off"
	}
	return e.threshold
}

func (e *Plugin) thresholdResult() string {
	if e.assessment.Status == sdk.AssessmentEmpty {
		return "not_applicable"
	}
	threshold := e.selectedThreshold()
	if threshold == "off" {
		return "disabled"
	}
	level, _ := sdk.ParseRiskLevel(threshold)
	if *e.assessment.OverallLevel >= level {
		return "met"
	}
	return "below"
}

type reportContext struct {
	Project   *string `json:"project"`
	Chdir     string  `json:"chdir"`
	Workspace string  `json:"workspace"`
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
	r := riskReport{SchemaVersion: "tfui.risk.report/v1", InputSchemaVersion: "tfui.risk.input/v1", Context: e.reportContext, PlanID: a.PlanID, Status: a.Status, Levels: map[string]int{"none": 0, "low": 0, "medium": 0, "high": 0, "critical": 0}, Resources: []reportResource{}}
	r.Policy.Kind, r.Policy.ID, r.Policy.Digest = a.Policy.Kind, a.Policy.ID, a.Policy.Digest
	if a.OverallLevel != nil {
		level := a.OverallLevel.String()
		r.OverallLevel = &level
	}
	threshold := e.selectedThreshold()
	if threshold != "off" {
		r.Threshold.Level = &threshold
	}
	r.Threshold.Result = e.thresholdResult()
	for _, row := range a.Resources {
		resource := reportResource{Address: row.Address, Actions: row.Actions, Status: row.Status, Exclusion: row.Exclusion}
		if row.Deposed != "" {
			resource.Deposed = &row.Deposed
		}
		if row.Status == sdk.ResourceClassified {
			r.Counts.Eligible++
			r.Counts.Classified++
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
		} else {
			r.Counts.NotAssessed++
		}
		r.Resources = append(r.Resources, resource)
	}
	slices.SortFunc(r.Resources, func(a, b reportResource) int {
		return cmp.Or(cmp.Compare(a.Address, b.Address), cmp.Compare(deposedKey(a), deposedKey(b)))
	})
	return r
}

// Stdout emits only a complete or empty versioned assessment report.
func (e *Plugin) Stdout() ([]byte, error) {
	if e.status != sdk.StatusDone || e.assessment == nil {
		return nil, nil
	}
	r := e.report()
	if e.input.JSON {
		data, err := json.MarshalIndent(r, "", "  ")
		return append(data, '\n'), err
	}
	var out strings.Builder
	if r.Status == sdk.AssessmentEmpty {
		fmt.Fprintf(&out, "No eligible mutations (%d changes not assessed).\n", r.Counts.NotAssessed)
	} else {
		fmt.Fprintf(&out, "Risk: %s\n", *r.OverallLevel)
	}
	fmt.Fprintf(&out, "Policy: %s\nMutations: %d eligible, %d classified; %d changes not assessed\nThreshold: %s (%s)\n", r.Policy.ID, r.Counts.Eligible, r.Counts.Classified, r.Counts.NotAssessed, e.selectedThreshold(), r.Threshold.Result)
	for _, row := range r.Resources {
		if row.Status != sdk.ResourceClassified {
			continue
		}
		fmt.Fprintf(&out, "%s %s: %s\n", strings.Join(row.Actions, "/"), instanceLabel(row.Address, deposedKey(row)), *row.Level)
		for _, reason := range *row.Reasons {
			fmt.Fprintf(&out, "  %s: %s", reason.Code, reason.Message)
			if len(reason.Paths) > 0 {
				// Assessment paths contain only validated JSON strings and integer indexes.
				paths, _ := json.Marshal(reason.Paths)
				fmt.Fprintf(&out, " [paths: %s]", paths)
			}
			out.WriteByte('\n')
		}
	}
	for _, row := range r.Resources {
		if row.Status == sdk.ResourceNotAssessed {
			fmt.Fprintf(&out, "%s %s: not assessed (%s)\n", strings.Join(row.Actions, "/"), instanceLabel(row.Address, deposedKey(row)), row.Exclusion)
		}
	}
	return []byte(out.String()), nil
}

func deposedKey(row reportResource) string {
	if row.Deposed == nil {
		return ""
	}
	return *row.Deposed
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
	if e.planError != "" {
		return []byte(e.planError + "\n")
	}
	if e.riskDiagnostic != "" {
		diagnostic := e.riskDiagnostic
		if e.assessment != nil && e.assessment.ErrorCode == "" {
			rows := append([]sdk.PolicyResourceAssessment(nil), e.assessment.Resources...)
			slices.SortFunc(rows, func(a, b sdk.PolicyResourceAssessment) int {
				return cmp.Or(cmp.Compare(a.Address, b.Address), cmp.Compare(a.Deposed, b.Deposed))
			})
			for _, row := range rows {
				if row.ErrorCode != "" {
					diagnostic = string(row.ErrorCode) + " for " + instanceLabel(row.Address, row.Deposed)
					break
				}
			}
		}
		return []byte("risk unavailable: " + diagnostic + "\n")
	}
	if e.status == sdk.StatusDone && e.assessment != nil && e.thresholdResult() == "met" {
		return []byte(fmt.Sprintf("risk threshold met: %s >= %s\n", e.assessment.OverallLevel.String(), e.selectedThreshold()))
	}
	return nil
}

func (e *Plugin) ExitCode() int {
	if e.status == sdk.StatusError {
		return 1
	}
	if e.status == sdk.StatusDone && e.assessment != nil && e.thresholdResult() == "met" {
		return 3
	}
	return 0
}
