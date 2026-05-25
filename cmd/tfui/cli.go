package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/config"
)

// resolveProjectDir resolves the --project flag value to an absolute directory path.
// Accepts:
//   - A directory path: resolved to absolute
//   - A path to tfui.yaml: uses its parent directory
//   - A path ending in tfui.yaml that doesn't exist yet: uses parent directory
func resolveProjectDir(project string) string {
	dir := project

	if dir == "" || dir == "." {
		dir = "."
	} else {
		// If it points to a file (or looks like it points to tfui.yaml), use parent dir
		base := filepath.Base(dir)
		if strings.EqualFold(base, config.HCLConfigFileName) {
			dir = filepath.Dir(dir)
		} else if info, err := os.Stat(dir); err == nil && !info.IsDir() {
			// If it's an existing file (not a directory), use parent
			dir = filepath.Dir(dir)
		}
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}
