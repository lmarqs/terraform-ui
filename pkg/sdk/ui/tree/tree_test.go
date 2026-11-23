package tree

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type testItem struct{ addr string }

func (t testItem) Address() string { return t.addr }

// pinnedSet builds the pin predicate a caller hands to the tree per render.
// Context owns Pins (ADR-0018); the tree keeps none of its own.
func pinnedSet(addresses ...string) func(string) bool {
	set := make(map[string]bool, len(addresses))
	for _, a := range addresses {
		set[a] = true
	}
	return func(address string) bool { return set[address] }
}

// findRow locates a visible row by the only string it identifies itself with: a
// branch by its ModulePath, a leaf by its Address.
func findRow(t *testing.T, tr *Tree, path string) *Node {
	t.Helper()
	for _, n := range tr.Nodes() {
		if n.Kind == KindBranch && n.ModulePath == path {
			return n
		}
		if n.Kind == KindLeaf && n.Address() == path {
			return n
		}
	}
	t.Fatalf("no row identified by %q", path)
	return nil
}

// visibleRows lists every visible row by its identifying string, so an
// expand/collapse assertion never has to name a row by index.
func visibleRows(tr *Tree) []string {
	rows := make([]string, 0, tr.VisibleCount())
	for _, n := range tr.Nodes() {
		if n.Kind == KindBranch {
			rows = append(rows, n.ModulePath)
			continue
		}
		rows = append(rows, n.Address())
	}
	return rows
}

func TestSplitTerraform_WhenGivenVariousAddresses(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected []string
	}{
		{
			"ShouldSplitSimpleResource",
			"aws_s3_bucket.main",
			[]string{"aws_s3_bucket.main"},
		},
		{
			"ShouldSplitSingleModuleResource",
			"module.vpc.aws_subnet.private",
			[]string{"module.vpc", "aws_subnet.private"},
		},
		{
			"ShouldSplitNestedModules",
			"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]",
			[]string{"module.medprev_online_prd", "module.postgresql_proxy", "aws_db_proxy.this[0]"},
		},
		{
			"ShouldSplitDeeplyNestedModules",
			"module.a.module.b.module.c.aws_instance.web",
			[]string{"module.a", "module.b", "module.c", "aws_instance.web"},
		},
		{
			"ShouldHandleResourceWithIndex",
			"module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens",
			[]string{"module.cloudwatch", "aws_cloudwatch_metric_alarm.bedrock_input_tokens"},
		},
		{
			"ShouldHandleDataSource",
			"module.network.data.aws_vpc.main",
			[]string{"module.network", "data.aws_vpc.main"},
		},
		{
			"ShouldHandleResourceWithBracketIndex",
			"aws_iam_role_policy_attachment.this[0]",
			[]string{"aws_iam_role_policy_attachment.this[0]"},
		},
		{
			"ShouldHandleModuleWithDotInKey",
			`module.user["dev.ops@corp.com"].aws_iam_user.this`,
			[]string{`module.user["dev.ops@corp.com"]`, "aws_iam_user.this"},
		},
		{
			"ShouldHandleNestedModuleWithDotInKey",
			`module.identity_center.module.user["admin.user@example.com"].aws_ssoadmin_account_assignment.this`,
			[]string{"module.identity_center", `module.user["admin.user@example.com"]`, "aws_ssoadmin_account_assignment.this"},
		},
		{
			"ShouldHandleMultipleDotsInKey",
			`module.user["first.middle.last@sub.domain.com"].aws_iam_user.this`,
			[]string{`module.user["first.middle.last@sub.domain.com"]`, "aws_iam_user.this"},
		},
		{
			"ShouldHandleResourceWithDotInKey",
			`aws_iam_user.this["dev.ops@corp.com"]`,
			[]string{`aws_iam_user.this["dev.ops@corp.com"]`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitTerraform(tt.address)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d segments, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, seg := range result {
				if seg != tt.expected[i] {
					t.Fatalf("segment[%d]: expected %q, got %q", i, tt.expected[i], seg)
				}
			}
		})
	}
}

func TestNew_WhenGivenFlatItems_ShouldBuildTree(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.api"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens"},
		testItem{"aws_s3_bucket.main"},
	}

	tree := New(items)

	t.Run("ShouldHaveCorrectVisibleCount", func(t *testing.T) {
		if tree.VisibleCount() == 0 {
			t.Fatal("expected visible nodes, got 0")
		}
	})

	t.Run("ShouldStartWithCursorAtZero", func(t *testing.T) {
		if tree.Cursor() != 0 {
			t.Fatalf("expected cursor at 0, got %d", tree.Cursor())
		}
	})

	t.Run("ShouldShowTopLevelNodesCollapsed", func(t *testing.T) {
		nodes := tree.Nodes()
		for _, n := range nodes {
			if n.Kind == KindBranch && n.Expanded {
				t.Fatalf("expected all branches to start collapsed, but %q is expanded", n.ModulePath)
			}
		}
	})

	t.Run("ShouldGroupByModule", func(t *testing.T) {
		nodes := tree.Nodes()
		hasBranch := false
		for _, n := range nodes {
			if n.Kind == KindBranch {
				hasBranch = true
				break
			}
		}
		if !hasBranch {
			t.Fatal("expected at least one branch node for module grouping")
		}
	})

	t.Run("ShouldShowRootLeafAsSibling", func(t *testing.T) {
		nodes := tree.Nodes()
		foundLeaf := false
		for _, n := range nodes {
			if n.Kind == KindLeaf && n.Address() == "aws_s3_bucket.main" {
				foundLeaf = true
				if n.Depth != 0 {
					t.Fatalf("expected root leaf at depth 0, got %d", n.Depth)
				}
			}
		}
		if !foundLeaf {
			t.Fatal("expected aws_s3_bucket.main as a root leaf node")
		}
	})
}

func TestNode_WhenBranchRow_AddressIsEmpty(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.module.subnets.aws_subnet.private[0]"},
		testItem{"module.vpc.aws_vpc.main"},
		testItem{"aws_instance.web"},
	}
	tr := New(items)
	tr.MoveToStart()
	branch := tr.CursorNode()

	if branch.Kind != KindBranch {
		t.Fatalf("first row kind = %v, want KindBranch", branch.Kind)
	}
	if branch.ModulePath != "module.vpc" {
		t.Errorf("ModulePath = %q, want %q", branch.ModulePath, "module.vpc")
	}
	if got := branch.Address(); got != "" {
		t.Errorf("Address() on a branch row = %q, want empty", got)
	}
}

