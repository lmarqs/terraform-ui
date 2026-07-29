package sdk

// ConfigStrings reads a list-of-strings option out of the map handed to
// Plugin.Configure, returning def when the key is absent or carries another
// shape. Both []string and []interface{} are accepted, since the value's origin
// is a config decoder the plugin does not see.
func ConfigStrings(cfg map[string]interface{}, key string, def []string) []string {
	raw, ok := cfg[key]
	if !ok {
		return def
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, elem := range v {
			if s, ok := elem.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return def
}
