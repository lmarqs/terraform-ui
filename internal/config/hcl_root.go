package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/zclconf/go-cty/cty"
)

const HCLConfigFileName = "tfui.hcl"

func LoadRoot(dir string) (*RootConfig, error) {
	path := filepath.Join(dir, HCLConfigFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ConfigNotFoundError{Dir: dir}
		}
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	if len(data) == 0 {
		return &RootConfig{}, nil
	}

	var raw hclRootFile
	err = hclsimple.Decode(path, data, nil, &raw)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return convertRootFile(&raw), nil
}

type hclRootFile struct {
	Terraform *hclTerraformBlock `hcl:"terraform,block"`
	Members   []hclMemberBlock   `hcl:"member,block"`
	Cache     *hclCacheBlock     `hcl:"cache,block"`
	AI        *hclAIBlock        `hcl:"ai,block"`
	Defaults  *hclDefaultsBlock  `hcl:"defaults,block"`
	Remain    hcl.Body           `hcl:",remain"`
}

type hclTerraformBlock struct {
	Bin string `hcl:"bin,optional"`
}

type hclMemberBlock struct {
	Path   string   `hcl:"path,label"`
	Remain hcl.Body `hcl:",remain"`
}

type hclCacheBlock struct {
	StalenessThreshold string `hcl:"staleness_threshold"`
}

type hclAIBlock struct {
	Enabled  bool   `hcl:"enabled,optional"`
	Provider string `hcl:"provider,optional"`
	Model    string `hcl:"model,optional"`
	Region   string `hcl:"region,optional"`
}

type hclDefaultsBlock struct {
	Parallelism int              `hcl:"parallelism,optional"`
	Lock        *bool            `hcl:"lock"`
	VarFiles    []hclVarFile     `hcl:"var_file,block"`
	Plugins     []hclPluginBlock `hcl:"plugin,block"`
	Remain      hcl.Body         `hcl:",remain"`
}

type hclVarFile struct {
	Path string `hcl:"path,label"`
}

type hclPluginBlock struct {
	Name    string   `hcl:"name,label"`
	Enabled *bool    `hcl:"enabled"`
	Remain  hcl.Body `hcl:",remain"`
}

func convertRootFile(raw *hclRootFile) *RootConfig {
	cfg := &RootConfig{}

	if raw.Terraform != nil {
		cfg.Terraform.Bin = raw.Terraform.Bin
	}

	for _, m := range raw.Members {
		cfg.Members = append(cfg.Members, MemberConfig{Path: m.Path})
	}

	if raw.Cache != nil {
		cfg.Cache.StalenessThreshold = raw.Cache.StalenessThreshold
	}

	if raw.AI != nil {
		cfg.AI.Enabled = raw.AI.Enabled
		cfg.AI.Provider = raw.AI.Provider
		cfg.AI.Model = raw.AI.Model
		cfg.AI.Region = raw.AI.Region
	}

	if raw.Defaults != nil {
		cfg.Defaults.Parallelism = raw.Defaults.Parallelism
		cfg.Defaults.Lock = raw.Defaults.Lock

		for _, vf := range raw.Defaults.VarFiles {
			cfg.Defaults.VarFiles = append(cfg.Defaults.VarFiles, vf.Path)
		}

		if len(raw.Defaults.Plugins) > 0 {
			cfg.Defaults.Plugins = make(map[string]PluginSettings)
			for _, p := range raw.Defaults.Plugins {
				ps := PluginSettings{
					Enabled: p.Enabled == nil || *p.Enabled,
					Options: extractPluginOptions(p.Remain),
				}
				cfg.Defaults.Plugins[p.Name] = ps
			}
		}
	}

	return cfg
}

func extractPluginOptions(body hcl.Body) map[string]interface{} {
	if body == nil {
		return nil
	}
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() || len(attrs) == 0 {
		return nil
	}
	opts := make(map[string]interface{})
	for name, attr := range attrs {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			continue
		}
		switch {
		case val.Type() == cty.String:
			opts[name] = val.AsString()
		case val.Type() == cty.Bool:
			opts[name] = val.True()
		case val.Type() == cty.Number:
			f, _ := val.AsBigFloat().Float64()
			opts[name] = f
		}
	}
	return opts
}