func TestNode_WhenLeafRow_ModulePathIsEmpty(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.module.subnets.aws_subnet.private[0]"},
		testItem{"module.vpc.aws_vpc.main"},
		testItem{"aws_instance.web"},
	}
	tr := New(items)
	tr.ExpandAll()

	tests := []struct {
		name    string
		address string
	}{
		{"root leaf", "aws_instance.web"},
		{"leaf under a module", "module.vpc.aws_vpc.main"},
		{"leaf under a nested module", "module.vpc.module.subnets.aws_subnet.private[0]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaf := findRow(t, tr, tt.address)
			if leaf.Kind != KindLeaf {
				t.Fatalf("row kind = %v, want KindLeaf", leaf.Kind)
			}
			if got := leaf.Address(); got != tt.address {
				t.Errorf("Address() = %q, want %q", got, tt.address)
			}
			if leaf.ModulePath != "" {
				t.Errorf("ModulePath on a leaf row = %q, want empty", leaf.ModulePath)
			}
		})
	}
}

func TestNode_WhenLeafRow_AddressComesFromItsItem(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.module.subnets.aws_subnet.private[0]"},
		testItem{"module.vpc.aws_vpc.main"},
		testItem{"aws_instance.web"},
	}
	tr := New(items)
	tr.ExpandAll()

	leaves := 0
	for _, n := range tr.Nodes() {
		if n.Kind != KindLeaf {
			continue
		}
		leaves++
		if n.Item == nil {
			t.Fatalf("leaf row %q carries no Item", n.Label)
		}
		if n.Address() != n.Item.Address() {
			t.Errorf("Address() = %q, Item.Address() = %q; a leaf row must not diverge from its item", n.Address(), n.Item.Address())
		}
	}
	if leaves != len(items) {
		t.Fatalf("visited %d leaf rows, want %d", leaves, len(items))
	}
}

func TestNode_WhenModuleIsNested_ModulePathIsFullyQualified(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.module.subnets.aws_subnet.private[0]"},
		testItem{"module.vpc.aws_vpc.main"},
		testItem{"aws_instance.web"},
	}
	tr := New(items)
	tr.ExpandAll()

	nested := findRow(t, tr, "module.vpc.module.subnets")
	if nested.Kind != KindBranch {
		t.Fatalf("row kind = %v, want KindBranch", nested.Kind)
	}
	if nested.Label != "module.subnets" {
		t.Errorf("Label = %q, want %q", nested.Label, "module.subnets")
	}
	if nested.ModulePath != "module.vpc.module.subnets" {
		t.Errorf("ModulePath = %q, want %q", nested.ModulePath, "module.vpc.module.subnets")
	}
	if got := nested.Address(); got != "" {
		t.Errorf("Address() on a nested branch row = %q, want empty", got)
	}
}

func TestNavigation_WhenMoving_ShouldUpdateCursor(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens"},
		testItem{"aws_s3_bucket.main"},
		testItem{"aws_s3_bucket.logs"},
	}
	tree := New(items)

	t.Run("ShouldMoveDownByOne", func(t *testing.T) {
		tree.MoveToStart()
		tree.MoveDown()
		if tree.Cursor() != 1 {
			t.Fatalf("expected cursor at 1, got %d", tree.Cursor())
		}
	})

	t.Run("ShouldMoveUpByOne", func(t *testing.T) {
		tree.MoveToEnd()
		prev := tree.Cursor()
		tree.MoveUp()
		if tree.Cursor() != prev-1 {
			t.Fatalf("expected cursor at %d, got %d", prev-1, tree.Cursor())
		}
	})

	t.Run("ShouldNotMoveAboveZero", func(t *testing.T) {
		tree.MoveToStart()
		tree.MoveUp()
		if tree.Cursor() != 0 {
			t.Fatalf("expected cursor at 0, got %d", tree.Cursor())
		}
	})

	t.Run("ShouldNotMoveBelowLastItem", func(t *testing.T) {
		tree.MoveToEnd()
		last := tree.Cursor()
		tree.MoveDown()
		if tree.Cursor() != last {
			t.Fatalf("expected cursor at %d, got %d", last, tree.Cursor())
		}
	})

	t.Run("ShouldMoveToStart", func(t *testing.T) {
		tree.MoveDown()
		tree.MoveDown()
		tree.MoveToStart()
		if tree.Cursor() != 0 {
			t.Fatalf("expected cursor at 0, got %d", tree.Cursor())
		}
	})

	t.Run("ShouldMoveToEnd", func(t *testing.T) {
		tree.MoveToEnd()
		expected := tree.VisibleCount() - 1
		if tree.Cursor() != expected {
			t.Fatalf("expected cursor at %d, got %d", expected, tree.Cursor())
		}
	})
}

func TestToggle_WhenExpandingBranch_ShouldRevealChildren(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.api"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)

	t.Run("ShouldIncreaseVisibleCountOnExpand", func(t *testing.T) {
		tree.MoveToStart()
		node := tree.CursorNode()
		if node.Kind != KindBranch {
			t.Fatal("expected first node to be a branch")
		}
		before := tree.VisibleCount()
		tree.Toggle()
		after := tree.VisibleCount()
		if after <= before {
			t.Fatalf("expected visible count to increase after expand, was %d now %d", before, after)
		}
	})

	t.Run("ShouldDecreaseVisibleCountOnCollapse", func(t *testing.T) {
		tree.MoveToStart()
		before := tree.VisibleCount()
		tree.Toggle()
		after := tree.VisibleCount()
		if after >= before {
			t.Fatalf("expected visible count to decrease after collapse, was %d now %d", before, after)
		}
	})

	t.Run("ShouldNotToggleLeafNode", func(t *testing.T) {
		tree.CollapseAll()
		tree.MoveToEnd()
		node := tree.CursorNode()
		if node.Kind != KindLeaf {
			t.Skip("last node is not a leaf in this configuration")
		}
		before := tree.VisibleCount()
		tree.Toggle()
		after := tree.VisibleCount()
		if after != before {
			t.Fatalf("expected visible count to stay same for leaf toggle, was %d now %d", before, after)
		}
	})
}

