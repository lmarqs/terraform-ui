package sdk

import "errors"

// RiskSettingsError validates comparison settings without preventing Plan review.
func RiskSettingsError(cfg map[string]interface{}) error {
	if _, exists := cfg["level"]; exists {
		return errors.New("configuration_invalid\n\nhint: risk setting level is unsupported; use threshold")
	}
	if value, exists := cfg["threshold"]; exists {
		switch value {
		case "off", "none", "low", "medium", "high", "critical":
		default:
			return errors.New("configuration_invalid")
		}
	}
	return nil
}
