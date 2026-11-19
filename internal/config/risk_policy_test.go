package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadChild_GivenRiskPolicySelectors_ShouldPreserveExecutionSettings(t *testing.T) {
	dir := t.TempDir()
	writeHCL(t, dir, `risk_policy = "policies/member"
var "region" { value = "us-east-1" }
workspace "prod" {
  risk_policy = "policies/prod"
  lock_timeout = "30s"
  var "region" { value = "us-west-2" }
}`)
	memberConfig, err := LoadChild(dir)
	if err != nil {
		t.Fatalf("valid dedicated policy selectors must not prevent Member execution settings loading: %v", err)
	}
	resolved := Resolve(&RootConfig{}, memberConfig, "prod")
	if resolved.Vars()["region"] != "us-west-2" || resolved.LockTimeout() != "30s" {
		t.Fatalf("workspace execution settings lost: vars=%v lock_timeout=%q", resolved.Vars(), resolved.LockTimeout())
	}
}

func TestResolveRiskPolicy_GivenLayeredHCL_ShouldSelectCompleteSourceWithDeclarationOrigin(t *testing.T) {
	tests := []struct{ name, memberConfig, workspace, relative, origin string }{
		{"inherit root", `workspace "prod" { lock_timeout = "30s" }`, "prod", "policies/root", "root"},
		{"member replaces root", `risk_policy = "policies/member"`, "prod", "policies/member", "member"},
		{"exact beats earlier glob", `risk_policy = "policies/member"
workspace "prod*" { risk_policy = "policies/glob" }
workspace "prod" { risk_policy = "policies/exact" }`, "prod", "policies/exact", "member"},
		{"first glob wins", `workspace "prod*" { risk_policy = "policies/first" }
workspace "*" { risk_policy = "policies/second" }`, "production", "policies/first", "member"},
		{"member reset", `risk_policy = ""`, "prod", "", "member"},
		{"workspace reset", `workspace "prod" { risk_policy = "" }`, "prod", "", "member"},
		{"literal spaces", `risk_policy = " policies/team "`, "prod", " policies/team ", "member"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			member := filepath.Join(root, "services", "api")
			if err := os.MkdirAll(member, 0755); err != nil {
				t.Fatal(err)
			}
			writeHCL(t, root, `member "services/api" {}
defaults { risk_policy = "policies/root" }`)
			writeHCL(t, member, tt.memberConfig)
			selected, err := ResolveRiskPolicy(root, member, tt.workspace)
			if err != nil {
				t.Fatal(err)
			}
			origin := root
			if tt.origin == "member" {
				origin = member
			}
			path := ""
			if tt.relative != "" {
				path = filepath.Join(origin, tt.relative)
			}
			if selected.Path != path || selected.Origin != origin {
				t.Fatalf("selection = %+v, want path=%q origin=%q", selected, path, origin)
			}
		})
	}
}

func TestResolveRiskPolicy_GivenNoLoadedSelector_ShouldUseBuiltinWithoutParentDiscovery(t *testing.T) {
	parent := t.TempDir()
	member := filepath.Join(parent, "standalone")
	if err := os.Mkdir(member, 0755); err != nil {
		t.Fatal(err)
	}
	writeHCL(t, parent, `defaults { risk_policy = "policies/parent" }`)
	selected, err := ResolveRiskPolicy(member, member, "default")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Path != "" || selected.Origin != "" {
		t.Fatalf("zero config must not discover parent selector: %+v", selected)
	}
}

func TestResolveRiskPolicy_GivenRootAsChdir_ShouldParseRootOnlyOnce(t *testing.T) {
	root := t.TempDir()
	writeHCL(t, root, `terraform { bin = "terraform" }
defaults { risk_policy = "policy.rego" }`)
	selected, err := ResolveRiskPolicy(root, root, "prod")
	if err != nil {
		t.Fatalf("root must not be reparsed as Member: %v", err)
	}
	if selected.Path != filepath.Join(root, "policy.rego") || selected.Origin != root {
		t.Fatalf("selection = %+v", selected)
	}
}

func TestResolveRiskPolicy_GivenAbsoluteSelector_ShouldKeepPathAndDeclarationOrigin(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "absolute policy.rego")
	writeHCL(t, root, "defaults { risk_policy = "+`"`+filepath.ToSlash(path)+`"`+" }")
	selected, err := ResolveRiskPolicy(root, root, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Path != path || selected.Origin != root {
		t.Fatalf("selection=%+v", selected)
	}
}

