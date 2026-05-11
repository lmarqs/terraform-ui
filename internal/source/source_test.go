package source

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseScheme(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		scheme    string
		wantError bool
	}{
		// Stdin
		{"stdin dash", "-", "stdin", false},

		// Valid local paths (absolute)
		{"absolute root", "/plan.json", "", false},
		{"absolute nested", "/home/user/infra/plan.json", "", false},
		{"absolute trailing slash", "/dir/", "", false},

		// Valid local paths (relative, explicit prefix required)
		{"relative dot-slash", "./plan.json", "", false},
		{"relative dot-slash nested", "./dir/plan.json", "", false},
		{"relative parent", "../state.json", "", false},
		{"relative parent deep", "../../infra/plan.json", "", false},
		{"relative current dir", "./", "", false},

		// Valid file:// scheme (normalized to local)
		{"file scheme absolute", "file:///home/user/plan.json", "", false},
		{"file scheme relative", "file://./plan.json", "", false},

		// Valid remote schemes
		{"s3", "s3://bucket/key.json", "s3", false},
		{"https", "https://example.com/plan.json", "https", false},
		{"http", "http://localhost:8080/state.json", "http", false},
		{"gcs", "gcs://bucket/state.json", "gcs", false},
		{"custom valid scheme", "myscheme://path", "myscheme", false},
		{"scheme with digits", "s3v2://bucket/key", "s3v2", false},
		{"scheme with hyphen", "my-scheme://path", "my-scheme", false},
		{"scheme with plus", "svn+ssh://host/path", "svn+ssh", false},

		// ERRORS: ambiguous (no explicit prefix, no scheme)
		{"bare filename", "plan.json", "", true},
		{"bare word", "plan", "", true},
		{"relative no prefix", "dir/plan.json", "", true},
		{"bare with extension", "state.tfstate", "", true},
		{"hyphenated filename", "my-plan.json", "", true},
		{"tilde path", "~/plan.json", "", true},

		// ERRORS: invalid schemes
		{"digit-start scheme", "123://path", "", true},
		{"underscore scheme", "my_scheme://path", "", true},
		{"space in scheme", "my scheme://path", "", true},
		{"slash before ://", "path/with://slashes", "", true},

		// ERRORS: empty
		{"empty string", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseScheme(tt.uri)
			if tt.wantError {
				if err == nil {
					t.Errorf("parseScheme(%q) expected error, got scheme=%q", tt.uri, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseScheme(%q) unexpected error: %v", tt.uri, err)
				return
			}
			if got != tt.scheme {
				t.Errorf("parseScheme(%q) = %q, want %q", tt.uri, got, tt.scheme)
			}
		})
	}
}

func TestLocalProvider(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "plan.json"), []byte(`{"changes":[]}`), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "deep.json"), []byte(`{"deep":true}`), 0644)

	t.Run("absolute path", func(t *testing.T) {
		p := &LocalProvider{}
		data, err := p.Read(ctx, filepath.Join(dir, "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"changes":[]}` {
			t.Errorf("got %q", data)
		}
	})

	t.Run("file:// strips prefix", func(t *testing.T) {
		p := &LocalProvider{}
		data, err := p.Read(ctx, "file://"+filepath.Join(dir, "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"changes":[]}` {
			t.Errorf("got %q", data)
		}
	})

	t.Run("relative with BaseDir", func(t *testing.T) {
		p := &LocalProvider{BaseDir: dir}
		data, err := p.Read(ctx, "./plan.json")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"changes":[]}` {
			t.Errorf("got %q", data)
		}
	})

	t.Run("relative nested with BaseDir", func(t *testing.T) {
		p := &LocalProvider{BaseDir: dir}
		data, err := p.Read(ctx, "./sub/deep.json")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"deep":true}` {
			t.Errorf("got %q", data)
		}
	})

	t.Run("absolute ignores BaseDir", func(t *testing.T) {
		p := &LocalProvider{BaseDir: "/wrong/path"}
		data, err := p.Read(ctx, filepath.Join(dir, "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"changes":[]}` {
			t.Errorf("got %q", data)
		}
	})

	t.Run("error: file not found", func(t *testing.T) {
		p := &LocalProvider{BaseDir: dir}
		_, err := p.Read(ctx, "./missing.json")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("error: empty path", func(t *testing.T) {
		p := &LocalProvider{BaseDir: dir}
		_, err := p.Read(ctx, "")
		if err == nil {
			t.Error("expected error for empty path")
		}
	})
}

func TestResolver(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "plan.json"), []byte("plan data"), 0644)

	ctx := context.Background()

	t.Run("resolves absolute path", func(t *testing.T) {
		r := NewResolver(&LocalProvider{})
		data, err := r.Resolve(ctx, filepath.Join(dir, "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "plan data" {
			t.Errorf("got %q", data)
		}
	})

	t.Run("resolves relative path", func(t *testing.T) {
		r := NewResolver(&LocalProvider{BaseDir: dir})
		data, err := r.Resolve(ctx, "./plan.json")
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "plan data" {
			t.Errorf("got %q", data)
		}
	})

	t.Run("resolves file:// to local", func(t *testing.T) {
		r := NewResolver(&LocalProvider{})
		data, err := r.Resolve(ctx, "file://"+filepath.Join(dir, "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "plan data" {
			t.Errorf("got %q", data)
		}
	})

	t.Run("error: unsupported scheme", func(t *testing.T) {
		r := NewResolver(&LocalProvider{})
		_, err := r.Resolve(ctx, "s3://bucket/key.json")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("error: ambiguous bare filename", func(t *testing.T) {
		r := NewResolver(&LocalProvider{BaseDir: dir})
		_, err := r.Resolve(ctx, "plan.json")
		if err == nil {
			t.Error("expected error for ambiguous bare filename")
		}
	})

	t.Run("error: no providers", func(t *testing.T) {
		r := NewResolver()
		_, err := r.Resolve(ctx, "/some/path.json")
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("register adds provider", func(t *testing.T) {
		r := NewResolver()
		r.Register(&LocalProvider{})
		data, err := r.Resolve(ctx, filepath.Join(dir, "plan.json"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "plan data" {
			t.Errorf("got %q", data)
		}
	})

	t.Run("stdin requires provider", func(t *testing.T) {
		r := NewResolver(&LocalProvider{})
		_, err := r.Resolve(ctx, "-")
		if err == nil {
			t.Error("expected error: stdin provider not registered")
		}
	})

	t.Run("stdin dispatch", func(t *testing.T) {
		// Can't test actual stdin reading without OS pipe,
		// but verify scheme dispatch works
		r := NewResolver(&LocalProvider{}, &StdinProvider{})
		// StdinProvider.Read will fail because stdin is not a pipe in tests
		_, err := r.Resolve(ctx, "-")
		if err == nil {
			t.Skip("stdin unexpectedly had data")
		}
		// Error should be from StdinProvider, not from scheme resolution
		if err.Error() == `unsupported URI "-"` {
			t.Error("should dispatch to stdin provider, not fail on scheme")
		}
	})
}

func TestStdinProvider(t *testing.T) {
	p := &StdinProvider{}

	if p.Scheme() != "stdin" {
		t.Errorf("scheme = %q, want 'stdin'", p.Scheme())
	}

	t.Run("consumed flag prevents double read", func(t *testing.T) {
		p := &StdinProvider{consumed: true}
		_, err := p.Read(context.Background(), "-")
		if err == nil {
			t.Error("expected error for double read")
		}
	})
}
