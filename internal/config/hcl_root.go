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

// DefaultMemberPath is the implicit single member used when a tfui.hcl defines
// no member blocks: the project root itself.
const DefaultMemberPath = "."

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
		return convertRootFile(&hclRootFile{}), nil
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
	if len(cfg.Members) == 0 {
		cfg.Members = []MemberConfig{{Path: DefaultMemberPath}}
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
		if v, ok := pluginOptionValue(val); ok {
			opts[name] = v
		}
	}
	return opts
}

// pluginOptionValue converts one HCL attribute into the plugin option types the
// SDK exposes to plugins: string, bool, float64, and []string. Anything else has
// no reader on the plugin side, so it is dropped rather than passed on in a
// shape no plugin can consume.
func pluginOptionValue(val cty.Value) (interface{}, bool) {
	switch {
	case val.Type() == cty.String:
		return val.AsString(), true
	case val.Type() == cty.Bool:
		return val.True(), true
	case val.Type() == cty.Number:
		f, _ := val.AsBigFloat().Float64()
		return f, true
	case isStringSequence(val.Type()):
		return stringsFromCty(val), true
	}
	return nil, false
}

// isStringSequence reports whether a value can be read as an ordered sequence.
// Maps and objects iterate too, but they carry keys no plugin option consumes —
// and AsValueSlice does not accept them.
func isStringSequence(t cty.Type) bool {
	return t.IsTupleType() || t.IsListType() || t.IsSetType()
}

// stringsFromCty collects the string elements of a tuple or list value. Non-string
// elements are skipped: a resource address list with a number in it is a typo,
// not an instruction.
func stringsFromCty(val cty.Value) []string {
	out := []string{}
	for _, elem := range val.AsValueSlice() {
		if elem.Type() == cty.String && !elem.IsNull() {
			out = append(out, elem.AsString())
		}
	}
	return out
}
