package main

import (
	"testing"

	"github.com/lmarqs/terraform-ui/internal/config"
	tfuiplan "github.com/lmarqs/terraform-ui/plugins/plan"
	"github.com/spf13/cobra"
)

// TestBuildPlanCommand_WhenTargetCarriesQuotesAndCommas_ShouldBindVerbatim
// pins the flag binding to StringArrayVar: terraform addresses are not CSV
// records, so an indexed address must reach the plugin exactly as typed
// (lmarqs/terraform-ui#59).
func TestBuildPlanCommand_WhenTargetCarriesQuotesAndCommas_ShouldBindVerbatim(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "indexed address with a space",
			args: []string{`--target=module.a["x y"]`},
			want: []string{`module.a["x y"]`},
		},
		{
			name: "index key containing a comma stays one address",
			args: []string{`--target=module.a["x,y"]`},
			want: []string{`module.a["x,y"]`},
		},
		{
			name: "repeated flags accumulate in order",
			args: []string{"--target=aws_instance.web", "--target=module.db"},
			want: []string{"aws_instance.web", "module.db"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{cfg: config.Config{}}
			var captured tfuiplan.Input
			c := buildPlanCommand(session)
			c.RunE = func(_ *cobra.Command, _ []string) error {
				captured = tfuiplan.Input{Targets: mustStringArray(c, "target")}
				return nil
			}

			c.SetArgs(tt.args)
			if err := c.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			assertStrings(t, "Input.Targets", captured.Targets, tt.want)
		})
	}
}

func assertStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %q, want %q", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %q, want %q", label, got, want)
		}
	}
}
