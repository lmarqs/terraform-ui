package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalProvider reads bytes from local filesystem paths and file:// URIs.
// Relative paths are resolved against BaseDir (typically the process CWD).
type LocalProvider struct {
	BaseDir string
}

func (p *LocalProvider) Scheme() string { return "" }

func (p *LocalProvider) Read(_ context.Context, uri string) ([]byte, error) {
	path := strings.TrimPrefix(uri, "file://")

	if !filepath.IsAbs(path) && p.BaseDir != "" {
		path = filepath.Join(p.BaseDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading local file %q: %w", path, err)
	}
	return data, nil
}
