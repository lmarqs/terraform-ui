package riskpolicy

import (
	"strings"

	"github.com/open-policy-agent/opa/v1/ast"
)

const evaluatorVersion = "v1.19.1"
const capabilityDigest = "sha256:fac4b2c6a2137bfbc140325885931e3a9a328a940ca33fff52b37b002699fed5"

// Declarations come from the pinned runtime; tests compare the complete profile
// with the frozen contract artifact so dependency upgrades cannot expand it.
func capabilities() *ast.Capabilities {
	c := &ast.Capabilities{
		FutureKeywords: []string{"not"},
		Features:       []string{"keywords_in_refs", "rego_v1", "template_strings"},
		AllowNet:       []string{},
	}
	for _, name := range strings.Fields("eq equal neq lt lte gt gte count is_array is_object is_string is_number is_boolean is_null object.get object.keys array.concat concat sprintf startswith endswith contains lower upper to_number") {
		c.Builtins = append(c.Builtins, ast.BuiltinMap[name])
	}
	return c
}

func deniedOutput(module *ast.Module) bool {
	denied := false
	ast.WalkExprs(module, func(e *ast.Expr) bool {
		if e.IsCall() && outputCall(e.Operator().String()) {
			denied = true
		}
		return false
	})
	ast.WalkTerms(module, func(t *ast.Term) bool {
		if call, ok := t.Value.(ast.Call); ok && outputCall(call[0].String()) {
			denied = true
		}
		return false
	})
	return denied
}

func outputCall(name string) bool { return name == "print" || name == "trace" }
