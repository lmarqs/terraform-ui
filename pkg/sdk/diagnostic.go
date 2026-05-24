package sdk

// DiagnosticSeverity classifies validation diagnostics.
type DiagnosticSeverity string

const (
	SeverityError   DiagnosticSeverity = "error"
	SeverityWarning DiagnosticSeverity = "warning"
)

func (s DiagnosticSeverity) IsError() bool   { return s == SeverityError }
func (s DiagnosticSeverity) IsWarning() bool { return s == SeverityWarning }

// Diagnostic represents a terraform validation diagnostic (error or warning).
type Diagnostic struct {
	Severity DiagnosticSeverity
	Summary  string
	Detail   string
	File     string
	Line     int
}
