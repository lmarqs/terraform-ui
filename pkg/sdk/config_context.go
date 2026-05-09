package sdk

import (
	"strings"
	"time"
)

// ConfigContext wraps the parsed tfui.yaml with dot-notation access.
// All getters accept a default value — no nil surprises.
type ConfigContext struct {
	raw map[string]interface{}
}

// NewConfigContext creates a ConfigContext from a parsed yaml map.
func NewConfigContext(raw map[string]interface{}) *ConfigContext {
	if raw == nil {
		raw = make(map[string]interface{})
	}
	return &ConfigContext{raw: raw}
}

// Get traverses a dot-separated path and returns the value and whether it was found.
func (c *ConfigContext) Get(path string) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = c.raw

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// GetString returns the string at path, or defaultValue if missing or wrong type.
func (c *ConfigContext) GetString(path string, defaultValue string) string {
	v, ok := c.Get(path)
	if !ok {
		return defaultValue
	}
	s, ok := v.(string)
	if !ok {
		return defaultValue
	}
	return s
}

// GetInt returns the int at path, or defaultValue if missing or wrong type.
func (c *ConfigContext) GetInt(path string, defaultValue int) int {
	v, ok := c.Get(path)
	if !ok {
		return defaultValue
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return defaultValue
	}
}

// GetBool returns the bool at path, or defaultValue if missing or wrong type.
func (c *ConfigContext) GetBool(path string, defaultValue bool) bool {
	v, ok := c.Get(path)
	if !ok {
		return defaultValue
	}
	b, ok := v.(bool)
	if !ok {
		return defaultValue
	}
	return b
}

// GetDuration returns the duration at path (parsed from string like "5m"), or defaultValue.
func (c *ConfigContext) GetDuration(path string, defaultValue time.Duration) time.Duration {
	v, ok := c.Get(path)
	if !ok {
		return defaultValue
	}
	s, ok := v.(string)
	if !ok {
		return defaultValue
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue
	}
	return d
}

// GetStringSlice returns []string at path, or defaultValue.
func (c *ConfigContext) GetStringSlice(path string, defaultValue []string) []string {
	v, ok := c.Get(path)
	if !ok {
		return defaultValue
	}
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return defaultValue
	}
}

// Raw returns the underlying map for advanced access.
func (c *ConfigContext) Raw() map[string]interface{} {
	return c.raw
}
