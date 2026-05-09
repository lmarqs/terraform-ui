package terraform

import "testing"

func TestExtractModule(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "root resource has no module",
			address:  "aws_instance.web",
			expected: "",
		},
		{
			name:     "single module prefix",
			address:  "module.vpc.aws_subnet.main",
			expected: "module.vpc",
		},
		{
			name:     "nested modules fully preserved",
			address:  "module.vpc.module.subnets.aws_subnet.a",
			expected: "module.vpc.module.subnets",
		},
		{
			name:     "deeply nested modules",
			address:  "module.a.module.b.module.c.aws_instance.web",
			expected: "module.a.module.b.module.c",
		},
		{
			name:     "indexed resource in module",
			address:  "module.ecs.aws_ecs_service.svc[0]",
			expected: "module.ecs",
		},
		{
			name:     "indexed resource in nested module",
			address:  "module.vpc.module.subnets.aws_subnet.private[1]",
			expected: "module.vpc.module.subnets",
		},
		{
			name:     "data source has no module",
			address:  "data.aws_ami.latest",
			expected: "",
		},
		{
			name:     "module keyword alone at end",
			address:  "module.vpc",
			expected: "module.vpc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractModule(tt.address)
			if result != tt.expected {
				t.Errorf("ExtractModule(%q) = %q, want %q", tt.address, result, tt.expected)
			}
		})
	}
}

func TestGroupByModule(t *testing.T) {
	tests := []struct {
		name           string
		changes        []PlanChange
		expectedGroups map[string]ActionSummary
		expectedCounts map[string]int
	}{
		{
			name:           "empty changes returns no groups",
			changes:        []PlanChange{},
			expectedGroups: map[string]ActionSummary{},
			expectedCounts: map[string]int{},
		},
		{
			name: "root resources grouped under root",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "aws_instance.web", Module: ""},
					Action:   ActionCreate,
				},
				{
					Resource: Resource{Address: "aws_iam_role.old", Module: ""},
					Action:   ActionDelete,
				},
			},
			expectedGroups: map[string]ActionSummary{
				"root": {Add: 1, Destroy: 1},
			},
			expectedCounts: map[string]int{
				"root": 2,
			},
		},
		{
			name: "single module prefix extracted",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "module.vpc.aws_subnet.private", Module: "module.vpc"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "module.vpc.aws_route_table.private", Module: "module.vpc"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "module.vpc.aws_nat_gateway.main", Module: "module.vpc"},
					Action:   ActionCreate,
				},
			},
			expectedGroups: map[string]ActionSummary{
				"module.vpc": {Add: 1, Change: 2},
			},
			expectedCounts: map[string]int{
				"module.vpc": 3,
			},
		},
		{
			name: "nested modules fully preserved",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "module.vpc.module.subnets.aws_subnet.private[0]", Module: "module.vpc.module.subnets"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "module.vpc.module.subnets.aws_subnet.private[1]", Module: "module.vpc.module.subnets"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "module.vpc.aws_vpc.main", Module: "module.vpc"},
					Action:   ActionUpdate,
				},
			},
			expectedGroups: map[string]ActionSummary{
				"module.vpc.module.subnets": {Change: 2},
				"module.vpc":                {Change: 1},
			},
			expectedCounts: map[string]int{
				"module.vpc.module.subnets": 2,
				"module.vpc":                1,
			},
		},
		{
			name: "mixed root and module resources",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "aws_instance.web", Module: ""},
					Action:   ActionCreate,
				},
				{
					Resource: Resource{Address: "module.vpc.aws_subnet.a", Module: "module.vpc"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "module.vpc.aws_subnet.b", Module: "module.vpc"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "aws_iam_role.old", Module: ""},
					Action:   ActionDelete,
				},
			},
			expectedGroups: map[string]ActionSummary{
				"root":       {Add: 1, Destroy: 1},
				"module.vpc": {Change: 2},
			},
			expectedCounts: map[string]int{
				"root":       2,
				"module.vpc": 2,
			},
		},
		{
			name: "replace actions counted correctly",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "module.x.aws_instance.y", Module: "module.x"},
					Action:   ActionDeleteThenCreate,
				},
				{
					Resource: Resource{Address: "module.x.aws_instance.z", Module: "module.x"},
					Action:   ActionCreateThenDelete,
				},
			},
			expectedGroups: map[string]ActionSummary{
				"module.x": {Replace: 2},
			},
			expectedCounts: map[string]int{
				"module.x": 2,
			},
		},
		{
			name: "indexed resources grouped with parent module",
			changes: []PlanChange{
				{
					Resource: Resource{Address: "module.ecs.aws_ecs_service.svc[0]", Module: "module.ecs"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "module.ecs.aws_ecs_service.svc[1]", Module: "module.ecs"},
					Action:   ActionUpdate,
				},
				{
					Resource: Resource{Address: "module.ecs.aws_ecs_task_definition.task", Module: "module.ecs"},
					Action:   ActionCreate,
				},
			},
			expectedGroups: map[string]ActionSummary{
				"module.ecs": {Add: 1, Change: 2},
			},
			expectedCounts: map[string]int{
				"module.ecs": 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GroupByModule(tt.changes)

			// Build a map for easy lookup
			resultMap := make(map[string]ModuleGroup)
			for _, g := range result {
				resultMap[g.Module] = g
			}

			if len(resultMap) != len(tt.expectedGroups) {
				t.Fatalf("got %d groups, want %d groups", len(resultMap), len(tt.expectedGroups))
			}

			for mod, wantSummary := range tt.expectedGroups {
				g, exists := resultMap[mod]
				if !exists {
					t.Errorf("expected group %q not found", mod)
					continue
				}
				if g.Summary != wantSummary {
					t.Errorf("group %q summary = %+v, want %+v", mod, g.Summary, wantSummary)
				}
				if wantCount, ok := tt.expectedCounts[mod]; ok {
					if len(g.Changes) != wantCount {
						t.Errorf("group %q changes count = %d, want %d", mod, len(g.Changes), wantCount)
					}
				}
			}
		})
	}
}

func TestGroupByModule_Sorting(t *testing.T) {
	changes := []PlanChange{
		{
			Resource: Resource{Address: "module.z.aws_instance.a", Module: "module.z"},
			Action:   ActionCreate,
		},
		{
			Resource: Resource{Address: "module.a.aws_instance.b", Module: "module.a"},
			Action:   ActionCreate,
		},
		{
			Resource: Resource{Address: "aws_instance.c", Module: ""},
			Action:   ActionCreate,
		},
	}

	result := GroupByModule(changes)

	if len(result) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(result))
	}

	expectedOrder := []string{"module.a", "module.z", "root"}
	for i, g := range result {
		if g.Module != expectedOrder[i] {
			t.Errorf("group[%d].Module = %q, want %q", i, g.Module, expectedOrder[i])
		}
	}
}
