package sdk

import "testing"

func TestGroupByModule_WhenEmpty_ShouldReturnEmptySlice(t *testing.T) {
	result := GroupByModule(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(result))
	}
}

func TestGroupByModule_WhenSingleModule_ShouldReturnOneGroup(t *testing.T) {
	changes := []PlanChange{
		{Resource: Resource{Address: "module.vpc.aws_vpc.main", Module: "module.vpc"}, Action: ActionCreate},
		{Resource: Resource{Address: "module.vpc.aws_subnet.a", Module: "module.vpc"}, Action: ActionUpdate},
	}

	result := GroupByModule(changes)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}
	if result[0].Module != "module.vpc" {
		t.Errorf("Module = %q, want %q", result[0].Module, "module.vpc")
	}
	if len(result[0].Changes) != 2 {
		t.Errorf("Changes length = %d, want 2", len(result[0].Changes))
	}
	if result[0].Summary.Add != 1 {
		t.Errorf("Summary.Add = %d, want 1", result[0].Summary.Add)
	}
	if result[0].Summary.Change != 1 {
		t.Errorf("Summary.Change = %d, want 1", result[0].Summary.Change)
	}
}

func TestGroupByModule_WhenRootResources_ShouldUseRootAsModule(t *testing.T) {
	changes := []PlanChange{
		{Resource: Resource{Address: "aws_instance.web", Module: ""}, Action: ActionDelete},
	}

	result := GroupByModule(changes)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}
	if result[0].Module != "root" {
		t.Errorf("Module = %q, want %q", result[0].Module, "root")
	}
	if result[0].Summary.Destroy != 1 {
		t.Errorf("Summary.Destroy = %d, want 1", result[0].Summary.Destroy)
	}
}

func TestGroupByModule_WhenMultipleModules_ShouldSortAlphabetically(t *testing.T) {
	changes := []PlanChange{
		{Resource: Resource{Address: "module.ecs.aws_ecs_service.app", Module: "module.ecs"}, Action: ActionCreate},
		{Resource: Resource{Address: "module.alb.aws_lb.main", Module: "module.alb"}, Action: ActionUpdate},
		{Resource: Resource{Address: "aws_instance.bastion", Module: ""}, Action: ActionDelete},
	}

	result := GroupByModule(changes)

	if len(result) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(result))
	}
	if result[0].Module != "module.alb" {
		t.Errorf("result[0].Module = %q, want %q", result[0].Module, "module.alb")
	}
	if result[1].Module != "module.ecs" {
		t.Errorf("result[1].Module = %q, want %q", result[1].Module, "module.ecs")
	}
	if result[2].Module != "root" {
		t.Errorf("result[2].Module = %q, want %q", result[2].Module, "root")
	}
}

func TestGroupByModule_WhenReplaceActions_ShouldCountReplace(t *testing.T) {
	changes := []PlanChange{
		{Resource: Resource{Address: "aws_instance.web", Module: ""}, Action: ActionDeleteThenCreate},
		{Resource: Resource{Address: "aws_instance.api", Module: ""}, Action: ActionCreateThenDelete},
	}

	result := GroupByModule(changes)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}
	if result[0].Summary.Replace != 2 {
		t.Errorf("Summary.Replace = %d, want 2", result[0].Summary.Replace)
	}
}

func TestGroupByModule_WhenMixedActions_ShouldCountEachType(t *testing.T) {
	changes := []PlanChange{
		{Resource: Resource{Address: "aws_instance.a", Module: "mod"}, Action: ActionCreate},
		{Resource: Resource{Address: "aws_instance.b", Module: "mod"}, Action: ActionCreate},
		{Resource: Resource{Address: "aws_instance.c", Module: "mod"}, Action: ActionUpdate},
		{Resource: Resource{Address: "aws_instance.d", Module: "mod"}, Action: ActionDelete},
		{Resource: Resource{Address: "aws_instance.e", Module: "mod"}, Action: ActionDeleteThenCreate},
		{Resource: Resource{Address: "aws_instance.f", Module: "mod"}, Action: ActionRead},
	}

	result := GroupByModule(changes)

	if len(result) != 1 {
		t.Fatalf("expected 1 group, got %d", len(result))
	}
	s := result[0].Summary
	if s.Add != 2 {
		t.Errorf("Summary.Add = %d, want 2", s.Add)
	}
	if s.Change != 1 {
		t.Errorf("Summary.Change = %d, want 1", s.Change)
	}
	if s.Destroy != 1 {
		t.Errorf("Summary.Destroy = %d, want 1", s.Destroy)
	}
	if s.Replace != 1 {
		t.Errorf("Summary.Replace = %d, want 1", s.Replace)
	}
}
