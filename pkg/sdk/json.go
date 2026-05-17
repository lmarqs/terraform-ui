package sdk

import "encoding/json"

// MarshalJSON encodes v as indented JSON with a trailing newline.
// Panics if marshaling fails — only use with structs whose fields are
// known-safe types (string, int, bool, slices/maps thereof).
func MarshalJSON(v any) []byte {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic("sdk.MarshalJSON: " + err.Error())
	}
	return append(data, '\n')
}
