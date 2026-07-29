//go:build integration

package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestPluginConfig_Targets_ShouldScopeThePlan covers the documented
// `plugin "plan" { targets = [...] }` option, which used to parse and then
// change nothing at all (lmarqs/terraform-ui#57).
func TestPluginConfig_Targets_ShouldScopeThePlan(t *testing.T) {
	project := copyFixtureWithoutInit(t, "plugin-config")
	initInDir(t, filepath.Join(project, "envs", "demo"))

	tests := []struct {
		name    string
		args    []string
		want    string
		notWant string
	}{
		{
			name:    "config targets scope the plan",
			args:    nil,
			want:    "terraform_data.scoped",
			notWant: "terraform_data.unscoped",
		},
		{
			// A project default that an explicit flag cannot narrow would be the
			// wrong shape for targeting.
			name:    "an explicit --target replaces the config default",
			args:    []string{"-target", "terraform_data.unscoped"},
			want:    "terraform_data.unscoped",
			notWant: "terraform_data.scoped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"plan", "-project", project, "-chdir", "envs/demo", "-ci"}, tt.args...)
			stdout, stderr, err := runTfui(args...)
			if err != nil && !isExitCode(err, 2) {
				t.Fatalf("plan failed: %v\nstderr: %s", err, stderr)
			}

			if !strings.Contains(stdout, tt.want) {
				t.Errorf("expected %q in the plan, got: %q", tt.want, stdout)
			}
			if strings.Contains(stdout, tt.notWant) {
				t.Errorf("plan listed %q, which the target scope excludes: %q", tt.notWant, stdout)
			}
			if !strings.Contains(stdout, "1 to add") {
				t.Errorf("expected '1 to add' in the summary, got: %q", stdout)
			}
		})
	}
}

// TestPluginConfig_Disabled_ShouldRefuseTheCommand keeps `enabled = false` from
// producing a silent success: the plugin is absent from the registry, so there
// is nothing to run.
func TestPluginConfig_Disabled_ShouldRefuseTheCommand(t *testing.T) {
	project := copyFixtureWithoutInit(t, "plugin-config")
	initInDir(t, filepath.Join(project, "envs", "demo"))

	stdout, stderr, err := runTfui("state", "-project", project, "-chdir", "envs/demo", "-ci")
	if err == nil {
		t.Fatalf("expected a failure for a disabled plugin\nstdout: %q\nstderr: %q", stdout, stderr)
	}
	if !strings.Contains(stderr, "disabled in tfui.hcl") {
		t.Errorf("expected the error to name the disabled plugin, got: %q", stderr)
	}
}
