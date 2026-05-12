package source

import (
	"context"
	"fmt"
	"strings"
)

// Provider reads raw bytes from a URI scheme.
type Provider interface {
	Scheme() string
	Read(ctx context.Context, uri string) ([]byte, error)
}

// Resolver dispatches URIs to the appropriate Provider based on explicit scheme.
//
// Resolution rules (no guessing, no heuristics):
//   - Starts with "/" → local absolute path
//   - Starts with "./" or "../" → local relative path
//   - Has valid "scheme://" prefix → dispatches to registered scheme provider
//   - Anything else → error (ambiguous, user must be explicit)
type Resolver struct {
	providers map[string]Provider
}

// NewResolver creates a Resolver with the given providers.
func NewResolver(providers ...Provider) *Resolver {
	r := &Resolver{providers: make(map[string]Provider)}
	for _, p := range providers {
		r.providers[p.Scheme()] = p
	}
	return r
}

// Register adds a provider to the resolver.
func (r *Resolver) Register(p Provider) {
	r.providers[p.Scheme()] = p
}

// Resolve reads bytes from the given URI.
func (r *Resolver) Resolve(ctx context.Context, uri string) ([]byte, error) {
	scheme, err := parseScheme(uri)
	if err != nil {
		return nil, err
	}

	p, ok := r.providers[scheme]
	if !ok {
		supported := make([]string, 0, len(r.providers))
		for s := range r.providers {
			switch s {
			case "":
				supported = append(supported, "local paths (./ or /)")
			case "stdin":
				supported = append(supported, "- (stdin)")
			default:
				supported = append(supported, s+"://")
			}
		}
		return nil, fmt.Errorf("unsupported URI %q (supported: %s)", uri, strings.Join(supported, ", "))
	}

	return p.Read(ctx, uri)
}

// parseScheme determines the provider scheme for a URI.
// Returns "" for local paths, "stdin" for "-", the scheme string for URIs, or an error for ambiguous input.
func parseScheme(uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("empty URI")
	}

	// Stdin: literal "-" only
	if uri == "-" {
		return "stdin", nil
	}

	// Absolute local path
	if strings.HasPrefix(uri, "/") {
		return "", nil
	}

	// Relative local path (must be explicit)
	if strings.HasPrefix(uri, "./") || strings.HasPrefix(uri, "../") {
		return "", nil
	}

	// Explicit scheme
	if idx := strings.Index(uri, "://"); idx > 0 {
		scheme := uri[:idx]
		if !isValidScheme(scheme) {
			return "", fmt.Errorf("invalid URI scheme in %q: scheme must start with a letter and contain only [a-z0-9+.-]", uri)
		}
		if scheme == "file" {
			return "", nil
		}
		return scheme, nil
	}

	// Anything else is ambiguous
	return "", fmt.Errorf(
		"ambiguous URI %q: use explicit path (./%s or /absolute/path) or scheme (file://)",
		uri, uri,
	)
}

// isValidScheme checks RFC 3986: scheme = ALPHA *( ALPHA / DIGIT / "+" / "-" / "." )
func isValidScheme(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, c := range s {
		if i == 0 {
			if !isAlpha(c) {
				return false
			}
		} else {
			if !isAlpha(c) && !isDigit(c) && c != '+' && c != '-' && c != '.' {
				return false
			}
		}
	}
	return true
}

func isAlpha(c rune) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isDigit(c rune) bool { return c >= '0' && c <= '9' }
