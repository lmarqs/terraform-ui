package ui

import "testing"

func TestFuzzyFilter_BasicMatch(t *testing.T) {
	f := NewFuzzyFilter(func(s string) string { return s })
	f.SetItems([]string{
		"aws_instance.web",
		"aws_s3_bucket.data",
		"aws_iam_role.admin",
		"google_compute_instance.app",
	})

	f.SetQuery("inst")
	results := f.Results()

	if len(results) != 2 {
		t.Fatalf("Results() len = %d, want 2", len(results))
	}
	// Both instance items should match
	found := map[string]bool{}
	for _, r := range results {
		found[r] = true
	}
	if !found["aws_instance.web"] || !found["google_compute_instance.app"] {
		t.Errorf("expected instance items, got %v", results)
	}
}

func TestFuzzyFilter_EmptyQuery(t *testing.T) {
	f := NewFuzzyFilter(func(s string) string { return s })
	items := []string{"a", "b", "c"}
	f.SetItems(items)

	f.SetQuery("")
	results := f.Results()

	if len(results) != 3 {
		t.Fatalf("empty query: Results() len = %d, want 3", len(results))
	}
}

func TestFuzzyFilter_NoMatch(t *testing.T) {
	f := NewFuzzyFilter(func(s string) string { return s })
	f.SetItems([]string{"aws_instance.web", "aws_s3_bucket.data"})

	f.SetQuery("zzzzz")
	results := f.Results()

	if len(results) != 0 {
		t.Errorf("no match: Results() len = %d, want 0", len(results))
	}
}

func TestFuzzyFilter_OriginalOrder(t *testing.T) {
	f := NewFuzzyFilter(func(s string) string { return s })
	items := []string{
		"module.vpc.aws_subnet.private",
		"module.vpc.aws_subnet.public",
		"aws_instance.bastion",
	}
	f.SetItems(items)

	f.SetQuery("subnet")
	ordered := f.OriginalOrder()

	if len(ordered) != 2 {
		t.Fatalf("OriginalOrder() len = %d, want 2", len(ordered))
	}
	// Should preserve original item order
	if ordered[0] != "module.vpc.aws_subnet.private" {
		t.Errorf("first = %q, want module.vpc.aws_subnet.private", ordered[0])
	}
	if ordered[1] != "module.vpc.aws_subnet.public" {
		t.Errorf("second = %q, want module.vpc.aws_subnet.public", ordered[1])
	}
}

func TestFuzzyFilter_IsActive(t *testing.T) {
	f := NewFuzzyFilter(func(s string) string { return s })
	f.SetItems([]string{"a", "b"})

	if f.IsActive() {
		t.Error("expected IsActive() = false before query")
	}

	f.SetQuery("a")
	if !f.IsActive() {
		t.Error("expected IsActive() = true after query")
	}

	f.Clear()
	if f.IsActive() {
		t.Error("expected IsActive() = false after Clear")
	}
}

func TestFuzzyFilter_Clear(t *testing.T) {
	f := NewFuzzyFilter(func(s string) string { return s })
	f.SetItems([]string{"a", "b", "c"})
	f.SetQuery("a")

	f.Clear()

	if f.Query() != "" {
		t.Errorf("after Clear: Query() = %q, want empty", f.Query())
	}
	if len(f.Results()) != 3 {
		t.Errorf("after Clear: Results() len = %d, want 3", len(f.Results()))
	}
}

func TestFuzzyFilter_CustomAccessor(t *testing.T) {
	type item struct {
		name  string
		value int
	}
	f := NewFuzzyFilter(func(i item) string { return i.name })
	f.SetItems([]item{
		{name: "alpha", value: 1},
		{name: "beta", value: 2},
		{name: "gamma", value: 3},
	})

	f.SetQuery("bet")
	results := f.Results()

	if len(results) != 1 {
		t.Fatalf("custom accessor: Results() len = %d, want 1", len(results))
	}
	if results[0].name != "beta" {
		t.Errorf("got %v, want beta", results[0])
	}
}
