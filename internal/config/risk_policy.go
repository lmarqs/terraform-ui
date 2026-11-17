package config

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

const riskPolicyAttribute = "risk_policy"

// RiskPolicySelection retains the declaration origin even when a selector is inherited.
// An empty Path selects the complete built-in policy.
type RiskPolicySelection struct {
	Path   string
	Origin string
}

// ResolveRiskPolicy rereads policy configuration independently of execution settings:
// callers can retain usable Plan settings while reporting unavailable risk.
func ResolveRiskPolicy(projectDir, chdirDir, workspace string) (RiskPolicySelection, error) {
	project, err := filepath.Abs(projectDir)
	if err != nil {
		return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
	}
	chdir, err := filepath.Abs(chdirDir)
	if err != nil {
		return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
	}
	root, err := LoadRoot(project)
	if isConfigNotFound(err) {
		return RiskPolicySelection{}, nil
	}
	if err != nil {
		return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
	}
	declared := []*PolicySelector{root.Defaults.RiskPolicy}
	var child *ChildConfig
	// A root Member uses the root schema, never the Member schema a second time.
	if project != chdir {
		child, err = LoadChild(chdir)
		if err != nil && !isConfigNotFound(err) {
			return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
		}
	}
	if child != nil {
		declared = append(declared, child.RiskPolicy)
		if workspace != "" {
			if ws := matchWorkspace(child.Workspaces, workspace); ws != nil {
				declared = append(declared, ws.RiskPolicy)
			}
		}
	}
	// An invalid declaration fails even when a later layer overrides it, so it
	// never silently falls back to another layer's policy.
	for _, selector := range declared {
		if selector != nil && selector.Invalid {
			return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
		}
	}
	return selectionFor(Resolve(root, child, workspace).RiskPolicy()), nil
}

func isConfigNotFound(err error) bool {
	var notFound *ConfigNotFoundError
	return errors.As(err, &notFound)
}

func selectionFor(selector *PolicySelector) RiskPolicySelection {
	if selector == nil {
		return RiskPolicySelection{}
	}
	if selector.Value == "" {
		return RiskPolicySelection{Origin: selector.Origin}
	}
	path := selector.Value
	if !filepath.IsAbs(path) {
		path = filepath.Join(selector.Origin, path)
	}
	return RiskPolicySelection{Path: filepath.Clean(path), Origin: selector.Origin}
}

// riskPolicyRedefinitions holds the parser's "Attribute redefined" errors for
// risk_policy. The loaders record each one as an invalid selector in the body
// that declares it; any left unclaimed stays a load error.
type riskPolicyRedefinitions struct {
	pending hcl.Diagnostics
}

// splitRiskPolicyRedefinitions separates risk_policy redefinitions from the
// other parse diagnostics.
func splitRiskPolicyRedefinitions(src []byte, diags hcl.Diagnostics) (hcl.Diagnostics, *riskPolicyRedefinitions) {
	var rest hcl.Diagnostics
	redefinitions := &riskPolicyRedefinitions{}
	for _, diag := range diags {
		if diag.Summary == "Attribute redefined" && diag.Subject != nil && string(diag.Subject.SliceBytes(src)) == riskPolicyAttribute {
			redefinitions.pending = append(redefinitions.pending, diag)
			continue
		}
		rest = append(rest, diag)
	}
	return rest, redefinitions
}

// claim removes the redefinitions that sit directly in body, outside its nested
// blocks, and reports whether there were any.
func (r *riskPolicyRedefinitions) claim(body hcl.Body) bool {
	syntax := body.(*hclsyntax.Body)
	var unclaimed hcl.Diagnostics
	for _, diag := range r.pending {
		if !directlyIn(syntax, diag.Subject.Start.Byte) {
			unclaimed = append(unclaimed, diag)
		}
	}
	claimed := len(unclaimed) < len(r.pending)
	r.pending = unclaimed
	return claimed
}

func directlyIn(body *hclsyntax.Body, offset int) bool {
	if !body.SrcRange.ContainsOffset(offset) {
		return false
	}
	for _, block := range body.Blocks {
		if block.Body.SrcRange.ContainsOffset(offset) {
			return false
		}
	}
	return true
}

// policySelector reads the risk_policy attribute declared in body.
func (r *riskPolicyRedefinitions) policySelector(body hcl.Body, attr *hcl.Attribute, origin string) *PolicySelector {
	redefined := r.claim(body)
	if attr == nil {
		return nil
	}
	selector := &PolicySelector{Origin: origin, Invalid: true}
	value, diags := attr.Expr.Value(nil)
	if redefined || diags.HasErrors() || !value.IsKnown() || value.IsNull() || value.Type() != cty.String {
		return selector
	}
	path := value.AsString()
	if path != "" && (strings.TrimSpace(path) == "" || strings.Contains(path, "://")) {
		return selector
	}
	return &PolicySelector{Value: path, Origin: origin}
}
