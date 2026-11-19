package sdk

import "slices"

// PolicyAssessment contains classifications from one selected policy and reviewed plan.
// Only a complete assessment has an overall level. The host may dispatch the
// result of a cancelled request; the requester ignores it by AssessmentID.
type PolicyAssessment struct {
	Status       AssessmentStatus
	PlanID       string
	Policy       PolicyIdentity
	OverallLevel *RiskLevel
	Resources    []PolicyResourceAssessment
	ErrorCode    ErrorCode
}

// PolicyIdentity identifies immutable policy bytes and the evaluation profile.
type PolicyIdentity struct {
	Kind             string
	ID               string
	Digest           string
	EvaluatorVersion string
	CapabilityDigest string
}

// PolicyResourceAssessment retains exclusions and failures without assigning a level.
type PolicyResourceAssessment struct {
	Address   string
	Deposed   string
	Actions   []string
	Status    ResourceStatus
	Level     *RiskLevel
	Reasons   []PolicyReason
	Exclusion Exclusion
	ErrorCode ErrorCode
}

// PolicyReason is a validated author explanation referring to tagged input values.
type PolicyReason struct {
	Code    string
	Message string
	Paths   [][]any
}

// Usable reports whether the assessment classified every eligible change, or found none to classify.
func (a PolicyAssessment) Usable() bool {
	return a.Status == AssessmentComplete || a.Status == AssessmentEmpty
}

// InstanceKey identifies one resource instance object: the current object has an empty Deposed key.
type InstanceKey struct {
	Address string
	Deposed string
}

// AssessmentRows indexes an assessment's rows by instance object.
type AssessmentRows map[InstanceKey]PolicyResourceAssessment

// Rows indexes the assessment's rows, so per-change lookups stay constant time on large plans.
func (a PolicyAssessment) Rows() AssessmentRows {
	rows := make(AssessmentRows, len(a.Resources))
	for _, row := range a.Resources {
		rows[InstanceKey{Address: row.Address, Deposed: row.Deposed}] = row
	}
	return rows
}

// ApplyAssessment returns a copy of summary whose changes carry the level of their classified row in rows.
// Nil rows keep each change's existing Risk; otherwise changes without a classified level get RiskNone.
func ApplyAssessment(summary *PlanSummary, rows AssessmentRows) *PlanSummary {
	if summary == nil {
		return nil
	}
	result := *summary
	result.Changes = slices.Clone(summary.Changes)
	if rows == nil {
		return &result
	}
	for i := range result.Changes {
		change := &result.Changes[i]
		change.Risk = RiskNone
		row, ok := rows[change.Resource.InstanceKey()]
		if ok && row.Status == ResourceClassified && row.Level != nil {
			change.Risk = *row.Level
		}
	}
	return &result
}
