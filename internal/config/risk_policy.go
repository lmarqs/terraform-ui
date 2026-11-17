package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"

	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

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
	root, err := policyConfigBody(project)
	if err != nil {
		return RiskPolicySelection{}, err
	}
	selection := RiskPolicySelection{}
	if root == nil {
		return selection, nil
	}
	content, _, diags := root.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "defaults"}}})
	if diags.HasErrors() || len(content.Blocks) > 1 {
		return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
	}
	if len(content.Blocks) == 1 {
		selection, err = policySelector(content.Blocks[0].Body, project, selection)
		if err != nil {
			return RiskPolicySelection{}, err
		}
	}
	// A root Member uses the root schema, never the Member schema a second time.
	if project == chdir {
		return selection, nil
	}
	memberBody, err := policyConfigBody(chdir)
	if err != nil {
		return RiskPolicySelection{}, err
	}
	if memberBody == nil {
		return selection, nil
	}
	memberContent, diags := memberBody.Content(childSchema())
	if diags.HasErrors() || rejectLockedBlocks(memberContent) != nil {
		return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
	}
	var blocks []*hcl.Block
	var workspaces []WorkspaceConfig
	for _, block := range memberContent.Blocks {
		if block.Type != "workspace" {
			continue
		}
		plugins, _, diags := block.Body.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "plugin", LabelNames: []string{"name"}}}})
		if diags.HasErrors() || rejectLockedBlocks(plugins) != nil {
			return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
		}
		blocks = append(blocks, block)
		workspaces = append(workspaces, WorkspaceConfig{Name: block.Labels[0]})
	}
	selection, err = policySelector(memberBody, chdir, selection)
	if err != nil {
		return RiskPolicySelection{}, err
	}
	matched := matchWorkspace(workspaces, workspace)
	if workspace != "" && matched != nil {
		for i := range workspaces {
			if matched == &workspaces[i] {
				return policySelector(blocks[i].Body, chdir, selection)
			}
		}
	}
	return selection, nil
}

func policyConfigBody(dir string) (hcl.Body, error) {
	path := filepath.Join(dir, HCLConfigFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, sdk.ErrorConfigurationInvalid
	}
	file, diags := hclparse.NewParser().ParseHCL(data, path)
	if diags.HasErrors() {
		return nil, sdk.ErrorConfigurationInvalid
	}
	return file.Body, nil
}

func policySelector(body hcl.Body, origin string, inherited RiskPolicySelection) (RiskPolicySelection, error) {
	content, _, _ := body.PartialContent(&hcl.BodySchema{Attributes: []hcl.AttributeSchema{{Name: "risk_policy"}}})
	attr, ok := content.Attributes["risk_policy"]
	if !ok {
		return inherited, nil
	}
	value, diags := attr.Expr.Value(nil)
	if diags.HasErrors() || !value.IsKnown() || value.IsNull() || value.Type() != cty.String {
		return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
	}
	path := value.AsString()
	if path == "" {
		return RiskPolicySelection{Origin: origin}, nil
	}
	if strings.TrimSpace(path) == "" || strings.Contains(path, "://") {
		return RiskPolicySelection{}, sdk.ErrorConfigurationInvalid
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(origin, path)
	}
	return RiskPolicySelection{Path: filepath.Clean(path), Origin: origin}, nil
}