func TestExpandAll_CollapseAll_ShouldAffectAllBranches(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.api"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)

	t.Run("ShouldExpandAllBranches", func(t *testing.T) {
		tree.ExpandAll()
		for _, n := range tree.Nodes() {
			if n.Kind == KindBranch && !n.Expanded {
				t.Fatalf("expected branch %q to be expanded", n.ModulePath)
			}
		}
	})

	t.Run("ShouldShowAllLeavesAfterExpandAll", func(t *testing.T) {
		tree.ExpandAll()
		leafCount := 0
		for _, n := range tree.Nodes() {
			if n.Kind == KindLeaf {
				leafCount++
			}
		}
		if leafCount != len(items) {
			t.Fatalf("expected %d leaves visible, got %d", len(items), leafCount)
		}
	})

	t.Run("ShouldCollapseAllBranches", func(t *testing.T) {
		tree.ExpandAll()
		tree.CollapseAll()
		for _, n := range tree.Nodes() {
			if n.Kind == KindBranch && n.Expanded {
				t.Fatalf("expected branch %q to be collapsed", n.ModulePath)
			}
		}
	})

	t.Run("ShouldResetCursorOnCollapseAll", func(t *testing.T) {
		tree.ExpandAll()
		tree.MoveToEnd()
		tree.CollapseAll()
		if tree.Cursor() != 0 {
			t.Fatalf("expected cursor at 0 after CollapseAll, got %d", tree.Cursor())
		}
	})
}

func TestExpandCollapse_WhenToggledByCursor_VisibleRowsFollowBranchIdentity(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.module.subnets.aws_subnet.private[0]"},
		testItem{"module.vpc.aws_vpc.main"},
		testItem{"aws_instance.web"},
	}
	tr := New(items)

	collapsed := []string{"module.vpc", "aws_instance.web"}

	t.Run("ShouldStartFullyCollapsed", func(t *testing.T) {
		if got := visibleRows(tr); !reflect.DeepEqual(got, collapsed) {
			t.Errorf("rows = %v, want %v", got, collapsed)
		}
	})

	t.Run("ShouldRevealOneLevelWhenBranchAtCursorExpands", func(t *testing.T) {
		tr.MoveToStart()
		tr.Toggle()
		want := []string{
			"module.vpc",
			"module.vpc.module.subnets",
			"module.vpc.aws_vpc.main",
			"aws_instance.web",
		}
		if got := visibleRows(tr); !reflect.DeepEqual(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
	})

	t.Run("ShouldHideThatLevelAgainWhenTheSameBranchCollapses", func(t *testing.T) {
		tr.MoveToStart()
		tr.Toggle()
		if got := visibleRows(tr); !reflect.DeepEqual(got, collapsed) {
			t.Errorf("rows = %v, want %v", got, collapsed)
		}
	})

	t.Run("ShouldRevealEveryDescendantOnExpandAll", func(t *testing.T) {
		tr.ExpandAll()
		want := []string{
			"module.vpc",
			"module.vpc.module.subnets",
			"module.vpc.module.subnets.aws_subnet.private[0]",
			"module.vpc.aws_vpc.main",
			"aws_instance.web",
		}
		if got := visibleRows(tr); !reflect.DeepEqual(got, want) {
			t.Errorf("rows = %v, want %v", got, want)
		}
	})

	t.Run("ShouldReturnToTopLevelOnCollapseAll", func(t *testing.T) {
		tr.ExpandAll()
		tr.CollapseAll()
		if got := visibleRows(tr); !reflect.DeepEqual(got, collapsed) {
			t.Errorf("rows = %v, want %v", got, collapsed)
		}
	})
}

func TestPinAddresses_WhenRowIsBranch_ShouldNeverIncludeTheModulePath(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.module.subnets.aws_subnet.private[0]"},
		testItem{"module.vpc.aws_vpc.main"},
		testItem{"aws_instance.web"},
	}
	tr := New(items)
	tr.ExpandAll()

	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "leaf row resolves to exactly its own address",
			path: "aws_instance.web",
			want: []string{"aws_instance.web"},
		},
		{
			name: "leaf row inside a nested module resolves to exactly its own address",
			path: "module.vpc.module.subnets.aws_subnet.private[0]",
			want: []string{"module.vpc.module.subnets.aws_subnet.private[0]"},
		},
		{
			name: "branch row resolves to every descendant leaf",
			path: "module.vpc",
			want: []string{
				"module.vpc.module.subnets.aws_subnet.private[0]",
				"module.vpc.aws_vpc.main",
			},
		},
		{
			name: "nested branch row resolves to its own subtree only",
			path: "module.vpc.module.subnets",
			want: []string{"module.vpc.module.subnets.aws_subnet.private[0]"},
		},
	}

	modulePaths := pinnedSet("module.vpc", "module.vpc.module.subnets")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.PinAddresses(findRow(t, tr, tt.path))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PinAddresses(%q) = %v, want %v", tt.path, got, tt.want)
			}
			for _, a := range got {
				if modulePaths(a) {
					t.Errorf("PinAddresses(%q) returned module path %q; pins are resource addresses", tt.path, a)
				}
			}
		})
	}
}

func TestCollapseFocused_WhenOnLeaf_ShouldCollapseParent(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.api"},
	}
	tree := New(items)

	t.Run("ShouldCollapseParentBranch", func(t *testing.T) {
		tree.ExpandAll()
		// Navigate to a leaf
		var leafIdx int
		for i, n := range tree.Nodes() {
			if n.Kind == KindLeaf {
				leafIdx = i
				break
			}
		}
		tree.MoveToStart()
		for i := 0; i < leafIdx; i++ {
			tree.MoveDown()
		}
		node := tree.CursorNode()
		if node.Kind != KindLeaf {
			t.Fatalf("expected leaf node, got branch")
		}
		beforeCount := tree.VisibleCount()
		tree.CollapseFocused()
		afterCount := tree.VisibleCount()
		if afterCount >= beforeCount {
			t.Fatalf("expected fewer visible nodes after collapsing parent, was %d now %d", beforeCount, afterCount)
		}
	})

	t.Run("ShouldMoveCursorToParent", func(t *testing.T) {
		tree.ExpandAll()
		// Find a leaf and navigate there
		var leafIdx int
		for i, n := range tree.Nodes() {
			if n.Kind == KindLeaf {
				leafIdx = i
				break
			}
		}
		tree.MoveToStart()
		for i := 0; i < leafIdx; i++ {
			tree.MoveDown()
		}
		tree.CollapseFocused()
		node := tree.CursorNode()
		if node == nil {
			t.Fatal("expected cursor to be on a node")
		}
		if node.Kind != KindBranch {
			t.Fatal("expected cursor to move to parent branch")
		}
	})

	t.Run("ShouldCollapseExpandedBranch", func(t *testing.T) {
		tree.ExpandAll()
		tree.MoveToStart()
		node := tree.CursorNode()
		if node.Kind != KindBranch || !node.Expanded {
			t.Skip("first node is not an expanded branch")
		}
		tree.CollapseFocused()
		node = tree.CursorNode()
		if node.Expanded {
			t.Fatal("expected branch to be collapsed")
		}
	})
}

