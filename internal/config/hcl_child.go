package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

var lockedBlocks = []string{"terraform", "member", "cache", "ai", "defaults"}

func LoadChild(dir string) (*ChildConfig, error) {
	path := filepath.Join(dir, HCLConfigFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ConfigNotFoundError{Dir: dir}
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if len(data) == 0 {
		return &ChildConfig{}, nil
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing %s: %w", path, diags)
	}

	body := file.Body
	content, diags := body.Content(childSchema())
	if diags.HasErrors() {
		return nil, fmt.Errorf("decoding %s: %w", path, diags)
	}

	if err := rejectLockedBlocks(content); err != nil {
		return nil, err
	}

	return convertChildContent(content), nil
}

func childSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "terraform"},
			{Type: "member", LabelNames: []string{"path"}},
			{Type: "cache"},
			{Type: "ai"},
			{Type: "defaults"},
			{Type: "plugin", LabelNames: []string{"name"}},
			{Type: "var_file", LabelNames: []string{"path"}},
			{Type: "var", LabelNames: []string{"name"}},
			{Type: "workspace", LabelNames: []string{"name"}},
		},
	}
}

func rejectLockedBlocks(content *hcl.BodyContent) error {
	for _, block := range content.Blocks {
		for _, locked := range lockedBlocks {
			if block.Type == locked {
				return fmt.Errorf("block %q is not allowed in child config (locked to root tfui.hcl)", locked)
			}
		}
	}
	return nil
}

func convertChildContent(content *hcl.BodyContent) *ChildConfig {
	cfg := &ChildConfig{}

	for _, block := range content.Blocks {
		switch block.Type {
		case "var_file":
			cfg.VarFiles = append(cfg.VarFiles, block.Labels[0])
		case "var":
			if cfg.Vars == nil {
				cfg.Vars = make(map[string]string)
			}
			val := extractVarValue(block.Body)
			cfg.Vars[block.Labels[0]] = val
		case "plugin":
			if cfg.Plugins == nil {
				cfg.Plugins = make(map[string]PluginSettings)
			}
			cfg.Plugins[block.Labels[0]] = extractChildPlugin(block.Body)
		case "workspace":
			ws := convertWorkspaceBlock(block)
			cfg.Workspaces = append(cfg.Workspaces, ws)
		}
	}

	return cfg
}

func convertWorkspaceBlock(block *hcl.Block) WorkspaceConfig {
	ws := WorkspaceConfig{
		Name: block.Labels[0],
	}

	wsContent, diags := block.Body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "lock_timeout"},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "var_file", LabelNames: []string{"path"}},
			{Type: "var", LabelNames: []string{"name"}},
			{Type: "plugin", LabelNames: []string{"name"}},
		},
	})
	if diags.HasErrors() {
		return ws
	}

	if attr, ok := wsContent.Attributes["lock_timeout"]; ok {
		val, diags := attr.Expr.Value(nil)
		if !diags.HasErrors() && val.Type() == cty.String {
			ws.LockTimeout = val.AsString()
		}
	}

	for _, b := range wsContent.Blocks {
		switch b.Type {
		case "var_file":
			ws.VarFiles = append(ws.VarFiles, b.Labels[0])
		case "var":
			if ws.Vars == nil {
				ws.Vars = make(map[string]string)
			}
			ws.Vars[b.Labels[0]] = extractVarValue(b.Body)
		case "plugin":
			if ws.Plugins == nil {
				ws.Plugins = make(map[string]PluginSettings)
			}
			ws.Plugins[b.Labels[0]] = extractChildPlugin(b.Body)
		}
	}

	return ws
}

func extractVarValue(body hcl.Body) string {
	content, diags := body.Content(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "value"},
		},
	})
	if diags.HasErrors() {
		return ""
	}
	attr, ok := content.Attributes["value"]
	if !ok {
		return ""
	}
	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() || val.Type() != cty.String {
		return ""
	}
	return val.AsString()
}

func extractChildPlugin(body hcl.Body) PluginSettings {
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		return PluginSettings{Enabled: true}
	}

	ps := PluginSettings{
		Enabled: true,
		Options: make(map[string]interface{}),
	}

	for name, attr := range attrs {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			continue
		}
		if name == "enabled" && val.Type() == cty.Bool {
			ps.Enabled = val.True()
			continue
		}
		switch {
		case val.Type() == cty.String:
			ps.Options[name] = val.AsString()
		case val.Type() == cty.Bool:
			ps.Options[name] = val.True()
		case val.Type() == cty.Number:
			f, _ := val.AsBigFloat().Float64()
			ps.Options[name] = f
		}
	}

	return ps
}
