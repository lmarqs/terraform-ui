package sdk

import "testing"

func TestLoadAIConfig_WhenAllFieldsSet_ShouldReturnPopulatedConfig(t *testing.T) {
	cfg := NewConfigContext(map[string]interface{}{
		"ai": map[string]interface{}{
			"enabled":  true,
			"provider": "bedrock",
			"model":    "claude-3-5-sonnet",
			"region":   "us-east-1",
		},
	})

	result := LoadAIConfig(cfg)

	if !result.Enabled {
		t.Error("Enabled = false, want true")
	}
	if result.Provider != "bedrock" {
		t.Errorf("Provider = %q, want %q", result.Provider, "bedrock")
	}
	if result.Model != "claude-3-5-sonnet" {
		t.Errorf("Model = %q, want %q", result.Model, "claude-3-5-sonnet")
	}
	if result.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", result.Region, "us-east-1")
	}
}

func TestLoadAIConfig_WhenNoAISection_ShouldReturnDefaults(t *testing.T) {
	cfg := NewConfigContext(map[string]interface{}{})

	result := LoadAIConfig(cfg)

	if result.Enabled {
		t.Error("Enabled = true, want false")
	}
	if result.Provider != "" {
		t.Errorf("Provider = %q, want empty", result.Provider)
	}
	if result.Model != "" {
		t.Errorf("Model = %q, want empty", result.Model)
	}
	if result.Region != "" {
		t.Errorf("Region = %q, want empty", result.Region)
	}
}

func TestLoadAIConfig_WhenPartialConfig_ShouldReturnDefaultsForMissing(t *testing.T) {
	cfg := NewConfigContext(map[string]interface{}{
		"ai": map[string]interface{}{
			"enabled": true,
		},
	})

	result := LoadAIConfig(cfg)

	if !result.Enabled {
		t.Error("Enabled = false, want true")
	}
	if result.Provider != "" {
		t.Errorf("Provider = %q, want empty", result.Provider)
	}
}