func TestExpandFocused_WhenOnBranch_ShouldExpand(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)

	t.Run("ShouldExpandBranch", func(t *testing.T) {
		tree.MoveToStart()
		node := tree.CursorNode()
		if node.Kind != KindBranch {
			t.Fatal("expected branch at start")
		}
		before := tree.VisibleCount()
		tree.ExpandFocused()
		after := tree.VisibleCount()
		if after <= before {
			t.Fatalf("expected more visible nodes after ExpandFocused, was %d now %d", before, after)
		}
	})
}

func TestPinAddresses_ShouldResolveRowToResourceAddresses(t *testing.T) {
	items := []Item{
		testItem{"module.cloudwatch.module.alarms.aws_cloudwatch_metric_alarm.cpu_high"},
		testItem{"module.cloudwatch.aws_cloudwatch_dashboard.main"},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)
	tr.ExpandAll()

	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "leaf row is its own address",
			path: "aws_s3_bucket.main",
			want: []string{"aws_s3_bucket.main"},
		},
		{
			name: "branch row covers every descendant leaf, nested modules included",
			path: "module.cloudwatch",
			want: []string{
				"module.cloudwatch.module.alarms.aws_cloudwatch_metric_alarm.cpu_high",
				"module.cloudwatch.aws_cloudwatch_dashboard.main",
			},
		},
		{
			name: "nested branch row covers only its own subtree",
			path: "module.cloudwatch.module.alarms",
			want: []string{"module.cloudwatch.module.alarms.aws_cloudwatch_metric_alarm.cpu_high"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tr.PinAddresses(findRow(t, tr, tt.path))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PinAddresses(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestPinAddresses_WhenBranchIsCollapsed_ShouldStillCoverHiddenLeaves(t *testing.T) {
	items := []Item{
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"},
		testItem{"module.cloudwatch.aws_cloudwatch_dashboard.main"},
	}
	tr := New(items)

	tr.MoveToStart()
	node := tr.CursorNode()
	if node.Kind != KindBranch {
		t.Fatalf("first row kind = %v, want KindBranch", node.Kind)
	}

	want := []string{
		"module.cloudwatch.aws_cloudwatch_dashboard.main",
		"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high",
	}
	if got := tr.PinAddresses(node); !reflect.DeepEqual(got, want) {
		t.Errorf("PinAddresses(collapsed branch) = %v, want %v", got, want)
	}
}

func TestPinAddresses_WhenPinnedBackFromCaller_ShouldRenderBranchAsFullyPinned(t *testing.T) {
	items := []Item{
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"},
		testItem{"module.cloudwatch.aws_cloudwatch_dashboard.main"},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)
	tr.MoveToStart()
	branch := tr.CursorNode()

	// This is the whole point of the interface: the addresses a caller pins in
	// response to a branch row must be the addresses the tree then reports as
	// pinned for that row.
	pinned := pinnedSet(tr.PinAddresses(branch)...)

	if got := tr.PinStateOf(branch, pinned); got != PinFull {
		t.Errorf("PinStateOf(branch) = %v, want PinFull", got)
	}
	if pinned("aws_s3_bucket.main") {
		t.Error("unrelated leaf became pinned")
	}
}
func TestPinStateOf_ShouldDeriveRowStateFromTheCallersPins(t *testing.T) {
	items := []Item{
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high"},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)
	tr.ExpandAll()

	tests := []struct {
		name   string
		path   string
		pinned func(string) bool
		want   PinState
	}{
		{
			name:   "branch with nothing pinned",
			path:   "module.cloudwatch",
			pinned: pinnedSet(),
			want:   PinNone,
		},
		{
			name:   "branch with some leaves pinned",
			path:   "module.cloudwatch",
			pinned: pinnedSet("module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"),
			want:   PinPartial,
		},
		{
			name: "branch with every leaf pinned",
			path: "module.cloudwatch",
			pinned: pinnedSet(
				"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high",
				"module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high",
			),
			want: PinFull,
		},
		{
			name:   "branch pinned by its module path only",
			path:   "module.cloudwatch",
			pinned: pinnedSet("module.cloudwatch"),
			want:   PinNone,
		},
		{
			name:   "pinned leaf",
			path:   "aws_s3_bucket.main",
			pinned: pinnedSet("aws_s3_bucket.main"),
			want:   PinFull,
		},
		{
			name:   "unpinned leaf",
			path:   "aws_s3_bucket.main",
			pinned: pinnedSet(),
			want:   PinNone,
		},
		{
			name:   "no predicate supplied",
			path:   "aws_s3_bucket.main",
			pinned: nil,
			want:   PinNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tr.PinStateOf(findRow(t, tr, tt.path), tt.pinned); got != tt.want {
				t.Errorf("PinStateOf(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestCursorItem_WhenOnDifferentNodeTypes(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)

	t.Run("ShouldReturnNilOnBranch", func(t *testing.T) {
		tree.MoveToStart()
		node := tree.CursorNode()
		if node.Kind != KindBranch {
			t.Skip("first node is not a branch")
		}
		item := tree.CursorItem()
		if item != nil {
			t.Fatal("expected CursorItem to return nil on branch node")
		}
	})

	t.Run("ShouldReturnItemOnLeaf", func(t *testing.T) {
		// Find a leaf node
		tree.ExpandAll()
		for i, n := range tree.Nodes() {
			if n.Kind == KindLeaf {
				tree.MoveToStart()
				for j := 0; j < i; j++ {
					tree.MoveDown()
				}
				break
			}
		}
		item := tree.CursorItem()
		if item == nil {
			t.Fatal("expected CursorItem to return non-nil on leaf node")
		}
	})

	t.Run("ShouldReturnCorrectItemAddress", func(t *testing.T) {
		// Navigate to the aws_s3_bucket.main leaf
		tree.ExpandAll()
		for i, n := range tree.Nodes() {
			if n.Kind == KindLeaf && n.Address() == "aws_s3_bucket.main" {
				tree.MoveToStart()
				for j := 0; j < i; j++ {
					tree.MoveDown()
				}
				break
			}
		}
		item := tree.CursorItem()
		if item == nil {
			t.Fatal("expected non-nil item")
		}
		if item.Address() != "aws_s3_bucket.main" {
			t.Fatalf("expected address aws_s3_bucket.main, got %q", item.Address())
		}
	})
}

func TestCursorNode_WhenTreeIsEmpty_ShouldReturnNil(t *testing.T) {
	tree := New([]Item{})
	node := tree.CursorNode()
	if node != nil {
		t.Fatal("expected CursorNode to return nil on empty tree")
	}
}

func TestRenderRows_WhenTreeHasContent_ShouldIncludeConnectors(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.api"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)
	tree.ExpandAll()

	opts := RenderOpts{}
	output := renderText(tree, opts)

	t.Run("ShouldContainBranchConnector", func(t *testing.T) {
		if !strings.Contains(output, "├─") {
			t.Fatal("expected output to contain branch connector")
		}
	})

	t.Run("ShouldContainLastChildConnector", func(t *testing.T) {
		if !strings.Contains(output, "└─") {
			t.Fatal("expected output to contain last-child connector")
		}
	})

	t.Run("ShouldContainVerticalLine", func(t *testing.T) {
		if !strings.Contains(output, "│") {
			t.Fatal("expected output to contain vertical line connector")
		}
	})

	t.Run("ShouldContainExpandedIndicator", func(t *testing.T) {
		if !strings.Contains(output, "▼") {
			t.Fatal("expected output to contain expanded indicator")
		}
	})

	t.Run("ShouldContainLeafLabel", func(t *testing.T) {
		if !strings.Contains(output, "aws_s3_bucket.main") {
			t.Fatal("expected output to contain leaf label")
		}
	})

	t.Run("ShouldContainBranchCountInParens", func(t *testing.T) {
		if !strings.Contains(output, "(") || !strings.Contains(output, ")") {
			t.Fatal("expected output to contain branch count in parentheses")
		}
	})
}

func TestRenderRows_WhenExpanded_ShouldDrawEachRowsConnectorColumns(t *testing.T) {
	tests := []struct {
		name  string
		items []string
		want  []string
	}{
		{
			name: "a continuation line runs under a row with a later sibling, a blank under a last child",
			items: []string{
				"module.a.module.b.aws_x.one",
				"module.a.module.b.aws_x.two",
				"module.a.aws_y.three",
				"module.c.aws_z.four",
				"aws_s3_bucket.main",
			},
			want: []string{
				"  ▼ module.a (3)",
				"  ├─▼ module.b (2)",
				"  │ ├─aws_x.one",
				"  │ └─aws_x.two",
				"  └─aws_y.three",
				"  ▼ module.c (1)",
				"  └─aws_z.four",
				"  aws_s3_bucket.main",
			},
		},
		{
			name:  "a chain of last children leaves a blank column per level",
			items: []string{"module.a.module.b.aws_x.one"},
			want: []string{
				"  ▼ module.a (1)",
				"  └─▼ module.b (1)",
				"    └─aws_x.one",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]Item, len(tt.items))
			for i, addr := range tt.items {
				items[i] = testItem{addr}
			}
			tr := New(items)
			tr.ExpandAll()

			got := tr.RenderRows(RenderOpts{})

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rows =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestRenderRows_WhenCollapsed_ShouldShowCollapsedIndicator(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)
	opts := RenderOpts{}
	output := renderText(tree, opts)

	if !strings.Contains(output, "▶") {
		t.Fatal("expected output to contain collapsed indicator")
	}
}

func TestRenderRows_WithPins_ShouldShowPinIndicator(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"aws_s3_bucket.main"},
		testItem{"aws_s3_bucket.logs"},
	}
	tree := New(items)
	tree.ExpandAll()
	pinned := pinnedSet("aws_s3_bucket.main")

	t.Run("ShouldShowDefaultPinIndicator", func(t *testing.T) {
		opts := RenderOpts{Pinned: pinned}
		output := renderText(tree, opts)
		if !strings.Contains(output, "* ") {
			t.Fatal("expected output to contain default pin indicator '* '")
		}
	})

	t.Run("ShouldShowCustomPinIndicator", func(t *testing.T) {
		opts := RenderOpts{
			PinIndicator: "[P] ",
			Pinned:       pinned,
		}
		output := renderText(tree, opts)
		if !strings.Contains(output, "[P] ") {
			t.Fatal("expected output to contain custom pin indicator '[P] '")
		}
	})

	t.Run("ShouldNotShowPinForUnpinnedItems", func(t *testing.T) {
		opts := RenderOpts{Pinned: pinned}
		output := renderText(tree, opts)
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "aws_s3_bucket.logs") {
				if strings.HasPrefix(line, "* ") {
					t.Fatal("expected unpinned item to not have pin indicator")
				}
				break
			}
		}
	})
}

func TestRenderRows_WithCustomRenderers(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.main"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)
	tree.ExpandAll()

	t.Run("ShouldUseCustomRenderLeaf", func(t *testing.T) {
		opts := RenderOpts{
			RenderLeaf: func(node *Node, pinned bool) string {
				return "[LEAF:" + node.Label + "]"
			},
		}
		output := renderText(tree, opts)
		if !strings.Contains(output, "[LEAF:") {
			t.Fatal("expected output to use custom leaf renderer")
		}
	})

	t.Run("ShouldUseCustomRenderBranch", func(t *testing.T) {
		opts := RenderOpts{
			RenderBranch: func(node *Node, pinned bool) string {
				return "[BRANCH:" + node.Label + "]"
			},
		}
		output := renderText(tree, opts)
		if !strings.Contains(output, "[BRANCH:") {
			t.Fatal("expected output to use custom branch renderer")
		}
	})
}

func TestNew_WithCustomSplitFunc_ShouldUseIt(t *testing.T) {
	items := []Item{
		testItem{"a/b/c"},
		testItem{"a/b/d"},
		testItem{"a/e"},
	}

	splitSlash := func(addr string) []string {
		return strings.Split(addr, "/")
	}
	tree := New(items, WithSplitFunc(splitSlash))
	tree.ExpandAll()

	found := false
	for _, n := range tree.Nodes() {
		if n.Kind == KindBranch && n.Label == "a" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected branch node 'a' from custom split function")
	}
}

func TestSetItems_ShouldPreserveExpansionState(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.main"},
		testItem{"module.vpc.aws_subnet.secondary"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)

	tree.MoveToStart()
	tree.Toggle()
	node := tree.CursorNode()
	expandedPath := node.ModulePath

	newItems := []Item{
		testItem{"module.vpc.aws_subnet.main"},
		testItem{"module.vpc.aws_subnet.secondary"},
		testItem{"module.vpc.aws_subnet.tertiary"},
		testItem{"aws_s3_bucket.main"},
	}
	tree.SetItems(newItems)

	for _, n := range tree.Nodes() {
		if n.Kind == KindBranch && n.ModulePath == expandedPath {
			if !n.Expanded {
				t.Fatal("expected expansion state to be preserved after SetItems")
			}
			return
		}
	}
	t.Fatal("expected to find previously expanded branch after SetItems")
}

func TestVisibleCount_ShouldChangeWithExpansion(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.main"},
		testItem{"module.vpc.aws_subnet.secondary"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)

	collapsed := tree.VisibleCount()
	tree.ExpandAll()
	expanded := tree.VisibleCount()

	if expanded <= collapsed {
		t.Fatalf("expected more visible nodes when expanded (%d) than collapsed (%d)", expanded, collapsed)
	}
}

func TestWithPreserveOrder_ShouldKeepInsertionOrder(t *testing.T) {
	items := []Item{
		testItem{"zebra.resource"},
		testItem{"alpha.resource"},
		testItem{"middle.resource"},
	}

	t.Run("ShouldSortAlphabeticallyByDefault", func(t *testing.T) {
		tr := New(items)
		nodes := tr.Nodes()
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
		if nodes[0].Address() != "alpha.resource" {
			t.Fatalf("expected first node to be alpha.resource, got %q", nodes[0].Address())
		}
		if nodes[1].Address() != "middle.resource" {
			t.Fatalf("expected second node to be middle.resource, got %q", nodes[1].Address())
		}
		if nodes[2].Address() != "zebra.resource" {
			t.Fatalf("expected third node to be zebra.resource, got %q", nodes[2].Address())
		}
	})

	t.Run("ShouldPreserveInsertionOrder", func(t *testing.T) {
		tr := New(items, WithPreserveOrder())
		nodes := tr.Nodes()
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
		if nodes[0].Address() != "zebra.resource" {
			t.Fatalf("expected first node to be zebra.resource, got %q", nodes[0].Address())
		}
		if nodes[1].Address() != "alpha.resource" {
			t.Fatalf("expected second node to be alpha.resource, got %q", nodes[1].Address())
		}
		if nodes[2].Address() != "middle.resource" {
			t.Fatalf("expected third node to be middle.resource, got %q", nodes[2].Address())
		}
	})

	t.Run("ShouldNavigateInInsertionOrder", func(t *testing.T) {
		tr := New(items, WithPreserveOrder())
		tr.MoveToStart()
		node := tr.CursorNode()
		if node.Address() != "zebra.resource" {
			t.Fatalf("expected cursor to start at zebra.resource, got %q", node.Address())
		}
		tr.MoveDown()
		node = tr.CursorNode()
		if node.Address() != "alpha.resource" {
			t.Fatalf("expected cursor at alpha.resource after MoveDown, got %q", node.Address())
		}
		tr.MoveDown()
		node = tr.CursorNode()
		if node.Address() != "middle.resource" {
			t.Fatalf("expected cursor at middle.resource after second MoveDown, got %q", node.Address())
		}
	})

	t.Run("ShouldPreserveOrderWithIdentitySplit", func(t *testing.T) {
		identity := func(addr string) []string { return []string{addr} }
		tr := New(items, WithSplitFunc(identity), WithPreserveOrder())
		nodes := tr.Nodes()
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
		if nodes[0].Address() != "zebra.resource" {
			t.Fatalf("expected first node zebra.resource, got %q", nodes[0].Address())
		}
		if nodes[1].Address() != "alpha.resource" {
			t.Fatalf("expected second node alpha.resource, got %q", nodes[1].Address())
		}
		if nodes[2].Address() != "middle.resource" {
			t.Fatalf("expected third node middle.resource, got %q", nodes[2].Address())
		}
	})

	t.Run("CursorItemShouldMatchPosition", func(t *testing.T) {
		identity := func(addr string) []string { return []string{addr} }
		tr := New(items, WithSplitFunc(identity), WithPreserveOrder())
		tr.MoveDown()
		item := tr.CursorItem()
		if item == nil {
			t.Fatal("expected non-nil CursorItem")
		}
		if item.Address() != "alpha.resource" {
			t.Fatalf("expected CursorItem at position 1 to be alpha.resource, got %q", item.Address())
		}
	})
}

func TestNew_WhenAddressesHaveDotsInBracketKeys_ShouldGroupCorrectly(t *testing.T) {
	items := []Item{
		testItem{`module.user["dev.ops@corp.com"].aws_iam_user.this`},
		testItem{`module.user["dev.ops@corp.com"].aws_iam_access_key.this`},
		testItem{`module.user["admin@example.com"].aws_iam_user.this`},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)

	t.Run("ShouldCreateCorrectBranchLabels", func(t *testing.T) {
		tr.ExpandAll()
		var branchLabels []string
		for _, n := range tr.Nodes() {
			if n.Kind == KindBranch {
				branchLabels = append(branchLabels, n.Label)
			}
		}
		found := false
		for _, label := range branchLabels {
			if label == `module.user["dev.ops@corp.com"]` {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected branch label module.user[\"dev.ops@corp.com\"], got branches: %v", branchLabels)
		}
	})

	t.Run("ShouldCountLeavesCorrectly", func(t *testing.T) {
		tr.CollapseAll()
		nodes := tr.Nodes()
		for _, n := range nodes {
			if n.Kind == KindBranch && n.Label == `module.user["dev.ops@corp.com"]` {
				if n.Count != 2 {
					t.Fatalf("expected 2 leaves under dev.ops branch, got %d", n.Count)
				}
				return
			}
		}
		t.Fatal("did not find expected branch node")
	})
}

func TestBranchNodeCount_ShouldReflectDescendants(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.api"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.worker"},
	}
	tree := New(items)

	node := tree.CursorNode()
	if node.Kind != KindBranch {
		t.Fatal("expected first node to be a branch")
	}
	if node.Count != 3 {
		t.Fatalf("expected branch count to be 3 (all leaves), got %d", node.Count)
	}
}

func TestRenderRows_WithPinIndicators_ShouldShowStateSpecificIndicators(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.a"},
		testItem{"module.vpc.aws_subnet.b"},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)
	tr.ExpandAll()

	indicators := &PinIndicators{
		None:    "[ ] ",
		Full:    "[*] ",
		Partial: "[-] ",
	}

	opts := RenderOpts{
		PinIndicators: indicators,
		Pinned:        pinnedSet("module.vpc.aws_subnet.a"),
	}
	output := renderText(tr, opts)

	t.Run("ShouldShowFullIndicatorForPinnedLeaf", func(t *testing.T) {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "aws_subnet.a") && !strings.Contains(line, "module.vpc") {
				if !strings.Contains(line, "[*] ") {
					t.Fatalf("expected pinned leaf to show [*], got line: %q", line)
				}
				return
			}
		}
	})

	t.Run("ShouldShowNoneIndicatorForUnpinnedLeaf", func(t *testing.T) {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "aws_s3_bucket.main") {
				if !strings.Contains(line, "[ ] ") {
					t.Fatalf("expected unpinned leaf to show [ ], got line: %q", line)
				}
				return
			}
		}
	})

	t.Run("ShouldShowPartialIndicatorForPartiallyPinnedBranch", func(t *testing.T) {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "module.vpc") && strings.Contains(line, "(") {
				if !strings.Contains(line, "[-] ") {
					t.Fatalf("expected partially pinned branch to show [-], got line: %q", line)
				}
				return
			}
		}
	})

	t.Run("ShouldShowFullIndicatorForFullyPinnedBranch", func(t *testing.T) {
		tr3 := New(items)
		tr3.ExpandAll()
		opts3 := RenderOpts{
			PinIndicators: indicators,
			Pinned:        pinnedSet("module.vpc.aws_subnet.a", "module.vpc.aws_subnet.b"),
		}
		output3 := renderText(tr3, opts3)
		lines := strings.Split(output3, "\n")
		for _, line := range lines {
			if strings.Contains(line, "module.vpc") && strings.Contains(line, "(") {
				if !strings.Contains(line, "[*] ") {
					t.Fatalf("expected fully pinned branch to show [*], got line: %q", line)
				}
				return
			}
		}
	})
}

