package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lmarqs/terraform-ui/internal/config"
)

func effectiveWorkDir(cfg config.Config) string {
	if cfg.Chdir != "" {
		return filepath.Join(cfg.Dir, cfg.Chdir)
	}
	return cfg.WorkingDir()
}

func validateChdir(cfg config.Config) error {
	dir := filepath.Join(cfg.Dir, cfg.Chdir)

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("chdir %q not found (resolved to %s)", cfg.Chdir, dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("chdir %q is not a directory (resolved to %s)", cfg.Chdir, dir)
	}
	if !config.HasTerraformFiles(dir) {
		return fmt.Errorf("chdir %q has no .tf files (resolved to %s)", cfg.Chdir, dir)
	}
	return nil
}

func resolveProjectDir(project string) string {
	dir := project

	if dir == "" || dir == "." {
		dir = "."
	} else {
		base := filepath.Base(dir)
		if strings.EqualFold(base, config.HCLConfigFileName) {
			dir = filepath.Dir(dir)
		} else if info, err := os.Stat(dir); err == nil && !info.IsDir() {
			dir = filepath.Dir(dir)
		}
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return dir
	}
	return abs
}
