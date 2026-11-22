package sdk

import (
	"reflect"
	"testing"
)

func TestPlanOptions_WithDefaultTargets(t *testing.T) {
	defaults := []string{"module.networking"}
	tests := []struct {
		name     string
		selected []string
		defaults []string
		want     []string
	}{
		{name: "no selection takes the defaults", defaults: defaults, want: defaults},
		{name: "a selection replaces the defaults", selected: []string{"aws_instance.web"}, defaults: defaults, want: []string{"aws_instance.web"}},
		{name: "neither leaves the plan untargeted", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := PlanOptions{Targets: tt.selected, Parallelism: 3}

			got := opts.WithDefaultTargets(tt.defaults)

			if !reflect.DeepEqual(got.Targets, tt.want) {
				t.Errorf("Targets = %q, want %q", got.Targets, tt.want)
			}
			if got.Parallelism != 3 {
				t.Errorf("Parallelism = %d, want the other fields kept", got.Parallelism)
			}
		})
	}
}
