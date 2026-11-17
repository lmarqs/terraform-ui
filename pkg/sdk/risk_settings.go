package sdk

import "fmt"

// RiskSettingsError validates comparison settings without preventing Plan review.
func RiskSettingsError(cfg map[string]interface{}) error {
	if _, exists := cfg["level"]; exists {
		return fmt.Errorf("%w\n\nhint: risk setting level is unsupported; use threshold", ErrorConfigurationInvalid)
	}
	if value, exists := cfg["threshold"]; exists {
		name, _ := value.(string)
		if _, valid := ParseRiskLevel(name); name != "off" && !valid {
			return ErrorConfigurationInvalid
		}
	}
	return nil
}
