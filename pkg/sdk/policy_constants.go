package sdk

// AssessmentStatus is the overall state of one policy assessment.
type AssessmentStatus string

const (
	AssessmentComplete    AssessmentStatus = "complete"
	AssessmentEmpty       AssessmentStatus = "empty"
	AssessmentPartial     AssessmentStatus = "partial"
	AssessmentUnavailable AssessmentStatus = "unavailable"
	AssessmentCancelled   AssessmentStatus = "cancelled"
)

// ResourceStatus is the state of one resource within an assessment.
type ResourceStatus string

const (
	ResourceClassified  ResourceStatus = "classified"
	ResourceNotAssessed ResourceStatus = "not_assessed"
	ResourceUnavailable ResourceStatus = "unavailable"
)

// Exclusion names why a resource change is not eligible for policy evaluation.
type Exclusion string

const (
	ExclusionRead       Exclusion = "read"
	ExclusionNoOp       Exclusion = "no_op"
	ExclusionForget     Exclusion = "forget"
	ExclusionPureImport Exclusion = "pure_import"
	ExclusionPureMove   Exclusion = "pure_move"
	ExclusionOutputOnly Exclusion = "output_only"
)

// ErrorCode is a stable policy or configuration failure code. It implements error so
// producers return the constants as sentinels and callers recover them with errors.As or errors.Is.
type ErrorCode string

func (c ErrorCode) Error() string { return string(c) }

const (
	ErrorConfigurationInvalid ErrorCode = "configuration_invalid"
	ErrorSourceUnreadable     ErrorCode = "source_unreadable"
	ErrorSourceInvalid        ErrorCode = "source_invalid"
	ErrorSourceChanged        ErrorCode = "source_changed"
	ErrorInputIncomplete      ErrorCode = "input_incomplete"
	ErrorInputInvalid         ErrorCode = "input_invalid"
	ErrorCompilationFailed    ErrorCode = "compilation_failed"
	ErrorResultUndefined      ErrorCode = "result_undefined"
	ErrorResultInvalid        ErrorCode = "result_invalid"
	ErrorResultCardinality    ErrorCode = "result_cardinality"
	ErrorEvaluationConflict   ErrorCode = "evaluation_conflict"
	ErrorEvaluationFailed     ErrorCode = "evaluation_failed"
	ErrorCancelled            ErrorCode = "cancelled"
)

// PolicyValueKind tags a policy input value with its JSON-like or uncertain kind.
type PolicyValueKind string

const (
	KindAbsent           PolicyValueKind = "absent"
	KindNull             PolicyValueKind = "null"
	KindString           PolicyValueKind = "string"
	KindBoolean          PolicyValueKind = "boolean"
	KindNumber           PolicyValueKind = "number"
	KindArray            PolicyValueKind = "array"
	KindObject           PolicyValueKind = "object"
	KindUnknown          PolicyValueKind = "unknown"
	KindSensitive        PolicyValueKind = "sensitive"
	KindSensitiveUnknown PolicyValueKind = "sensitive_unknown"
)
