package config

import (
	"os"
	"path/filepath"
	"testing"
)

var invalidRiskPolicyValues = []struct{ name, value string }{
	{"null", `risk_policy = null`},
	{"number", `risk_policy = 12`},
	{"bool", `risk_policy = true`},
	{"unknown expression", `risk_policy = env.POLICY`},
	{"URI", `risk_policy = "https://example.test/policy.rego"`},
	{"whitespace only", `risk_policy = "  "`},
	{"duplicate", "risk_policy = \"first\"\nrisk_policy = \"second\""},
}

func assertSelector(t *testing.T, label string, got *PolicySelector, want *PolicySelector) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("%s = %+v, want nil", label, *got)
		}
		return
	}
	if got == nil {
		t.Fatalf("%s = nil, want %+v", label, *want)
	}
	if *got != *want {
		t.Fatalf("%s = %+v, want %+v", label, *got, *want)
	}
}

func TestLoadRoot_GivenDefaultsRiskPolicy_ShouldRecordSelectorWithDeclaringOrigin(t *testing.T) {
	tests := []struct {
		name, hcl string
		want      *PolicySelector
	}{
		{"declared", `defaults { risk_policy = "policies/root" }`, &PolicySelector{Value: "policies/root"}},
		{"reset to builtin", `defaults { risk_policy = "" }`, &PolicySelector{Value: ""}},
		{"literal spaces", `defaults { risk_policy = " policies/team " }`, &PolicySelector{Value: " policies/team "}},
		{"absent", `defaults { parallelism = 3 }`, nil},
		{"no defaults block", `terraform { bin = "terraform" }`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeHCL(t, dir, tt.hcl)
			cfg, err := LoadRoot(dir)
			if err != nil {
				t.Fatalf("LoadRoot error = %v", err)
			}
			want := tt.want
			if want != nil {
				want.Origin = dir
			}
			assertSelector(t, "Defaults.RiskPolicy", cfg.Defaults.RiskPolicy, want)
		})
	}
}

func TestLoadRoot_GivenInvalidRiskPolicy_ShouldRecordInvalidWithoutError(t *testing.T) {
	for _, tt := range invalidRiskPolicyValues {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeHCL(t, dir, "defaults {\n"+tt.value+"\nparallelism = 4\n}")
			cfg, err := LoadRoot(dir)
			if err != nil {
				t.Fatalf("invalid risk_policy must not abort startup; LoadRoot error = %v", err)
			}
			if cfg.Defaults.RiskPolicy == nil || !cfg.Defaults.RiskPolicy.Invalid {
				t.Fatalf("Defaults.RiskPolicy = %+v, want Invalid", cfg.Defaults.RiskPolicy)
			}
			if cfg.Defaults.Parallelism != 4 {
				t.Fatalf("invalid risk_policy lost execution defaults: parallelism = %d", cfg.Defaults.Parallelism)
			}
		})
	}
}

func TestLoadChild_GivenRiskPolicySelectors_ShouldRecordMemberAndWorkspaceSelectors(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `risk_policy = "policies/member"
workspace "prod" { risk_policy = "policies/prod" }
workspace "staging" { risk_policy = "" }
workspace "dev" { lock_timeout = "5s" }`)
	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild error = %v", err)
	}
	assertSelector(t, "ChildConfig.RiskPolicy", cfg.RiskPolicy, &PolicySelector{Value: "policies/member", Origin: dir})
	want := map[string]*PolicySelector{
		"prod":    {Value: "policies/prod", Origin: dir},
		"staging": {Value: "", Origin: dir},
		"dev":     nil,
	}
	if len(cfg.Workspaces) != len(want) {
		t.Fatalf("workspaces = %+v", cfg.Workspaces)
	}
	for _, ws := range cfg.Workspaces {
		assertSelector(t, "workspace "+ws.Name+" RiskPolicy", ws.RiskPolicy, want[ws.Name])
	}
}

func TestLoadChild_GivenNoRiskPolicy_ShouldLeaveSelectorUnset(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `var "region" { value = "us-east-1" }`)
	cfg, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("LoadChild error = %v", err)
	}
	assertSelector(t, "ChildConfig.RiskPolicy", cfg.RiskPolicy, nil)
}

func TestLoadChild_GivenInvalidRiskPolicy_ShouldRecordInvalidWithoutError(t *testing.T) {
	for _, tt := range invalidRiskPolicyValues {
		t.Run("member/"+tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeHCL(t, dir, tt.value+"\nvar \"region\" { value = \"us-east-1\" }")
			cfg, err := LoadChild(dir)
			if err != nil {
				t.Fatalf("invalid risk_policy must be data, not a load error; LoadChild error = %v", err)
			}
			if cfg.RiskPolicy == nil || !cfg.RiskPolicy.Invalid {
				t.Fatalf("ChildConfig.RiskPolicy = %+v, want Invalid", cfg.RiskPolicy)
			}
			if cfg.Vars["region"] != "us-east-1" {
				t.Fatalf("invalid risk_policy lost member execution settings: vars = %v", cfg.Vars)
			}
		})
		t.Run("workspace/"+tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeHCL(t, dir, "workspace \"prod\" {\n"+tt.value+"\nlock_timeout = \"30s\"\n}")
			cfg, err := LoadChild(dir)
			if err != nil {
				t.Fatalf("invalid risk_policy must be data, not a load error; LoadChild error = %v", err)
			}
			if len(cfg.Workspaces) != 1 {
				t.Fatalf("workspaces = %+v", cfg.Workspaces)
			}
			ws := cfg.Workspaces[0]
			if ws.RiskPolicy == nil || !ws.RiskPolicy.Invalid {
				t.Fatalf("WorkspaceConfig.RiskPolicy = %+v, want Invalid", ws.RiskPolicy)
			}
			if ws.LockTimeout != "30s" {
				t.Fatalf("invalid risk_policy lost workspace execution settings: lock_timeout = %q", ws.LockTimeout)
			}
		})
	}
}