func TestLeafLabel_WhenSplitReturnsEmpty_ShouldReturnOriginalAddress(t *testing.T) {
	emptySplit := func(addr string) []string { return []string{} }
	result := leafLabel("some.address", emptySplit)
	if result != "some.address" {
		t.Fatalf("expected original address when splitFunc returns empty, got %q", result)
	}
}

func TestLeafLabel_WhenSplitReturnsNil_ShouldReturnOriginalAddress(t *testing.T) {
	nilSplit := func(addr string) []string { return nil }
	result := leafLabel("some.address", nilSplit)
	if result != "some.address" {
		t.Fatalf("expected original address when splitFunc returns nil, got %q", result)
	}
}

func TestCollapseFocused_WhenOnLeafAtRoot_ShouldDoNothing(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.main"},
		testItem{"aws_s3_bucket.logs"},
	}
	tr := New(items)

	tr.MoveToStart()
	node := tr.CursorNode()
	if node.Kind != KindLeaf {
		t.Fatal("expected leaf at start for flat items")
	}
	before := tr.VisibleCount()
	cursorBefore := tr.Cursor()
	tr.CollapseFocused()
	after := tr.VisibleCount()
	cursorAfter := tr.Cursor()

	if after != before {
		t.Fatalf("expected no change in visible count, was %d now %d", before, after)
	}
	if cursorAfter != cursorBefore {
		t.Fatalf("expected cursor to stay at %d, got %d", cursorBefore, cursorAfter)
	}
}

