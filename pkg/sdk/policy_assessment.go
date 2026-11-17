package sdk

// PolicyAssessment contains classifications from one selected policy and reviewed plan.
// Only a complete assessment has an overall level; cancelled results cannot be published.
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
