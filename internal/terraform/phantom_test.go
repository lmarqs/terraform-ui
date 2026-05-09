package terraform

import "testing"

func TestIsPhantomChange(t *testing.T) {
	tests := []struct {
		name     string
		change   PlanChange
		expected bool
	}{
		{
			name: "identical before/after JSON detected as phantom",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "ingress",
						OldValue: `[{"port":80},{"port":443}]`,
						NewValue: `[{"port":80},{"port":443}]`,
					},
				},
			},
			expected: true,
		},
		{
			name: "different before/after detected as real",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "ami",
						OldValue: `"ami-old"`,
						NewValue: `"ami-new"`,
					},
				},
			},
			expected: false,
		},
		{
			name: "array ordering difference treated as phantom",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "tags",
						OldValue: `[{"key":"Name","value":"web"},{"key":"Env","value":"prod"}]`,
						NewValue: `[{"key":"Env","value":"prod"},{"key":"Name","value":"web"}]`,
					},
				},
			},
			expected: true,
		},
		{
			name: "null fields ignored in comparison",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "config",
						OldValue: `{"tags":{"Name":"rt"},"propagating_vgws":null}`,
						NewValue: `{"tags":{"Name":"rt"}}`,
					},
				},
			},
			expected: true,
		},
		{
			name: "nested object difference detected as real",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "environment",
						OldValue: `{"variables":{"KEY":"old"}}`,
						NewValue: `{"variables":{"KEY":"new"}}`,
					},
				},
			},
			expected: false,
		},
		{
			name: "non-update action returns false",
			change: PlanChange{
				Action: ActionCreate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "name",
						OldValue: `"same"`,
						NewValue: `"same"`,
					},
				},
			},
			expected: false,
		},
		{
			name: "no attribute diffs returns false",
			change: PlanChange{
				Action:         ActionUpdate,
				AttributeDiffs: []AttributeDiff{},
			},
			expected: false,
		},
		{
			name: "non-JSON values compared as strings - same",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "description",
						OldValue: "plain text value",
						NewValue: "plain text value",
					},
				},
			},
			expected: true,
		},
		{
			name: "non-JSON values compared as strings - different",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "description",
						OldValue: "old text",
						NewValue: "new text",
					},
				},
			},
			expected: false,
		},
		{
			name: "multiple diffs all phantom",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "tags",
						OldValue: `{"a":"1","b":"2"}`,
						NewValue: `{"b":"2","a":"1"}`,
					},
					{
						Key:      "list",
						OldValue: `[1,2,3]`,
						NewValue: `[1,2,3]`,
					},
				},
			},
			expected: true,
		},
		{
			name: "one real diff among phantom diffs makes it real",
			change: PlanChange{
				Action: ActionUpdate,
				AttributeDiffs: []AttributeDiff{
					{
						Key:      "tags",
						OldValue: `{"a":"1"}`,
						NewValue: `{"a":"1"}`,
					},
					{
						Key:      "ami",
						OldValue: `"ami-old"`,
						NewValue: `"ami-new"`,
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPhantomChange(&tt.change)
			if result != tt.expected {
				t.Errorf("IsPhantomChange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDetectPhantomChanges(t *testing.T) {
	tests := []struct {
		name             string
		changes          []PlanChange
		wantPhantom      int
		wantReal         int
		wantAddresses    []string
		wantIsPhantomSet []bool
	}{
		{
			name:             "empty changes",
			changes:          []PlanChange{},
			wantPhantom:      0,
			wantReal:         0,
			wantAddresses:    []string{},
			wantIsPhantomSet: []bool{},
		},
		{
			name: "single phantom change",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "aws_security_group.default"},
					Action:   ActionUpdate,
					AttributeDiffs: []AttributeDiff{
						{Key: "name", OldValue: `"sg"`, NewValue: `"sg"`},
					},
				},
			},
			wantPhantom:      1,
			wantReal:         0,
			wantAddresses:    []string{"aws_security_group.default"},
			wantIsPhantomSet: []bool{true},
		},
		{
			name: "single real change",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "aws_instance.web"},
					Action:   ActionUpdate,
					AttributeDiffs: []AttributeDiff{
						{Key: "ami", OldValue: `"ami-old"`, NewValue: `"ami-new"`},
					},
				},
			},
			wantPhantom:      0,
			wantReal:         1,
			wantAddresses:    []string{},
			wantIsPhantomSet: []bool{false},
		},
		{
			name: "mixed phantom and real changes",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "aws_security_group.default"},
					Action:   ActionUpdate,
					AttributeDiffs: []AttributeDiff{
						{Key: "name", OldValue: `"sg"`, NewValue: `"sg"`},
					},
				},
				{
					Resource: Resource{Address: "aws_instance.web"},
					Action:   ActionUpdate,
					AttributeDiffs: []AttributeDiff{
						{Key: "ami", OldValue: `"ami-old"`, NewValue: `"ami-new"`},
					},
				},
				{
					Resource: Resource{Address: "aws_route_table.main"},
					Action:   ActionUpdate,
					AttributeDiffs: []AttributeDiff{
						{Key: "routes", OldValue: `[{"cidr":"10.0.0.0/8"}]`, NewValue: `[{"cidr":"10.0.0.0/8"}]`},
					},
				},
			},
			wantPhantom:      2,
			wantReal:         1,
			wantAddresses:    []string{"aws_security_group.default", "aws_route_table.main"},
			wantIsPhantomSet: []bool{true, false, true},
		},
		{
			name: "non-update actions are excluded from phantom detection",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "aws_instance.new"},
					Action:   ActionCreate,
					AttributeDiffs: []AttributeDiff{
						{Key: "ami", OldValue: `""`, NewValue: `"ami-123"`},
					},
				},
				{
					Resource: Resource{Address: "aws_instance.old"},
					Action:   ActionDelete,
					AttributeDiffs: []AttributeDiff{
						{Key: "ami", OldValue: `"ami-456"`, NewValue: `""`},
					},
				},
				{
					Resource: Resource{Address: "aws_security_group.default"},
					Action:   ActionUpdate,
					AttributeDiffs: []AttributeDiff{
						{Key: "name", OldValue: `"sg"`, NewValue: `"sg"`},
					},
				},
			},
			wantPhantom:      1,
			wantReal:         0,
			wantAddresses:    []string{"aws_security_group.default"},
			wantIsPhantomSet: []bool{false, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectPhantomChanges(tt.changes)

			if result.PhantomCount != tt.wantPhantom {
				t.Errorf("PhantomCount = %d, want %d", result.PhantomCount, tt.wantPhantom)
			}
			if result.RealCount != tt.wantReal {
				t.Errorf("RealCount = %d, want %d", result.RealCount, tt.wantReal)
			}
			if len(result.PhantomAddresses) != len(tt.wantAddresses) {
				t.Fatalf("PhantomAddresses length = %d, want %d", len(result.PhantomAddresses), len(tt.wantAddresses))
			}
			for i, addr := range result.PhantomAddresses {
				if addr != tt.wantAddresses[i] {
					t.Errorf("PhantomAddresses[%d] = %q, want %q", i, addr, tt.wantAddresses[i])
				}
			}
			for i, wantPhantom := range tt.wantIsPhantomSet {
				if i < len(tt.changes) && tt.changes[i].IsPhantom != wantPhantom {
					t.Errorf("changes[%d].IsPhantom = %v, want %v", i, tt.changes[i].IsPhantom, wantPhantom)
				}
			}
		})
	}
}

func TestNormalizedEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     interface{}
		expected bool
	}{
		{
			name:     "identical maps",
			a:        map[string]interface{}{"key": "value"},
			b:        map[string]interface{}{"key": "value"},
			expected: true,
		},
		{
			name:     "map with null field vs without",
			a:        map[string]interface{}{"key": "value", "extra": nil},
			b:        map[string]interface{}{"key": "value"},
			expected: true,
		},
		{
			name:     "arrays with different order",
			a:        []interface{}{"b", "a"},
			b:        []interface{}{"a", "b"},
			expected: true,
		},
		{
			name:     "different values",
			a:        map[string]interface{}{"key": "old"},
			b:        map[string]interface{}{"key": "new"},
			expected: false,
		},
		{
			name:     "nil values",
			a:        nil,
			b:        nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizedEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("NormalizedEqual() = %v, want %v", result, tt.expected)
			}
		})
	}
}
