package sdk

import "errors"

// DiagnosticError is a fixed failure code with a hint on how to resolve it.
type DiagnosticError struct {
	Code ErrorCode
	Hint string
}

// Error renders the code, followed by the hint when there is one.
func (e DiagnosticError) Error() string {
	if e.Hint == "" {
		return string(e.Code)
	}
	return string(e.Code) + "\n\nhint: " + e.Hint
}

// Unwrap exposes the code, so errors.Is matches it.
func (e DiagnosticError) Unwrap() error { return e.Code }

// DiagnosticHint returns the hint err carries, or "" when it carries none.
func DiagnosticHint(err error) string {
	var diagnostic DiagnosticError
	if errors.As(err, &diagnostic) {
		return diagnostic.Hint
	}
	return ""
}

// RiskSettingsError validates comparison settings without preventing Plan review.
func RiskSettingsError(cfg map[string]interface{}) error {
	if _, exists := cfg["level"]; exists {
		return DiagnosticError{Code: ErrorConfigurationInvalid, Hint: "risk setting level is unsupported; use threshold"}
	}
	if value, exists := cfg["threshold"]; exists {
		name, _ := value.(string)
		if _, valid := ParseRiskLevel(name); name != "off" && !valid {
			return ErrorConfigurationInvalid
		}
	}
	return nil
}