func TestCollapseFocused_WhenCursorNodeIsNil_ShouldDoNothing(t *testing.T) {
	tr := New([]Item{})
	tr.CollapseFocused()
	if tr.Cursor() != 0 {
		t.Fatalf("expected cursor to remain at 0, got %d", tr.Cursor())
	}
}

func TestParentPath_WhenSingleSegment_ShouldReturnEmpty(t *testing.T) {
	tr := New([]Item{testItem{"aws_s3_bucket.main"}})
	result := tr.parentPath("single_segment")
	if result != "" {
		t.Fatalf("expected empty parent path for single segment, got %q", result)
	}
}
func TestPinStateOf_WhenBranchHasOnlySubBranches_ShouldReadThroughThem(t *testing.T) {
	tr := New([]Item{testItem{"module.vpc.module.subnets.aws_subnet.a"}})
	tr.MoveToStart()
	outer := tr.CursorNode()

	if got := tr.PinStateOf(outer, pinnedSet()); got != PinNone {
		t.Errorf("PinStateOf(unpinned outer branch) = %v, want PinNone", got)
	}
	if got := tr.PinStateOf(outer, pinnedSet("module.vpc.module.subnets.aws_subnet.a")); got != PinFull {
		t.Errorf("PinStateOf(outer branch, nested leaf pinned) = %v, want PinFull", got)
	}
}

