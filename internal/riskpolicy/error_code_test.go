package riskpolicy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
)

func TestLoadSource_GivenFailure_ShouldCarryErrorCodeWithoutParsingText(t *testing.T) {
	tests := []struct {
		name string
		load func(t *testing.T, dir string) error
		want sdk.ErrorCode
	}{
		{"missing path", func(t *testing.T, dir string) error {
			_, err := LoadSource(config.RiskPolicySelection{Path: filepath.Join(dir, "missing.rego")}, nil)
			return err
		}, sdk.ErrorSourceUnreadable},
		{"non rego file", func(t *testing.T, dir string) error {
			path := filepath.Join(dir, "policy.json")
			writePolicyModule(t, path, "{}")
			_, err := LoadSource(config.RiskPolicySelection{Path: path}, nil)
			return err
		}, sdk.ErrorSourceInvalid},
		{"empty builtin", func(t *testing.T, dir string) error {
			_, err := LoadSource(config.RiskPolicySelection{}, nil)
			return err
		}, sdk.ErrorSourceInvalid},
		{"edit between reads", func(t *testing.T, dir string) error {
			path := filepath.Join(dir, "main.rego")
			writePolicyModule(t, path, "package original\n")
			reads := 0
			reader := sourceReader{stat: os.Stat, readDir: os.ReadDir, readFile: func(p string) ([]byte, error) {
				reads++
				if reads == 2 {
					writePolicyModule(t, p, "package changed\n")
				}
				return os.ReadFile(p)
			}}
			_, err := reader.load(config.RiskPolicySelection{Path: path}, nil)
			return err
		}, sdk.ErrorSourceChanged},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.load(t, t.TempDir())
			var code sdk.ErrorCode
			if !errors.As(err, &code) || code != tt.want {
				t.Fatalf("code = %q (error %v), want %q", code, err, tt.want)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("errors.Is(%v, %q) = false", err, tt.want)
			}
			if err.Error() != string(tt.want) {
				t.Fatalf("text = %q, want %q unchanged", err.Error(), tt.want)
			}
		})
	}
}
