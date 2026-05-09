package sdk

import (
	"testing"
	"time"
)

func TestNewConfigContext_NilMap(t *testing.T) {
	c := NewConfigContext(nil)
	if c.raw == nil {
		t.Fatal("expected non-nil raw map when created with nil")
	}
	if len(c.raw) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(c.raw))
	}
}

func TestNewConfigContext_PopulatedMap(t *testing.T) {
	m := map[string]interface{}{
		"key": "value",
	}
	c := NewConfigContext(m)
	if c.raw["key"] != "value" {
		t.Fatalf("expected raw map to contain key=value, got %v", c.raw["key"])
	}
}

func TestGet_SimpleKey(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"name": "terraform-ui",
	})
	v, ok := c.Get("name")
	if !ok {
		t.Fatal("expected key to be found")
	}
	if v != "terraform-ui" {
		t.Fatalf("expected terraform-ui, got %v", v)
	}
}

func TestGet_NestedPath(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"server": map[string]interface{}{
			"port": 8080,
		},
	})
	v, ok := c.Get("server.port")
	if !ok {
		t.Fatal("expected nested key to be found")
	}
	if v != 8080 {
		t.Fatalf("expected 8080, got %v", v)
	}
}

func TestGet_MissingPath(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"name": "test",
	})
	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestGet_MissingNestedPath(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"server": map[string]interface{}{
			"port": 8080,
		},
	})
	_, ok := c.Get("server.host")
	if ok {
		t.Fatal("expected missing nested key to return false")
	}
}

func TestGet_WrongIntermediateType(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"server": "not-a-map",
	})
	_, ok := c.Get("server.port")
	if ok {
		t.Fatal("expected wrong intermediate type to return false")
	}
}

func TestGet_DeepNesting(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"d": map[string]interface{}{
						"e": "deep-value",
					},
				},
			},
		},
	})
	v, ok := c.Get("a.b.c.d.e")
	if !ok {
		t.Fatal("expected deep nested key to be found")
	}
	if v != "deep-value" {
		t.Fatalf("expected deep-value, got %v", v)
	}
}

func TestGet_EmptyPath(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"": "empty-key",
	})
	v, ok := c.Get("")
	if !ok {
		t.Fatal("expected empty string key to be found")
	}
	if v != "empty-key" {
		t.Fatalf("expected empty-key, got %v", v)
	}
}

