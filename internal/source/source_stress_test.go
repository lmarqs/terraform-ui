package source

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseSchemeAdversarial(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		scheme    string
		wantError bool
	}{
		// Invalid schemes → error
		{"digit start", "123://path", "", true},
		{"underscore", "my_scheme://path", "", true},
		{"space", "my scheme://path", "", true},
		{"slash in scheme", "path/with://slashes", "", true},
		{"at sign", "user@host://path", "", true},
		{"exclamation", "alert!://path", "", true},
		{"empty before ://", "://path", "", true},
		{"unicode scheme", "ñ://path", "", true},
		{"emoji scheme", "📋://path", "", true},

		// Colon but no "://"
		{"bare colon", "key:value", "", true},
		{"colon slash", "s3:/bucket", "", true},
		{"double colon", "host::port", "", true},
		{"colon end", "scheme:", "", true},
		{"multiple colons", "a:b:c", "", true},
		{"port number", "localhost:8080", "", true},

		// Stdin
		{"stdin dash", "-", "stdin", false},
		{"double dash not stdin", "--", "", true},
		{"dash prefix", "-f", "", true},

		// Ambiguous paths (no ./ or / prefix) → error
		{"bare word", "plan", "", true},
		{"bare filename", "plan.json", "", true},
		{"bare nested", "dir/file.json", "", true},
		{"tilde path", "~/plan.json", "", true},
		{"windows path", "C:\\Users\\plan.json", "", true},
		{"just dash is stdin", "-", "stdin", false},
		{"just dot", ".", "", true},
		{"just double dot", "..", "", true},
		{"dot no slash", ".plan.json", "", true},
		{"double dot no slash", "..plan.json", "", true},

		// Valid: absolute paths
		{"absolute simple", "/plan.json", "", false},
		{"absolute deep", "/a/b/c/d.json", "", false},
		{"absolute root", "/", "", false},

		// Valid: explicit relative
		{"dot-slash", "./plan.json", "", false},
		{"dot-slash nested", "./a/b.json", "", false},
		{"parent relative", "../plan.json", "", false},
		{"parent deep", "../../a/b.json", "", false},

		// Valid: known schemes
		{"s3", "s3://bucket/key", "s3", false},
		{"https", "https://host/path", "https", false},
		{"http", "http://host/path", "http", false},
		{"gcs", "gcs://bucket/key", "gcs", false},
		{"file normalized", "file:///path", "", false},

		// Valid scheme edge cases
		{"single char", "x://path", "x", false},
		{"all valid chars", "a1+b2-c3.d4://path", "a1+b2-c3.d4", false},
		{"uppercase", "HTTP://host/path", "HTTP", false},

		// Empty and whitespace → error
		{"empty", "", "", true},
		{"whitespace", "   ", "", true},
		{"tab", "\t", "", true},
		{"newline", "\n", "", true},
		{"null byte", "\x00", "", true},

		// Long inputs
		{"long valid scheme", strings.Repeat("a", 100) + "://path", strings.Repeat("a", 100), false},
		{"long absolute path", "/" + strings.Repeat("dir/", 100) + "file.json", "", false},
		{"long ambiguous", strings.Repeat("a", 1000), "", true},

		// Potential injection
		{"null in path", "./plan\x00.json", "", false},
		{"newline in path", "./plan\n.json", "", false},
		{"semicolons", "./a;rm -rf /", "", false},
		{"backticks", "./`cmd`.json", "", false},
		{"dollar expansion", "./$HOME/plan.json", "", false},
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

func TestLocalProviderSecurity(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	os.WriteFile(filepath.Join(dir, "safe.json"), []byte("safe"), 0644)

	p := &LocalProvider{BaseDir: dir}

	t.Run("path traversal reads parent", func(t *testing.T) {
		// Create file outside base
		parent := filepath.Dir(dir)
		outside := filepath.Join(parent, "outside.json")
		os.WriteFile(outside, []byte("outside"), 0644)
		defer os.Remove(outside)

		// Path traversal works (provider is not a sandbox — security is at CLI layer)
		data, err := p.Read(ctx, "../outside.json")
		if err != nil {
			t.Skip("path traversal blocked by OS")
		}
		if string(data) != "outside" {
			t.Errorf("got %q", data)
		}
	})

	t.Run("null byte in filename", func(t *testing.T) {
		_, err := p.Read(ctx, "./safe\x00.json")
		if err == nil {
			t.Error("expected error: null byte should cause OS error")
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		restricted := filepath.Join(dir, "restricted.json")
		os.WriteFile(restricted, []byte("secret"), 0000)
		defer os.Chmod(restricted, 0644)

		_, err := p.Read(ctx, "./restricted.json")
		if err == nil {
			t.Error("expected error for permission denied")
		}
	})

	t.Run("directory not file", func(t *testing.T) {
		os.Mkdir(filepath.Join(dir, "adir"), 0755)
		_, err := p.Read(ctx, "./adir")
		if err == nil {
			t.Error("expected error reading directory")
		}
	})

	t.Run("broken symlink", func(t *testing.T) {
		link := filepath.Join(dir, "broken-link")
		os.Symlink("/nonexistent/target", link)
		_, err := p.Read(ctx, "./broken-link")
		if err == nil {
			t.Error("expected error for broken symlink")
		}
	})
}

func TestResolverConcurrency(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.json"), []byte("data"), 0644)

	r := NewResolver(&LocalProvider{BaseDir: dir})
	ctx := context.Background()

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			r.Resolve(ctx, "./test.json")
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestResolverErrorMessages(t *testing.T) {
	r := NewResolver(&LocalProvider{})
	ctx := context.Background()

	t.Run("ambiguous URI mentions fix", func(t *testing.T) {
		_, err := r.Resolve(ctx, "plan.json")
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "./plan.json") {
			t.Errorf("error should suggest ./plan.json, got: %s", msg)
		}
	})

	t.Run("unsupported scheme lists available", func(t *testing.T) {
		_, err := r.Resolve(ctx, "s3://bucket/key")
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "local paths") {
			t.Errorf("error should mention available providers, got: %s", msg)
		}
	})

	t.Run("invalid scheme is descriptive", func(t *testing.T) {
		_, err := r.Resolve(ctx, "123://path")
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "scheme must start with a letter") {
			t.Errorf("error should explain RFC rule, got: %s", msg)
		}
	})

	t.Run("empty URI", func(t *testing.T) {
		_, err := r.Resolve(ctx, "")
		if err == nil {
			t.Fatal("expected error")
		}
		msg := err.Error()
		if !strings.Contains(msg, "empty") {
			t.Errorf("error should say empty, got: %s", msg)
		}
	})
}
