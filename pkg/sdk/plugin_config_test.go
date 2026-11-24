package sdk

import (
	"slices"
	"testing"
)

func TestConfigStrings_GivenPluginOptions_ShouldReadTheList(t *testing.T) {
	def := []string{"fallback"}
	tests := []struct {
		name string
		cfg  map[string]interface{}
		want []string
	}{
		{name: "nil config falls back", cfg: nil, want: def},
		{name: "absent key falls back", cfg: map[string]interface{}{"other": []string{"x"}}, want: def},
		{
			name: "string slice is returned",
			cfg:  map[string]interface{}{"targets": []string{"module.a", "module.b"}},
			want: []string{"module.a", "module.b"},
		},
		{
			name: "interface slice keeps only strings",
			cfg:  map[string]interface{}{"targets": []interface{}{"module.a", 3, "module.b"}},
			want: []string{"module.a", "module.b"},
		},
		{name: "empty list is not a fallback", cfg: map[string]interface{}{"targets": []string{}}, want: []string{}},
		{name: "wrong shape falls back", cfg: map[string]interface{}{"targets": "module.a"}, want: def},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConfigStrings(tt.cfg, "targets", def); !slices.Equal(got, tt.want) {
				t.Errorf("ConfigStrings() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlanTargets_GivenPlanOptions_ShouldReadTheTargetsList(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]interface{}
		want    []string
	}{
		{name: "nil options have no targets", options: nil, want: nil},
		{name: "other options have no targets", options: map[string]interface{}{"other": []string{"x"}}, want: nil},
		{name: "targets list is read", options: map[string]interface{}{"targets": []interface{}{"module.a", "module.b"}}, want: []string{"module.a", "module.b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PlanTargets(tt.options); !slices.Equal(got, tt.want) {
				t.Errorf("PlanTargets() = %q, want %q", got, tt.want)
			}
		})
	}
}