func TestGet_PathWithOnlyDots(t *testing.T) {
	c := NewConfigContext(map[string]interface{}{
		"name": "test",
	})
	// Path "..." splits to ["", "", "", ""] — intermediate lookups for empty keys
	_, ok := c.Get("...")
	if ok {
		t.Fatal("expected path with only dots to return false")
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		path         string
		defaultValue string
		expected     string
	}{
		{
			name:         "valid string",
			config:       map[string]interface{}{"name": "terraform-ui"},
			path:         "name",
			defaultValue: "default",
			expected:     "terraform-ui",
		},
		{
			name:         "missing key returns default",
			config:       map[string]interface{}{},
			path:         "name",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "wrong type returns default",
			config:       map[string]interface{}{"name": 123},
			path:         "name",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty default value",
			config:       map[string]interface{}{},
			path:         "name",
			defaultValue: "",
			expected:     "",
		},
		{
			name:         "nested string",
			config:       map[string]interface{}{"db": map[string]interface{}{"host": "localhost"}},
			path:         "db.host",
			defaultValue: "127.0.0.1",
			expected:     "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigContext(tt.config)
			got := c.GetString(tt.path, tt.defaultValue)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		path         string
		defaultValue int
		expected     int
	}{
		{
			name:         "int value",
			config:       map[string]interface{}{"port": 8080},
			path:         "port",
			defaultValue: 3000,
			expected:     8080,
		},
		{
			name:         "int64 value",
			config:       map[string]interface{}{"port": int64(9090)},
			path:         "port",
			defaultValue: 3000,
			expected:     9090,
		},
		{
			name:         "float64 value",
			config:       map[string]interface{}{"port": float64(4040)},
			path:         "port",
			defaultValue: 3000,
			expected:     4040,
		},
		{
			name:         "missing key returns default",
			config:       map[string]interface{}{},
			path:         "port",
			defaultValue: 3000,
			expected:     3000,
		},
		{
			name:         "wrong type returns default",
			config:       map[string]interface{}{"port": "not-a-number"},
			path:         "port",
			defaultValue: 3000,
			expected:     3000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigContext(tt.config)
			got := c.GetInt(tt.path, tt.defaultValue)
			if got != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		path         string
		defaultValue bool
		expected     bool
	}{
		{
			name:         "true value",
			config:       map[string]interface{}{"enabled": true},
			path:         "enabled",
			defaultValue: false,
			expected:     true,
		},
		{
			name:         "false value",
			config:       map[string]interface{}{"enabled": false},
			path:         "enabled",
			defaultValue: true,
			expected:     false,
		},
		{
			name:         "missing key returns default true",
			config:       map[string]interface{}{},
			path:         "enabled",
			defaultValue: true,
			expected:     true,
		},
		{
			name:         "missing key returns default false",
			config:       map[string]interface{}{},
			path:         "enabled",
			defaultValue: false,
			expected:     false,
		},
		{
			name:         "wrong type returns default",
			config:       map[string]interface{}{"enabled": "yes"},
			path:         "enabled",
			defaultValue: false,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigContext(tt.config)
			got := c.GetBool(tt.path, tt.defaultValue)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		path         string
		defaultValue time.Duration
		expected     time.Duration
	}{
		{
			name:         "5m duration",
			config:       map[string]interface{}{"timeout": "5m"},
			path:         "timeout",
			defaultValue: time.Minute,
			expected:     5 * time.Minute,
		},
		{
			name:         "1h30m duration",
			config:       map[string]interface{}{"timeout": "1h30m"},
			path:         "timeout",
			defaultValue: time.Minute,
			expected:     90 * time.Minute,
		},
		{
			name:         "invalid duration returns default",
			config:       map[string]interface{}{"timeout": "not-a-duration"},
			path:         "timeout",
			defaultValue: 10 * time.Second,
			expected:     10 * time.Second,
		},
		{
			name:         "missing key returns default",
			config:       map[string]interface{}{},
			path:         "timeout",
			defaultValue: 30 * time.Second,
			expected:     30 * time.Second,
		},
		{
			name:         "wrong type returns default",
			config:       map[string]interface{}{"timeout": 5000},
			path:         "timeout",
			defaultValue: time.Second,
			expected:     time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigContext(tt.config)
			got := c.GetDuration(tt.path, tt.defaultValue)
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]interface{}
		path         string
		defaultValue []string
		expected     []string
	}{
		{
			name:         "[]string value",
			config:       map[string]interface{}{"tags": []string{"a", "b", "c"}},
			path:         "tags",
			defaultValue: nil,
			expected:     []string{"a", "b", "c"},
		},
		{
			name:         "[]interface{} with strings",
			config:       map[string]interface{}{"tags": []interface{}{"x", "y", "z"}},
			path:         "tags",
			defaultValue: nil,
			expected:     []string{"x", "y", "z"},
		},
		{
			name:         "missing key returns default",
			config:       map[string]interface{}{},
			path:         "tags",
			defaultValue: []string{"default"},
			expected:     []string{"default"},
		},
		{
			name:         "wrong type returns default",
			config:       map[string]interface{}{"tags": "not-a-slice"},
			path:         "tags",
			defaultValue: []string{"fallback"},
			expected:     []string{"fallback"},
		},
		{
			name:         "[]interface{} with mixed types keeps only strings",
			config:       map[string]interface{}{"tags": []interface{}{"a", 1, "b", true}},
			path:         "tags",
			defaultValue: nil,
			expected:     []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigContext(tt.config)
			got := c.GetStringSlice(tt.path, tt.defaultValue)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected len %d, got len %d (%v)", len(tt.expected), len(got), got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Fatalf("at index %d: expected %q, got %q", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestRaw(t *testing.T) {
	m := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	c := NewConfigContext(m)
	raw := c.Raw()
	if raw["key1"] != "value1" {
		t.Fatalf("expected key1=value1, got %v", raw["key1"])
	}
	if raw["key2"] != 42 {
		t.Fatalf("expected key2=42, got %v", raw["key2"])
	}
}

func TestRaw_NilInit(t *testing.T) {
	c := NewConfigContext(nil)
	raw := c.Raw()
	if raw == nil {
		t.Fatal("expected non-nil raw map")
	}
	if len(raw) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(raw))
	}
}
