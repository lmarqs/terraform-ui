package riskpolicy

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var reasonCode = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func validateResult(value any) (sdk.RiskLevel, []sdk.PolicyReason, string) {
	invalid := func() (sdk.RiskLevel, []sdk.PolicyReason, string) { return 0, nil, "result_invalid" }
	object, ok := value.(map[string]any)
	if !ok || len(object) != 2 {
		return invalid()
	}
	levelName, ok := object["level"].(string)
	if !ok {
		return invalid()
	}
	var level sdk.RiskLevel
	switch levelName {
	case "none":
		level = sdk.RiskNone
	case "low":
		level = sdk.RiskLow
	case "medium":
		level = sdk.RiskMedium
	case "high":
		level = sdk.RiskHigh
	case "critical":
		level = sdk.RiskCritical
	default:
		return invalid()
	}
	rawReasons, ok := object["reasons"].([]any)
	if !ok || len(rawReasons) == 0 || len(rawReasons) > 16 {
		return invalid()
	}
	reasons := make([]sdk.PolicyReason, 0, len(rawReasons))
	for _, raw := range rawReasons {
		reason, ok := raw.(map[string]any)
		if !ok || len(reason) != 3 {
			return invalid()
		}
		code, ok := reason["code"].(string)
		if !ok || !reasonCode.MatchString(code) {
			return invalid()
		}
		message, ok := reason["message"].(string)
		if !ok || strings.TrimSpace(message) == "" || utf8.RuneCountInString(message) > 512 || strings.ContainsFunc(message, unicode.IsControl) {
			return invalid()
		}
		paths, ok := reason["paths"].([]any)
		if !ok || len(paths) > 16 {
			return invalid()
		}
		// Path traversal validation is supplied with the resource-aware validator.
		// Until then, fail closed on path-bearing explanations.
		if len(paths) != 0 {
			return invalid()
		}
		reasons = append(reasons, sdk.PolicyReason{Code: code, Message: message, Paths: [][]any{}})
	}
	return level, reasons, ""
}