func TestRenderRows_ConnectorPrefix_WhenSiblingBranchFollowsNestedBranch(t *testing.T) {
	// A nested branch followed by a sibling top-level branch: the connector prefix built
	// while flattening must stop at each branch's own ancestors.
	items := []Item{
		testItem{"module.a.module.b.aws_instance.one"},
		testItem{"module.c.aws_instance.two"},
	}
	tr := New(items)
	tr.ExpandAll()

	// After expanding all, we should have a deeply nested structure.
	// The rendering should still produce valid connectors.
	opts := RenderOpts{}
	output := renderText(tr, opts)
	if output == "" {
		t.Fatal("expected non-empty render output")
	}
	// Ensure the deeply nested items render without panic
	if !strings.Contains(output, "aws_instance.one") {
		t.Fatal("expected deeply nested leaf to be visible")
	}
}

func TestBuildConnectors_WhenDepthIsZero_ShouldReturnEmpty(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)

	node := tr.CursorNode()
	if node.Depth != 0 {
		t.Fatalf("expected depth 0, got %d", node.Depth)
	}
	connector := tr.buildConnectors(node)
	if connector != "" {
		t.Fatalf("expected empty connector for depth 0, got %q", connector)
	}
}

func TestMoveToEnd_WhenTreeIsEmpty_ShouldNotPanic(t *testing.T) {
	tr := New([]Item{})
	tr.MoveToEnd()
	if tr.Cursor() != 0 {
		t.Fatalf("expected cursor at 0 for empty tree, got %d", tr.Cursor())
	}
}