func TestResolveRiskPolicy_GivenInvalidHCL_ShouldFailWithoutInheritedFallback(t *testing.T) {
	tests := []struct{ name, memberConfig string }{
		{"null", `risk_policy = null`}, {"number", `risk_policy = 12`}, {"bool", `risk_policy = true`},
		{"unknown expression", `risk_policy = env.POLICY`}, {"whitespace", `risk_policy = "  "`},
		{"duplicate", "risk_policy = \"first\"\nrisk_policy = \"second\""},
		{"malformed", `risk_policy = "unclosed`},
		{"workspace null", `workspace "prod" { risk_policy = null }`},
		{"member plugin remains forbidden", `plugin "risk" { enabled = false }`},
		{"workspace plugin remains forbidden", `workspace "prod" {
plugin "risk" { enabled = false }
}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			member := filepath.Join(root, "member")
			if err := os.Mkdir(member, 0755); err != nil {
				t.Fatal(err)
			}
			writeHCL(t, root, `defaults { risk_policy = "policies/root" }`)
			writeHCL(t, member, tt.memberConfig)
			_, err := ResolveRiskPolicy(root, member, "prod")
			if err == nil {
				t.Fatal("invalid selection must fail rather than restore root policy")
			}
			if !strings.Contains(err.Error(), "configuration_invalid") {
				t.Fatalf("expected fixed configuration diagnostic, got %v", err)
			}
		})
	}
}

func TestLoadRoot_GivenRiskPolicyDefault_ShouldPreserveExecutionDefaults(t *testing.T) {
	root := t.TempDir()
	writeHCL(t, root, `defaults {
 risk_policy = "policies/root"
 parallelism = 7
 var_file "root.tfvars" {}
 }`)
	loaded, err := LoadRoot(root)
	if err != nil {
		t.Fatalf("dedicated root selector must not prevent execution defaults loading: %v", err)
	}
	resolved := Resolve(loaded, nil, "default")
	if resolved.Parallelism() != 7 || len(resolved.VarFiles()) != 1 || resolved.VarFiles()[0] != "root.tfvars" {
		t.Fatalf("lost root execution defaults: parallelism=%d var_files=%v", resolved.Parallelism(), resolved.VarFiles())
	}
}

func TestResolveRiskPolicy_GivenInvalidRootOrMemberConfiguration_ShouldReportFixedFailure(t *testing.T) {
	tests := []struct{ name, root, member string }{
		{"root syntax", `defaults {`, ""},
		{"duplicate defaults", "defaults {}\ndefaults {}", ""},
		{"defaults label", `defaults "unexpected" {}`, ""},
		{"root null", `defaults { risk_policy = null }`, ""},
		{"root URI", `defaults { risk_policy = "https://example.test/policy.rego" }`, ""},
		{"unknown member attribute", "", `unknown = true`},
		{"workspace label missing", "", `workspace { risk_policy = "x" }`},
		{"workspace plugin label missing", "", `workspace "prod" { plugin {} }`},
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

func TestResolveRiskPolicy_GivenUnreadableConfiguration_ShouldNotRestoreInheritedSelection(t *testing.T) {
	for _, level := range []string{"root", "member"} {
		t.Run(level, func(t *testing.T) {
			project := t.TempDir()
			member := filepath.Join(project, "member")
			if err := os.Mkdir(member, 0755); err != nil {
				t.Fatal(err)
			}
			writeHCL(t, project, `defaults { risk_policy = "root.rego" }`)
			dir := project
			if level == "member" {
				dir = member
			} else if err := os.Remove(filepath.Join(project, HCLConfigFileName)); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(dir, HCLConfigFileName), 0755); err != nil {
				t.Fatal(err)
			}
			_, err := ResolveRiskPolicy(project, member, "prod")
			if err == nil || err.Error() != "configuration_invalid" {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestResolveRiskPolicy_GivenAbsentMemberFile_ShouldInheritRootSelection(t *testing.T) {
	project := t.TempDir()
	writeHCL(t, project, `defaults { risk_policy = "root.rego" }`)
	selection, err := ResolveRiskPolicy(project, filepath.Join(project, "member"), "prod")
	if err != nil || selection.Path != filepath.Join(project, "root.rego") || selection.Origin != project {
		t.Fatalf("selection=%+v error=%v", selection, err)
	}
}

func TestResolveRiskPolicy_GivenDeletedInvocationDirectory_ShouldReportFixedFailure(t *testing.T) {
	project := t.TempDir()
	cwd := filepath.Join(project, "cwd")
	if err := os.Mkdir(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	if err := os.Remove(cwd); err != nil {
		t.Fatal(err)
	}
	for _, dirs := range [][2]string{{".", project}, {project, "."}} {
		_, err := ResolveRiskPolicy(dirs[0], dirs[1], "prod")
		if err == nil || err.Error() != "configuration_invalid" {
			t.Fatalf("error=%v", err)
		}
	}
}

func TestResolveRiskPolicy_GivenMemberExecutionBlocks_ShouldPreservePolicySelection(t *testing.T) {
	project := t.TempDir()
	writeHCL(t, project, "")
	member := filepath.Join(project, "member")
	if err := os.Mkdir(member, 0755); err != nil {
		t.Fatal(err)
	}
	writeHCL(t, member, `risk_policy = "member.rego"
var "region" { value = "us-east-1" }
var_file "vars.tfvars" {}`)
	selection, err := ResolveRiskPolicy(project, member, "prod")
	if err != nil || selection.Path != filepath.Join(member, "member.rego") {
		t.Fatalf("selection=%+v error=%v", selection, err)
	}
}

func TestResolveRiskPolicy_GivenZeroConfigProjectWithChdir_ShouldNotDiscoverChdirConfiguration(t *testing.T) {
	for _, memberConfig := range []string{`risk_policy = "policies/unloaded.rego"`, `risk_policy = "unterminated`} {
		t.Run(memberConfig, func(t *testing.T) {
			project := t.TempDir()
			chdir := filepath.Join(project, "standalone")
			if err := os.Mkdir(chdir, 0755); err != nil {
				t.Fatal(err)
			}
			writeHCL(t, chdir, memberConfig)
			selected, err := ResolveRiskPolicy(project, chdir, "prod")
			if err != nil {
				t.Fatalf("unloaded Chdir configuration must not cause a policy error in zero-config startup: %v", err)
			}
			if selected.Path != "" || selected.Origin != "" {
				t.Fatalf("zero-config startup must use builtin instead of discovering Chdir configuration: %+v", selected)
			}
		})
	}
}
