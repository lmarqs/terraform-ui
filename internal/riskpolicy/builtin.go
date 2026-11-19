package riskpolicy

import _ "embed"

//go:embed builtin.rego
var builtinSource string

// BuiltinModules returns the complete built-in policy source.
func BuiltinModules() []Module {
	return []Module{{Name: "builtin.rego", Source: builtinSource}}
}
