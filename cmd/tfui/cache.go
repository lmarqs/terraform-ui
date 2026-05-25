package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lmarqs/terraform-ui/internal/source"
	"github.com/lmarqs/terraform-ui/internal/terraform"
)

func seedCache(cache *terraform.ServiceCache, planURI, stateURI string) error {
	if planURI == "-" && stateURI == "-" {
		return fmt.Errorf("stdin (-) can only be used by one flag per invocation; use a file for the other")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	resolver := source.NewResolver(
		&source.LocalProvider{BaseDir: cwd},
		&source.StdinProvider{},
	)
	ctx := context.Background()

	if planURI != "" {
		if planURI == "-" {
			data, resolveErr := resolver.Resolve(ctx, planURI)
			if resolveErr != nil {
				return fmt.Errorf("loading plan: %w", resolveErr)
			}
			if err := cache.SeedPlan("", data); err != nil {
				return fmt.Errorf("parsing plan: %w", err)
			}
		} else {
			planFile, resolveErr := resolveToAbsPath(cwd, planURI)
			if resolveErr != nil {
				return fmt.Errorf("resolving plan path: %w", resolveErr)
			}
			if err := cache.SeedPlan(planFile, nil); err != nil {
				return fmt.Errorf("loading plan: %w", err)
			}
		}
	}

	if stateURI != "" {
		if stateURI == "-" {
			data, resolveErr := resolver.Resolve(ctx, stateURI)
			if resolveErr != nil {
				return fmt.Errorf("loading state: %w", resolveErr)
			}
			if err := cache.SeedState("", data); err != nil {
				return fmt.Errorf("parsing state: %w", err)
			}
		} else {
			stateFile, resolveErr := resolveToAbsPath(cwd, stateURI)
			if resolveErr != nil {
				return fmt.Errorf("resolving state path: %w", resolveErr)
			}
			if err := cache.SeedState(stateFile, nil); err != nil {
				return fmt.Errorf("loading state: %w", err)
			}
		}
	}

	return nil
}

func resolveToAbsPath(baseDir, uri string) (string, error) {
	if filepath.IsAbs(uri) {
		return uri, nil
	}
	clean := uri
	if len(clean) > 2 && clean[:2] == "./" {
		clean = clean[2:]
	}
	if len(clean) > 7 && clean[:7] == "file://" {
		clean = clean[7:]
		if filepath.IsAbs(clean) {
			return clean, nil
		}
	}
	return filepath.Abs(filepath.Join(baseDir, clean))
}