func TestResolve_GivenRiskPolicyLayers_ShouldSelectDefaultsThenMemberThenWorkspace(t *testing.T) {
	rootSel := &PolicySelector{Value: "root.rego", Origin: "/project"}
	memberSel := &PolicySelector{Value: "member.rego", Origin: "/project/member"}
	exactSel := &PolicySelector{Value: "exact.rego", Origin: "/project/member"}
	globSel := &PolicySelector{Value: "glob.rego", Origin: "/project/member"}
	resetSel := &PolicySelector{Value: "", Origin: "/project/member"}
	invalidSel := &PolicySelector{Origin: "/project/member", Invalid: true}
	tests := []struct {
		name      string
		root      *RootConfig
		child     *ChildConfig
		workspace string
		want      *PolicySelector
	}{
		{"no root", nil, nil, "prod", nil},
		{"nothing declared", &RootConfig{}, &ChildConfig{}, "prod", nil},
		{"defaults only", &RootConfig{Defaults: DefaultsConfig{RiskPolicy: rootSel}}, nil, "prod", rootSel},
		{"member inherits defaults", &RootConfig{Defaults: DefaultsConfig{RiskPolicy: rootSel}}, &ChildConfig{}, "prod", rootSel},
		{"member replaces defaults", &RootConfig{Defaults: DefaultsConfig{RiskPolicy: rootSel}}, &ChildConfig{RiskPolicy: memberSel}, "prod", memberSel},
		{"member reset", &RootConfig{Defaults: DefaultsConfig{RiskPolicy: rootSel}}, &ChildConfig{RiskPolicy: resetSel}, "prod", resetSel},
		{"no workspace keeps member", &RootConfig{}, &ChildConfig{RiskPolicy: memberSel, Workspaces: []WorkspaceConfig{{Name: "prod", RiskPolicy: exactSel}}}, "", memberSel},
		{"unmatched workspace keeps member", &RootConfig{}, &ChildConfig{RiskPolicy: memberSel, Workspaces: []WorkspaceConfig{{Name: "prod", RiskPolicy: exactSel}}}, "staging", memberSel},
		{"matched workspace without selector keeps member", &RootConfig{}, &ChildConfig{RiskPolicy: memberSel, Workspaces: []WorkspaceConfig{{Name: "prod"}}}, "prod", memberSel},
		{"exact beats earlier glob", &RootConfig{}, &ChildConfig{RiskPolicy: memberSel, Workspaces: []WorkspaceConfig{{Name: "prod*", RiskPolicy: globSel}, {Name: "prod", RiskPolicy: exactSel}}}, "prod", exactSel},
		{"glob when no exact", &RootConfig{}, &ChildConfig{RiskPolicy: memberSel, Workspaces: []WorkspaceConfig{{Name: "prod*", RiskPolicy: globSel}}}, "production", globSel},
		{"workspace reset", &RootConfig{Defaults: DefaultsConfig{RiskPolicy: rootSel}}, &ChildConfig{Workspaces: []WorkspaceConfig{{Name: "prod", RiskPolicy: resetSel}}}, "prod", resetSel},
		{"invalid workspace replaces member", &RootConfig{}, &ChildConfig{RiskPolicy: memberSel, Workspaces: []WorkspaceConfig{{Name: "prod", RiskPolicy: invalidSel}}}, "prod", invalidSel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelector(t, "Resolve().RiskPolicy()", Resolve(tt.root, tt.child, tt.workspace).RiskPolicy(), tt.want)
		})
	}
}

// Every loaded layer's selector must be valid, even one a later layer overrides,
// so an invalid declaration never silently falls back to another layer.
func TestResolveRiskPolicy_GivenInvalidSelectorOverriddenByLaterLayer_ShouldStillReportFixedFailure(t *testing.T) {
	tests := []struct{ name, root, member string }{
		{"invalid root under valid member", `defaults { risk_policy = null }`, `risk_policy = "member.rego"`},
		{"invalid member under valid workspace", `defaults { risk_policy = "root.rego" }`, "risk_policy = 12\nworkspace \"prod\" { risk_policy = \"ws.rego\" }"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			member := filepath.Join(project, "member")
			if err := os.Mkdir(member, 0755); err != nil {
				t.Fatal(err)
			}
			writeHCL(t, project, tt.root)
			writeHCL(t, member, tt.member)
			selected, err := ResolveRiskPolicy(project, member, "prod")
			if err == nil || err.Error() != "configuration_invalid" || selected != (RiskPolicySelection{}) {
				t.Fatalf("selection=%+v error=%v", selected, err)
			}
		})
	}
}

func TestResolveRiskPolicy_GivenInvalidSelectorInUnmatchedWorkspace_ShouldSelectMemberPolicy(t *testing.T) {
	project := t.TempDir()
	member := filepath.Join(project, "member")
	if err := os.Mkdir(member, 0755); err != nil {
		t.Fatal(err)
	}
	writeHCL(t, project, "")
	writeHCL(t, member, `risk_policy = "member.rego"
workspace "staging" { risk_policy = null }`)
	selected, err := ResolveRiskPolicy(project, member, "prod")
	if err != nil || selected.Path != filepath.Join(member, "member.rego") || selected.Origin != member {
		t.Fatalf("selection=%+v error=%v", selected, err)
	}
}
