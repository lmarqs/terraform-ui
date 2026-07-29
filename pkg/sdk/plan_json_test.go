package sdk

import "testing"

func TestActionFromStrings_GivenTerraformActions_ShouldMapToAction(t *testing.T) {
	tests := []struct {
		name    string
		actions []string
		want    Action
	}{
		{"create", []string{"create"}, ActionCreate},
		{"read", []string{"read"}, ActionRead},
		{"update", []string{"update"}, ActionUpdate},
		{"delete", []string{"delete"}, ActionDelete},
		{"no-op", []string{"no-op"}, ActionNoOp},
		{"delete before create", []string{"delete", "create"}, ActionDeleteThenCreate},
		{"create before destroy", []string{"create", "delete"}, ActionCreateThenDelete},
		{"unrecognized action", []string{"forget"}, ActionNoOp},
		{"unrecognized pair", []string{"create", "update"}, ActionNoOp},
		{"empty", nil, ActionNoOp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ActionFromStrings(tt.actions); got != tt.want {
				t.Errorf("ActionFromStrings(%v) = %q, want %q", tt.actions, got, tt.want)
			}
		})
	}
}

func TestPlanJSONChangeCount_GivenPlanDocument_ShouldCountRealChanges(t *testing.T) {
	tests := []struct {
		name string
		data string
		want int
	}{
		{
			name: "create and update count",
			data: `{"resource_changes":[
				{"address":"a","change":{"actions":["create"]}},
				{"address":"b","change":{"actions":["update"]}}
			]}`,
			want: 2,
		},
		{
			name: "replace counts once",
			data: `{"resource_changes":[{"address":"a","change":{"actions":["delete","create"]}}]}`,
			want: 1,
		},
		{
			name: "no-op and data read do not count",
			data: `{"resource_changes":[
				{"address":"a","change":{"actions":["no-op"]}},
				{"address":"d","change":{"actions":["read"]}},
				{"address":"b","change":{"actions":["delete"]}}
			]}`,
			want: 1,
		},
		{
			name: "entry without a change object is skipped",
			data: `{"resource_changes":[{"address":"a"}]}`,
			want: 0,
		},
		{
			name: "empty resource_changes",
			data: `{"format_version":"1.2","resource_changes":[]}`,
			want: 0,
		},
		{
			name: "document without resource_changes",
			data: `{"format_version":"1.2"}`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PlanJSONChangeCount([]byte(tt.data))
			if err != nil {
				t.Fatalf("PlanJSONChangeCount() error = %v, want nil", err)
			}
			if got != tt.want {
				t.Errorf("PlanJSONChangeCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestPlanJSONChangeCount_GivenUnreadableBytes_ShouldError guards the contract
// that matters: a caller must never be able to read "cannot tell" as "clean".
func TestPlanJSONChangeCount_GivenUnreadableBytes_ShouldError(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"truncated", []byte(`{"resource_changes":[`)},
		{"not json", []byte("terraform: command not found")},
		{"empty", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PlanJSONChangeCount(tt.data)
			if err == nil {
				t.Fatalf("PlanJSONChangeCount(%q) error = nil, want error", tt.data)
			}
			if got != 0 {
				t.Errorf("PlanJSONChangeCount(%q) = %d, want 0 alongside the error", tt.data, got)
			}
		})
	}
}
