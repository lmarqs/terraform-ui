package riskpolicy

import (
	"encoding/json"
	"math/big"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

var reasonCode = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func validateResult(value any, input *sdk.PolicyInput) (sdk.RiskLevel, []sdk.PolicyReason, sdk.ErrorCode) {
	invalid := func() (sdk.RiskLevel, []sdk.PolicyReason, sdk.ErrorCode) { return 0, nil, sdk.ErrorResultInvalid }
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
		validated := make([][]any, 0, len(paths))
		for _, rawPath := range paths {
			path, ok := rawPath.([]any)
			if !ok || !validReasonPath(path, input) {
				return invalid()
			}
			validated = append(validated, path)
		}
		reasons = append(reasons, sdk.PolicyReason{Code: code, Message: message, Paths: validated})
	}
	return level, reasons, ""
}

func validReasonPath(path []any, input *sdk.PolicyInput) bool {
	if len(path) == 0 {
		return false
	}
	var node sdk.PolicyValue
	switch path[0] {
	case "before":
		node = input.Change.Before
	case "after":
		node = input.Change.After
	default:
		return false
	}
	for _, element := range path[1:] {
		switch node.Kind {
		case sdk.KindObject:
			key, ok := element.(string)
			if !ok {
				return false
			}
			node, ok = (*node.Entries)[key]
			if !ok {
				return false
			}
		case sdk.KindArray:
			index, ok := element.(json.Number)
			if !ok {
				return false
			}
			number, ok := new(big.Rat).SetString(string(index))
			if !ok || number.Sign() < 0 || !number.IsInt() || !number.Num().IsInt64() || number.Num().Int64() >= int64(len(*node.Items)) {
				return false
			}
			node = (*node.Items)[number.Num().Int64()]
		default:
			return false
		}
	}
	return true
}