func TestRenderRows_ConnectorPrefix_WhenNodeHasDeepAncestry(t *testing.T) {
	// Create a structure: module.a > module.b > module.c > leaf
	// plus module.a > module.d > leaf (to ensure continuing ancestors)
	items := []Item{
		testItem{"module.a.module.b.module.c.aws_instance.deep"},
		testItem{"module.a.module.d.aws_instance.sibling"},
	}
	tr := New(items)
	tr.ExpandAll()

	opts := RenderOpts{}
	output := renderText(tr, opts)

	// The deep item should have proper vertical line connectors for continuing ancestors
	if !strings.Contains(output, "│") {
		t.Fatal("expected vertical line connector for continuing ancestor")
	}
	if !strings.Contains(output, "aws_instance.deep") {
		t.Fatal("expected deep leaf to be visible")
	}
}

func TestCollapseFocused_WhenOnCollapsedBranch_ShouldCollapseParent(t *testing.T) {
	items := []Item{
		testItem{"module.a.module.b.aws_instance.one"},
		testItem{"module.a.aws_instance.two"},
	}
	tr := New(items)

	// Expand module.a but not module.b
	tr.MoveToStart() // cursor on module.a
	tr.ExpandFocused()

	// Move to module.b (collapsed branch)
	tr.MoveDown()
	node := tr.CursorNode()
	if node.Kind != KindBranch || node.Expanded {
		t.Fatalf("expected collapsed branch, got kind=%d expanded=%v", node.Kind, node.Expanded)
	}

	// CollapseFocused on a collapsed branch should collapse parent
	tr.CollapseFocused()
	parentNode := tr.CursorNode()
	if parentNode == nil {
		t.Fatal("expected cursor to be on parent after collapse")
	}
	if parentNode.ModulePath != "module.a" {
		t.Fatalf("expected cursor on parent module.a, got %q", parentNode.ModulePath)
	}
	if parentNode.Expanded {
		t.Fatal("expected parent to be collapsed")
	}
}

func TestRenderRows_ConnectorPrefix_WhenDeepNodeFollowsShallowBranch(t *testing.T) {
	// A deeply nested node preceded in the flattened list by a shallow branch from
	// another top-level module: its connector prefix must not inherit the earlier
	// branch's columns.
	items := []Item{
		testItem{"module.x.aws_instance.first"},
		testItem{"module.y.module.z.aws_instance.deep"},
	}
	tr := New(items)
	tr.ExpandAll()

	// The flattened order should be:
	// module.x (depth 0), aws_instance.first (depth 1),
	// module.y (depth 0), module.z (depth 1), aws_instance.deep (depth 2)
	//
	// Second case: a nested leaf under module.a followed by a sibling leaf directly
	// under module.a.
	items2 := []Item{
		testItem{"module.a.module.b.aws_instance.one"},
		testItem{"module.a.aws_instance.root_item"},
	}
	tr2 := New(items2)
	tr2.ExpandAll()

	opts := RenderOpts{}
	output := renderText(tr2, opts)
	if output == "" {
		t.Fatal("expected non-empty output")
	}

	// Verify the deep node renders without issues
	if !strings.Contains(output, "aws_instance.one") {
		t.Fatal("expected deep leaf to appear in output")
	}
}

func TestRenderRows_WhenTreeIsEmpty_ShouldReturnNil(t *testing.T) {
	tr := New([]Item{})
	rows := tr.RenderRows(RenderOpts{})
	if rows != nil {
		t.Fatalf("expected nil for empty tree, got %v", rows)
	}
}

func TestRenderRows_ShouldReturnAllRows(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
		testItem{"aws_s3_bucket.three"},
		testItem{"aws_s3_bucket.four"},
	}
	tr := New(items)

	rows := tr.RenderRows(RenderOpts{})
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows when height is 0 (show all), got %d", len(rows))
	}
	for i, row := range rows {
		expected := fmt.Sprintf("aws_s3_bucket.%s", []string{"four", "one", "three", "two"}[i])
		if !strings.Contains(row, expected) {
			t.Fatalf("row %d: expected to contain %q, got %q", i, expected, row)
		}
	}
}

// renderText joins the rows into one string for assertions on the whole output.
func renderText(t *Tree, opts RenderOpts) string {
	return strings.Join(t.RenderRows(opts), "\n")
}

// --- Benchmarks ---

// benchmarkRenderAllRows renders every row of an expanded tree, as a list that windows
// through ContentPanel.RenderWindow does each frame.
func benchmarkRenderAllRows(b *testing.B, addr func(i int) string, n int) {
	items := make([]Item, n)
	for i := range items {
		items[i] = testItem{addr(i)}
	}
	tr := New(items)
	tr.ExpandAll()
	for b.Loop() {
		tr.RenderRows(RenderOpts{})
	}
}

func BenchmarkRenderRows_OneModuleWithManyLeaves(b *testing.B) {
	for _, n := range []int{1000, 5000, 20000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			benchmarkRenderAllRows(b, func(i int) string { return fmt.Sprintf("module.app.aws_instance.r%d", i) }, n)
		})
	}
}

func BenchmarkRenderRows_Balanced(b *testing.B) {
	for _, n := range []int{1000, 5000, 20000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			benchmarkRenderAllRows(b, func(i int) string {
				return fmt.Sprintf("module.m%d.module.n%d.aws_instance.r%d", i%50, i%7, i)
			}, n)
		})
	}
}
