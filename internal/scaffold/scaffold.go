package scaffold

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/config"
)

// DetectBinary finds the first available terraform binary on PATH.
func DetectBinary() string {
	for _, bin := range []string{"terraform", "tofu", "terragrunt"} {
		if _, err := exec.LookPath(bin); err == nil {
			return bin
		}
	}
	return "terraform"
}

// DetectedMember represents a terraform directory found during scanning.
type DetectedMember struct {
	Path    string
	Enabled bool
}

// DetectMembers scans the directory for terraform project directories.
func DetectMembers(dir string) []DetectedMember {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}

	candidates := []string{
		"modules/*",
		"envs/*",
		"infra/*",
		"services/*/terraform",
	}

	var members []DetectedMember
	for _, candidate := range candidates {
		fullPattern := filepath.Join(absDir, candidate)
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if config.HasTerraformFiles(match) {
				rel, _ := filepath.Rel(absDir, match)
				members = append(members, DetectedMember{
					Path:    rel,
					Enabled: true,
				})
			}
		}
	}

	if config.HasTerraformFiles(absDir) {
		members = append(members, DetectedMember{
			Path:    ".",
			Enabled: true,
		})
	}

	return members
}

// GenerateConfig runs the detection logic non-interactively and returns HCL content.
func GenerateConfig(dir string) (string, error) {
	binary := DetectBinary()
	members := DetectMembers(dir)

	var enabled []string
	for _, m := range members {
		if m.Enabled {
			enabled = append(enabled, m.Path)
		}
	}

	return BuildHCL(binary, enabled), nil
}

// BuildHCL generates the tfui.hcl content from a binary name and member paths.
func BuildHCL(binary string, members []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "terraform {\n  bin = %q\n}\n", binary)

	if len(members) > 0 {
		sort.Strings(members)
		for _, m := range members {
			fmt.Fprintf(&b, "\nmember %q {}\n", m)
		}
	}

	return b.String()
}
