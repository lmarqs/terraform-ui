package sdk

import "fmt"

// RiskSettingsError validates comparison settings without preventing Plan review.
func RiskSettingsError(cfg map[string]interface{}) error {
	if _, exists := cfg["level"]; exists {
		return fmt.Errorf("%w\n\nhint: risk setting level is unsupported; use threshold", ErrorConfigurationInvalid)
	}
	if value, exists := cfg["threshold"]; exists {
		switch value {
		case "off", "none", "low", "medium", "high", "critical":
		default:
			return ErrorConfigurationInvalid
		}
	}
	return nil
}
