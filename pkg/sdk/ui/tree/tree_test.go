package tree

import (
	"fmt"
	"strings"
	"testing"
)

type testItem struct{ addr string }

func (t testItem) Address() string { return t.addr }

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
				t.Fatalf("expected all branches to start collapsed, but %q is expanded", n.Path)
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
			if n.Kind == KindLeaf && n.Path == "aws_s3_bucket.main" {
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
				t.Fatalf("expected branch %q to be expanded", n.Path)
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
				t.Fatalf("expected branch %q to be collapsed", n.Path)
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

func TestPinning_WhenTogglingPin_ShouldUpdateState(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens"},
		testItem{"aws_s3_bucket.main"},
		testItem{"aws_s3_bucket.logs"},
	}
	tree := New(items)
	tree.ExpandAll()

	t.Run("ShouldPinCurrentNode", func(t *testing.T) {
		// Move to a leaf
		for i, n := range tree.Nodes() {
			if n.Kind == KindLeaf {
				tree.MoveToStart()
				for j := 0; j < i; j++ {
					tree.MoveDown()
				}
				break
			}
		}
		node := tree.CursorNode()
		path := node.Path
		tree.TogglePin()
		if !tree.IsPinned(path) {
			t.Fatal("expected node to be pinned after TogglePin")
		}
	})

	t.Run("ShouldUnpinOnSecondToggle", func(t *testing.T) {
		node := tree.CursorNode()
		path := node.Path
		// Pin it first (might already be pinned from previous test)
		if !tree.IsPinned(path) {
			tree.TogglePin()
		}
		tree.TogglePin()
		if tree.IsPinned(path) {
			t.Fatal("expected node to be unpinned after second TogglePin")
		}
	})

	t.Run("ShouldReturnPinnedPaths", func(t *testing.T) {
		tree.SetPinned([]string{"aws_s3_bucket.main", "aws_s3_bucket.logs"})
		paths := tree.PinnedPaths()
		if len(paths) != 2 {
			t.Fatalf("expected 2 pinned paths, got %d", len(paths))
		}
		if paths[0] != "aws_s3_bucket.logs" || paths[1] != "aws_s3_bucket.main" {
			t.Fatalf("expected sorted pinned paths, got %v", paths)
		}
	})

	t.Run("ShouldPreserveOrderOnPin", func(t *testing.T) {
		freshTree := New(items)
		freshTree.ExpandAll()
		nodesBefore := freshTree.Nodes()
		var pathsBefore []string
		for _, n := range nodesBefore {
			pathsBefore = append(pathsBefore, n.Path)
		}
		freshTree.SetPinned([]string{"aws_s3_bucket.main"})
		nodesAfter := freshTree.Nodes()
		for i, n := range nodesAfter {
			if n.Path != pathsBefore[i] {
				t.Fatalf("expected order preserved after pin, but position %d changed from %q to %q", i, pathsBefore[i], n.Path)
			}
		}
	})

	t.Run("ShouldSetPinnedFromExternalList", func(t *testing.T) {
		tree.SetPinned([]string{"aws_s3_bucket.main"})
		if !tree.IsPinned("aws_s3_bucket.main") {
			t.Fatal("expected aws_s3_bucket.main to be pinned via SetPinned")
		}
		if tree.IsPinned("aws_s3_bucket.logs") {
			t.Fatal("expected aws_s3_bucket.logs to not be pinned after SetPinned with different list")
		}
	})
}

func TestTogglePin_WhenOnBranch_ShouldCascadeToChildren(t *testing.T) {
	items := []Item{
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high"},
		testItem{"module.cloudwatch.aws_cloudwatch_dashboard.main"},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)

	t.Run("ShouldPinAllChildrenWhenTogglingBranch", func(t *testing.T) {
		tr.MoveToStart()
		node := tr.CursorNode()
		if node.Kind != KindBranch {
			t.Fatal("expected first node to be a branch")
		}
		tr.TogglePin()
		if !tr.IsPinned("module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high") {
			t.Fatal("expected child leaf to be pinned")
		}
		if !tr.IsPinned("module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high") {
			t.Fatal("expected child leaf to be pinned")
		}
		if !tr.IsPinned("module.cloudwatch.aws_cloudwatch_dashboard.main") {
			t.Fatal("expected child leaf to be pinned")
		}
		if tr.IsPinned("aws_s3_bucket.main") {
			t.Fatal("expected unrelated leaf to not be pinned")
		}
	})

	t.Run("ShouldUnpinAllChildrenWhenTogglingFullyPinnedBranch", func(t *testing.T) {
		tr.MoveToStart()
		tr.TogglePin()
		if tr.IsPinned("module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high") {
			t.Fatal("expected child leaf to be unpinned")
		}
		if tr.IsPinned("module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high") {
			t.Fatal("expected child leaf to be unpinned")
		}
	})

	t.Run("ShouldPinAllWhenPartiallyPinned", func(t *testing.T) {
		tr.SetPinned([]string{"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"})
		tr.MoveToStart()
		state := tr.NodePinState("module.cloudwatch")
		if state != PinPartial {
			t.Fatalf("expected partial pin state, got %d", state)
		}
		tr.TogglePin()
		if !tr.IsPinned("module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high") {
			t.Fatal("expected all children to be pinned after toggling partially-pinned branch")
		}
		if !tr.IsPinned("module.cloudwatch.aws_cloudwatch_dashboard.main") {
			t.Fatal("expected all children to be pinned after toggling partially-pinned branch")
		}
	})
}

func TestNodePinState_ShouldReturnCorrectState(t *testing.T) {
	items := []Item{
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high"},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)

	t.Run("ShouldReturnNoneWhenNoPins", func(t *testing.T) {
		state := tr.NodePinState("module.cloudwatch")
		if state != PinNone {
			t.Fatalf("expected PinNone, got %d", state)
		}
	})

	t.Run("ShouldReturnPartialWhenSomePinned", func(t *testing.T) {
		tr.SetPinned([]string{"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high"})
		state := tr.NodePinState("module.cloudwatch")
		if state != PinPartial {
			t.Fatalf("expected PinPartial, got %d", state)
		}
	})

	t.Run("ShouldReturnFullWhenAllPinned", func(t *testing.T) {
		tr.SetPinned([]string{
			"module.cloudwatch.aws_cloudwatch_metric_alarm.cpu_high",
			"module.cloudwatch.aws_cloudwatch_metric_alarm.memory_high",
		})
		state := tr.NodePinState("module.cloudwatch")
		if state != PinFull {
			t.Fatalf("expected PinFull, got %d", state)
		}
	})

	t.Run("ShouldReturnFullForPinnedLeaf", func(t *testing.T) {
		tr.SetPinned([]string{"aws_s3_bucket.main"})
		state := tr.NodePinState("aws_s3_bucket.main")
		if state != PinFull {
			t.Fatalf("expected PinFull for pinned leaf, got %d", state)
		}
	})

	t.Run("ShouldReturnNoneForUnpinnedLeaf", func(t *testing.T) {
		tr.SetPinned([]string{})
		state := tr.NodePinState("aws_s3_bucket.main")
		if state != PinNone {
			t.Fatalf("expected PinNone for unpinned leaf, got %d", state)
		}
	})
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
			if n.Kind == KindLeaf && n.Path == "aws_s3_bucket.main" {
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

func TestRender_WhenTreeHasContent_ShouldIncludeConnectors(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"module.medprev_online_prd.module.medprev_api.aws_lambda_function.api"},
		testItem{"module.cloudwatch.aws_cloudwatch_metric_alarm.bedrock_input_tokens"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)
	tree.ExpandAll()

	opts := RenderOpts{
		Width:  80,
		Height: 30,
	}
	output := tree.Render(opts)

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

func TestRender_WhenCollapsed_ShouldShowCollapsedIndicator(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)
	opts := RenderOpts{Width: 80, Height: 30}
	output := tree.Render(opts)

	if !strings.Contains(output, "▶") {
		t.Fatal("expected output to contain collapsed indicator")
	}
}

func TestRender_WithPins_ShouldShowPinIndicator(t *testing.T) {
	items := []Item{
		testItem{"module.medprev_online_prd.module.postgresql_proxy.aws_db_proxy.this[0]"},
		testItem{"aws_s3_bucket.main"},
		testItem{"aws_s3_bucket.logs"},
	}
	tree := New(items)
	tree.ExpandAll()
	tree.SetPinned([]string{"aws_s3_bucket.main"})

	t.Run("ShouldShowDefaultPinIndicator", func(t *testing.T) {
		opts := RenderOpts{Width: 80, Height: 30}
		output := tree.Render(opts)
		if !strings.Contains(output, "* ") {
			t.Fatal("expected output to contain default pin indicator '* '")
		}
	})

	t.Run("ShouldShowCustomPinIndicator", func(t *testing.T) {
		opts := RenderOpts{
			Width:        80,
			Height:       30,
			PinIndicator: "[P] ",
		}
		output := tree.Render(opts)
		if !strings.Contains(output, "[P] ") {
			t.Fatal("expected output to contain custom pin indicator '[P] '")
		}
	})

	t.Run("ShouldNotShowPinForUnpinnedItems", func(t *testing.T) {
		opts := RenderOpts{Width: 80, Height: 30}
		output := tree.Render(opts)
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

func TestRender_WhenEmpty_ShouldReturnEmptyString(t *testing.T) {
	tree := New([]Item{})
	opts := RenderOpts{Width: 80, Height: 30}
	output := tree.Render(opts)
	if output != "" {
		t.Fatalf("expected empty string for empty tree, got %q", output)
	}
}

func TestRender_WithHeight_ShouldLimitVisibleRows(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
		testItem{"aws_s3_bucket.three"},
		testItem{"aws_s3_bucket.four"},
		testItem{"aws_s3_bucket.five"},
	}
	tree := New(items)

	opts := RenderOpts{Width: 80, Height: 3}
	output := tree.Render(opts)
	lines := strings.Split(output, "\n")
	if len(lines) > 3 {
		t.Fatalf("expected at most 3 lines, got %d", len(lines))
	}
}

func TestRender_WithSelectedStyle_ShouldHighlightCursor(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.main"},
		testItem{"aws_s3_bucket.logs"},
	}
	tree := New(items)

	called := false
	opts := RenderOpts{
		Width:  80,
		Height: 30,
		SelectedStyle: func(s string, width int) string {
			called = true
			return ">>>" + s + "<<<"
		},
	}
	output := tree.Render(opts)

	if !called {
		t.Fatal("expected SelectedStyle to be called")
	}
	if !strings.Contains(output, ">>>") {
		t.Fatal("expected output to contain selected style wrapper")
	}
}

func TestRender_WithCustomRenderers(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.main"},
		testItem{"aws_s3_bucket.main"},
	}
	tree := New(items)
	tree.ExpandAll()

	t.Run("ShouldUseCustomRenderLeaf", func(t *testing.T) {
		opts := RenderOpts{
			Width:  80,
			Height: 30,
			RenderLeaf: func(node *Node, pinned bool) string {
				return "[LEAF:" + node.Label + "]"
			},
		}
		output := tree.Render(opts)
		if !strings.Contains(output, "[LEAF:") {
			t.Fatal("expected output to use custom leaf renderer")
		}
	})

	t.Run("ShouldUseCustomRenderBranch", func(t *testing.T) {
		opts := RenderOpts{
			Width:  80,
			Height: 30,
			RenderBranch: func(node *Node, pinned bool) string {
				return "[BRANCH:" + node.Label + "]"
			},
		}
		output := tree.Render(opts)
		if !strings.Contains(output, "[BRANCH:") {
			t.Fatal("expected output to use custom branch renderer")
		}
	})
}

func TestRender_WithScrolling_ShouldFollowCursor(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.a"},
		testItem{"aws_s3_bucket.b"},
		testItem{"aws_s3_bucket.c"},
		testItem{"aws_s3_bucket.d"},
		testItem{"aws_s3_bucket.e"},
	}
	tree := New(items)

	tree.MoveToEnd()
	opts := RenderOpts{Width: 80, Height: 3}
	output := tree.Render(opts)

	if !strings.Contains(output, "aws_s3_bucket.e") {
		t.Fatal("expected last item to be visible when cursor is at end")
	}
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
	expandedPath := node.Path

	newItems := []Item{
		testItem{"module.vpc.aws_subnet.main"},
		testItem{"module.vpc.aws_subnet.secondary"},
		testItem{"module.vpc.aws_subnet.tertiary"},
		testItem{"aws_s3_bucket.main"},
	}
	tree.SetItems(newItems)

	for _, n := range tree.Nodes() {
		if n.Path == expandedPath && n.Kind == KindBranch {
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
		if nodes[0].Path != "alpha.resource" {
			t.Fatalf("expected first node to be alpha.resource, got %q", nodes[0].Path)
		}
		if nodes[1].Path != "middle.resource" {
			t.Fatalf("expected second node to be middle.resource, got %q", nodes[1].Path)
		}
		if nodes[2].Path != "zebra.resource" {
			t.Fatalf("expected third node to be zebra.resource, got %q", nodes[2].Path)
		}
	})

	t.Run("ShouldPreserveInsertionOrder", func(t *testing.T) {
		tr := New(items, WithPreserveOrder())
		nodes := tr.Nodes()
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
		if nodes[0].Path != "zebra.resource" {
			t.Fatalf("expected first node to be zebra.resource, got %q", nodes[0].Path)
		}
		if nodes[1].Path != "alpha.resource" {
			t.Fatalf("expected second node to be alpha.resource, got %q", nodes[1].Path)
		}
		if nodes[2].Path != "middle.resource" {
			t.Fatalf("expected third node to be middle.resource, got %q", nodes[2].Path)
		}
	})

	t.Run("ShouldNavigateInInsertionOrder", func(t *testing.T) {
		tr := New(items, WithPreserveOrder())
		tr.MoveToStart()
		node := tr.CursorNode()
		if node.Path != "zebra.resource" {
			t.Fatalf("expected cursor to start at zebra.resource, got %q", node.Path)
		}
		tr.MoveDown()
		node = tr.CursorNode()
		if node.Path != "alpha.resource" {
			t.Fatalf("expected cursor at alpha.resource after MoveDown, got %q", node.Path)
		}
		tr.MoveDown()
		node = tr.CursorNode()
		if node.Path != "middle.resource" {
			t.Fatalf("expected cursor at middle.resource after second MoveDown, got %q", node.Path)
		}
	})

	t.Run("ShouldPreserveOrderWithIdentitySplit", func(t *testing.T) {
		identity := func(addr string) []string { return []string{addr} }
		tr := New(items, WithSplitFunc(identity), WithPreserveOrder())
		nodes := tr.Nodes()
		if len(nodes) != 3 {
			t.Fatalf("expected 3 nodes, got %d", len(nodes))
		}
		if nodes[0].Path != "zebra.resource" {
			t.Fatalf("expected first node zebra.resource, got %q", nodes[0].Path)
		}
		if nodes[1].Path != "alpha.resource" {
			t.Fatalf("expected second node alpha.resource, got %q", nodes[1].Path)
		}
		if nodes[2].Path != "middle.resource" {
			t.Fatalf("expected third node middle.resource, got %q", nodes[2].Path)
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

func TestViewport_WhenCursorMovesWithinViewport_ShouldNotScroll(t *testing.T) {
	items := make([]Item, 20)
	for i := range items {
		items[i] = testItem{addr: fmt.Sprintf("aws_s3_bucket.item_%02d", i)}
	}
	tr := New(items)
	viewportHeight := 10

	t.Run("ShouldNotScrollWhenMovingDownWithinViewport", func(t *testing.T) {
		tr.MoveToStart()
		tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		for i := 0; i < viewportHeight-1; i++ {
			tr.MoveDown()
		}
		output := tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		if !strings.Contains(output, "item_00") {
			t.Fatal("expected first item to remain visible while cursor is within viewport")
		}
		if !strings.Contains(output, "item_09") {
			t.Fatal("expected item at cursor position to be visible")
		}
	})

	t.Run("ShouldScrollWhenCursorExceedsViewportBottom", func(t *testing.T) {
		tr.MoveToStart()
		tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		for i := 0; i < viewportHeight; i++ {
			tr.MoveDown()
		}
		output := tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		if strings.Contains(output, "item_00") {
			t.Fatal("expected first item to be scrolled out of view")
		}
		if !strings.Contains(output, "item_10") {
			t.Fatal("expected cursor item to be visible after scroll")
		}
	})

	t.Run("ShouldNotScrollWhenMovingUpWithinViewport", func(t *testing.T) {
		tr.MoveToEnd()
		tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		for i := 0; i < viewportHeight-1; i++ {
			tr.MoveUp()
		}
		output := tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		if !strings.Contains(output, "item_19") {
			t.Fatal("expected last item to remain visible while cursor is within viewport")
		}
	})

	t.Run("ShouldScrollWhenCursorExceedsViewportTop", func(t *testing.T) {
		tr.MoveToEnd()
		tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		for i := 0; i < viewportHeight; i++ {
			tr.MoveUp()
		}
		output := tr.Render(RenderOpts{Width: 80, Height: viewportHeight})
		if !strings.Contains(output, "item_09") {
			t.Fatal("expected cursor item to be visible after scrolling up")
		}
		if strings.Contains(output, "item_19") {
			t.Fatal("expected last item to be scrolled out of view")
		}
	})
}

func TestViewport_WhenAllItemsFitInView_ShouldShowAll(t *testing.T) {
	items := make([]Item, 5)
	for i := range items {
		items[i] = testItem{addr: fmt.Sprintf("aws_s3_bucket.item_%02d", i)}
	}
	tr := New(items)

	output := tr.Render(RenderOpts{Width: 80, Height: 10})
	lines := strings.Split(output, "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines when all items fit, got %d", len(lines))
	}
	for i := 0; i < 5; i++ {
		expected := fmt.Sprintf("item_%02d", i)
		if !strings.Contains(output, expected) {
			t.Fatalf("expected %q to be visible", expected)
		}
	}
}

func TestViewOffset_WhenCursorAtStart_ShouldReturnZero(t *testing.T) {
	items := make([]Item, 20)
	for i := range items {
		items[i] = testItem{addr: fmt.Sprintf("item_%02d", i)}
	}
	tr := New(items)
	tr.MoveToStart()

	offset := tr.ViewOffset(10)
	if offset != 0 {
		t.Fatalf("expected offset 0 when cursor is at start, got %d", offset)
	}
}

func TestViewOffset_WhenCursorAtEnd_ShouldReturnMaxOffset(t *testing.T) {
	items := make([]Item, 20)
	for i := range items {
		items[i] = testItem{addr: fmt.Sprintf("item_%02d", i)}
	}
	tr := New(items)
	tr.MoveToEnd()

	offset := tr.ViewOffset(10)
	expected := 10 // 20 items - 10 viewport height
	if offset != expected {
		t.Fatalf("expected offset %d when cursor is at end, got %d", expected, offset)
	}
}

func TestViewOffset_WhenMovingBackFromEnd_ShouldNotScrollUntilTopEdge(t *testing.T) {
	items := make([]Item, 20)
	for i := range items {
		items[i] = testItem{addr: fmt.Sprintf("item_%02d", i)}
	}
	tr := New(items)
	tr.MoveToEnd()
	tr.ViewOffset(10) // establish viewport at bottom

	// Move up 5 positions (still within the viewport)
	for i := 0; i < 5; i++ {
		tr.MoveUp()
	}
	offset := tr.ViewOffset(10)
	if offset != 10 {
		t.Fatalf("expected offset to remain at 10 while cursor is within viewport, got %d", offset)
	}

	// Move up 5 more (now cursor is at pos 9, hits top edge of viewport at offset 10)
	for i := 0; i < 5; i++ {
		tr.MoveUp()
	}
	offset = tr.ViewOffset(10)
	if offset != 9 {
		t.Fatalf("expected offset to decrease to 9 when cursor hits top edge, got %d", offset)
	}
}

func TestRender_WhenHeightIsZero_ShouldShowAllItems(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
		testItem{"aws_s3_bucket.three"},
	}
	tr := New(items)

	opts := RenderOpts{Width: 80, Height: 0}
	output := tr.Render(opts)
	lines := strings.Split(output, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines when height is 0 (show all), got %d", len(lines))
	}
}

func TestRender_WhenHeightIsNegative_ShouldShowAllItems(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
	}
	tr := New(items)

	opts := RenderOpts{Width: 80, Height: -1}
	output := tr.Render(opts)
	lines := strings.Split(output, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines when height is negative (show all), got %d", len(lines))
	}
}

func TestRender_WithTruncateRow_ShouldTruncateNonSelectedRows(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.very_long_name_that_should_be_truncated"},
		testItem{"aws_s3_bucket.second"},
	}
	tr := New(items)

	opts := RenderOpts{
		Width:  20,
		Height: 10,
		TruncateRow: func(s string, width int) string {
			if len(s) > width {
				return s[:width-3] + "..."
			}
			return s
		},
	}
	output := tr.Render(opts)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "very_long_name_that_should_be_truncated") {
			t.Fatal("expected long row to be truncated")
		}
	}
	_ = output
}

func TestRender_WithPinIndicators_ShouldShowStateSpecificIndicators(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.a"},
		testItem{"module.vpc.aws_subnet.b"},
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)
	tr.ExpandAll()
	tr.SetPinned([]string{"module.vpc.aws_subnet.a"})

	indicators := &PinIndicators{
		None:    "[ ] ",
		Full:    "[*] ",
		Partial: "[-] ",
	}

	opts := RenderOpts{
		Width:         80,
		Height:        30,
		PinIndicators: indicators,
	}
	output := tr.Render(opts)

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

	t.Run("ShouldShowNoneForPartialStateOnLeaf", func(t *testing.T) {
		// A leaf cannot be partially pinned, but let's ensure the code path
		// pinIndicatorFor with PinPartial + KindLeaf returns None
		tr2 := New(items)
		tr2.ExpandAll()
		tr2.SetPinned([]string{"module.vpc.aws_subnet.a"})
		// The aws_subnet.b leaf is not pinned and cannot be partial
		// But the branch module.vpc is partial
		// We need to specifically test the pinIndicatorFor branch for PinPartial on leaf.
		// Since NodePinState checks the internal tree, a leaf can only be PinNone or PinFull.
		// The code path exists defensively. We test it by calling directly.
		result := tr2.pinIndicatorFor(PinPartial, KindLeaf, opts)
		if result != "[ ] " {
			t.Fatalf("expected PinPartial on leaf to show None indicator [ ], got %q", result)
		}
	})

	t.Run("ShouldShowFullIndicatorForFullyPinnedBranch", func(t *testing.T) {
		tr3 := New(items)
		tr3.ExpandAll()
		tr3.SetPinned([]string{"module.vpc.aws_subnet.a", "module.vpc.aws_subnet.b"})
		opts3 := RenderOpts{
			Width:         80,
			Height:        30,
			PinIndicators: indicators,
		}
		output3 := tr3.Render(opts3)
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

func TestViewOffset_WhenHeightIsZero_ShouldReturnZero(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
	}
	tr := New(items)

	offset := tr.ViewOffset(0)
	if offset != 0 {
		t.Fatalf("expected ViewOffset to return 0 for height 0, got %d", offset)
	}
}

func TestViewOffset_WhenHeightIsNegative_ShouldReturnZero(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
	}
	tr := New(items)

	offset := tr.ViewOffset(-5)
	if offset != 0 {
		t.Fatalf("expected ViewOffset to return 0 for negative height, got %d", offset)
	}
}

func TestViewOffset_WhenHeightExceedsFlattenedCount_ShouldClampToZero(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
	}
	tr := New(items)
	tr.MoveToEnd()

	offset := tr.ViewOffset(100)
	if offset != 0 {
		t.Fatalf("expected ViewOffset to clamp to 0 when height exceeds items, got %d", offset)
	}
}

func TestViewOffset_WhenCursorAboveViewport_ShouldScrollUp(t *testing.T) {
	items := make([]Item, 20)
	for i := range items {
		items[i] = testItem{addr: "item_" + string(rune('a'+i))}
	}
	tr := New(items)

	// Move to end first to set viewport offset high
	tr.MoveToEnd()
	tr.ViewOffset(10)

	// Now move to start without calling ViewOffset in between
	tr.MoveToStart()
	offset := tr.ViewOffset(10)
	if offset != 0 {
		t.Fatalf("expected offset 0 when cursor is at start, got %d", offset)
	}
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

func TestTogglePin_WhenCursorIsNil_ShouldDoNothing(t *testing.T) {
	tr := New([]Item{})
	tr.TogglePin()
	paths := tr.PinnedPaths()
	if len(paths) != 0 {
		t.Fatalf("expected no pinned paths on empty tree, got %v", paths)
	}
}

func TestTogglePin_WhenBranchNotFound_ShouldDoNothing(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.main"},
	}
	tr := New(items)
	tr.MoveToStart()
	node := tr.CursorNode()
	if node.Kind != KindBranch {
		t.Fatal("expected branch node at start")
	}

	// Corrupt the path to simulate findNode returning nil
	originalPath := node.Path
	node.Path = "nonexistent.path"
	tr.TogglePin()
	paths := tr.PinnedPaths()
	if len(paths) != 0 {
		t.Fatalf("expected no pinned paths when node not found, got %v", paths)
	}
	node.Path = originalPath
}

func TestSetPinRecursive_WhenUnpinning_ShouldRemoveAllChildren(t *testing.T) {
	items := []Item{
		testItem{"module.vpc.aws_subnet.a"},
		testItem{"module.vpc.aws_subnet.b"},
		testItem{"module.vpc.aws_subnet.c"},
	}
	tr := New(items)

	// Pin all first
	tr.SetPinned([]string{
		"module.vpc.aws_subnet.a",
		"module.vpc.aws_subnet.b",
		"module.vpc.aws_subnet.c",
	})

	// Move to branch and toggle to unpin all
	tr.MoveToStart()
	node := tr.CursorNode()
	if node.Kind != KindBranch {
		t.Fatal("expected branch at start")
	}
	tr.TogglePin()

	if tr.IsPinned("module.vpc.aws_subnet.a") {
		t.Fatal("expected aws_subnet.a to be unpinned")
	}
	if tr.IsPinned("module.vpc.aws_subnet.b") {
		t.Fatal("expected aws_subnet.b to be unpinned")
	}
	if tr.IsPinned("module.vpc.aws_subnet.c") {
		t.Fatal("expected aws_subnet.c to be unpinned")
	}
}

func TestNodePinState_WhenBranchHasNoLeaves_ShouldReturnNone(t *testing.T) {
	// Create a tree where a branch has only sub-branches but no direct leaves
	items := []Item{
		testItem{"module.vpc.module.subnets.aws_subnet.a"},
	}
	tr := New(items)

	// The "module.vpc" branch has no direct items, only child "module.subnets"
	state := tr.nodePinState(tr.root.children[0])
	// Should be PinNone since nothing is pinned
	if state != PinNone {
		t.Fatalf("expected PinNone for unpinned branch, got %d", state)
	}
}

func TestGetAncestorContinuations_WhenAncestorSearchHitsSmallerDepth(t *testing.T) {
	// Build a tree that creates the n.Depth < d break condition in getAncestorContinuations.
	// This happens when walking backwards we encounter a node with depth less than
	// the ancestor depth we're searching for, meaning we've passed the ancestor.
	items := []Item{
		testItem{"module.a.module.b.aws_instance.one"},
		testItem{"module.c.aws_instance.two"},
	}
	tr := New(items)
	tr.ExpandAll()

	// After expanding all, we should have a deeply nested structure.
	// The rendering should still produce valid connectors.
	opts := RenderOpts{Width: 80, Height: 30}
	output := tr.Render(opts)
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

func TestIndexOf_WhenNodeNotInFlattened_ShouldReturnNegativeOne(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.main"},
	}
	tr := New(items)

	fakeNode := &Node{
		Kind:  KindLeaf,
		Label: "fake",
		Path:  "fake.path",
	}
	idx := tr.indexOf(fakeNode)
	if idx != -1 {
		t.Fatalf("expected -1 for node not in tree, got %d", idx)
	}
}

func TestMoveToEnd_WhenTreeIsEmpty_ShouldNotPanic(t *testing.T) {
	tr := New([]Item{})
	tr.MoveToEnd()
	if tr.Cursor() != 0 {
		t.Fatalf("expected cursor at 0 for empty tree, got %d", tr.Cursor())
	}
}

func TestAdjustViewport_WhenViewOffsetExceedsMax_ShouldClamp(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
		testItem{"aws_s3_bucket.three"},
	}
	tr := New(items)

	// Manually set viewOffset beyond max to test clamping
	tr.viewOffset = 100
	tr.adjustViewport(5)
	if tr.viewOffset != 0 {
		t.Fatalf("expected viewOffset to be clamped to 0, got %d", tr.viewOffset)
	}
}

func TestAdjustViewport_WhenViewOffsetIsNegative_ShouldClampToZero(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
	}
	tr := New(items)

	tr.viewOffset = -5
	tr.adjustViewport(10)
	if tr.viewOffset != 0 {
		t.Fatalf("expected viewOffset to be clamped to 0, got %d", tr.viewOffset)
	}
}

func TestRender_WhenEndIdxExceedsFlattenedLength_ShouldClamp(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
	}
	tr := New(items)

	// Height larger than item count; endIdx would exceed flattened length
	opts := RenderOpts{Width: 80, Height: 100}
	output := tr.Render(opts)
	lines := strings.Split(output, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestGetAncestorContinuations_WhenNodeHasDeepAncestry(t *testing.T) {
	// Create a structure: module.a > module.b > module.c > leaf
	// plus module.a > module.d > leaf (to ensure continuing ancestors)
	items := []Item{
		testItem{"module.a.module.b.module.c.aws_instance.deep"},
		testItem{"module.a.module.d.aws_instance.sibling"},
	}
	tr := New(items)
	tr.ExpandAll()

	opts := RenderOpts{Width: 80, Height: 30}
	output := tr.Render(opts)

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
	if parentNode.Path != "module.a" {
		t.Fatalf("expected cursor on parent module.a, got %q", parentNode.Path)
	}
	if parentNode.Expanded {
		t.Fatal("expected parent to be collapsed")
	}
}

func TestSetPinRecursive_WhenUnpinningNestedBranch_ShouldDeleteAllPins(t *testing.T) {
	items := []Item{
		testItem{"module.a.module.b.aws_instance.one"},
		testItem{"module.a.module.b.aws_instance.two"},
		testItem{"module.a.module.c.aws_instance.three"},
	}
	tr := New(items)

	// Pin all leaves via the top-level branch
	tr.MoveToStart()
	node := tr.CursorNode()
	if node.Kind != KindBranch {
		t.Fatal("expected branch at start")
	}
	tr.TogglePin() // pins all

	if !tr.IsPinned("module.a.module.b.aws_instance.one") {
		t.Fatal("expected nested leaf to be pinned")
	}
	if !tr.IsPinned("module.a.module.c.aws_instance.three") {
		t.Fatal("expected nested leaf to be pinned")
	}

	// Now toggle again to unpin all (goes through delete path in setPinRecursive for nested)
	tr.TogglePin()
	if tr.IsPinned("module.a.module.b.aws_instance.one") {
		t.Fatal("expected nested leaf to be unpinned after toggle")
	}
	if tr.IsPinned("module.a.module.b.aws_instance.two") {
		t.Fatal("expected nested leaf to be unpinned after toggle")
	}
	if tr.IsPinned("module.a.module.c.aws_instance.three") {
		t.Fatal("expected nested leaf to be unpinned after toggle")
	}
}

func TestNodePinState_WhenBranchHasEmptySubtree_ShouldReturnNone(t *testing.T) {
	tr := New([]Item{testItem{"module.a.aws_instance.one"}})

	// Directly test nodePinState with an empty treeNode (no items, no children)
	emptyNode := &treeNode{path: "empty", label: "empty"}
	state := tr.nodePinState(emptyNode)
	if state != PinNone {
		t.Fatalf("expected PinNone for empty branch (total=0), got %d", state)
	}
}

func TestViewOffset_WhenViewOffsetManuallySetNegative_ShouldClamp(t *testing.T) {
	items := make([]Item, 5)
	for i := range items {
		items[i] = testItem{addr: "item_" + string(rune('a'+i))}
	}
	tr := New(items)

	// Manually corrupt viewOffset to negative
	tr.viewOffset = -10
	offset := tr.ViewOffset(3)
	if offset < 0 {
		t.Fatalf("expected ViewOffset to never return negative, got %d", offset)
	}
}

func TestViewOffset_WhenCursorBelowViewport_ShouldScrollDown(t *testing.T) {
	items := make([]Item, 20)
	for i := range items {
		items[i] = testItem{addr: "item_" + string(rune('a'+i))}
	}
	tr := New(items)

	// Set cursor to item beyond the viewport height without calling ViewOffset before
	tr.MoveToStart()
	for i := 0; i < 15; i++ {
		tr.MoveDown()
	}

	// viewOffset is still 0, cursor is at 15, height is 5
	// This should trigger: cursor >= viewOffset+height
	offset := tr.ViewOffset(5)
	if offset == 0 {
		t.Fatal("expected ViewOffset to scroll down when cursor exceeds viewport")
	}
	// cursor(15) - height(5) + 1 = 11
	if offset != 11 {
		t.Fatalf("expected offset 11, got %d", offset)
	}
}

func TestGetAncestorContinuations_WhenWalkingBackHitsSmallerDepth(t *testing.T) {
	// To trigger the n.Depth < d break, we need a structure where
	// walking backwards from a node we encounter a node whose depth
	// is less than the ancestor depth we're looking for.
	// This happens when a deeply nested node is preceded by a shallow node
	// in the flattened list.
	//
	// Structure: module.a (depth 0) -> module.b (depth 1) -> leaf (depth 2)
	//            module.x (depth 0) -> leaf (depth 1)
	//
	// When rendering the leaf at depth 2 under module.b, we look for ancestor
	// at depth 1 (module.b). Walking backwards we find module.b.
	// Then for depth 0, we look for module.a. But what if the flattened order
	// puts module.x between them? Actually flattened is DFS so siblings come after.
	//
	// Better approach: Create structure where a depth-1 node under one branch
	// is preceded by a depth-0 node from another branch.
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
	// For aws_instance.deep at depth 2, getAncestorContinuations looks for:
	// d=0: walks back, finds module.z(depth1), finds module.y(depth0) -> !IsLast
	// d=1: walks back, finds module.z(depth1) -> !IsLast
	//
	// But to trigger depth < d break, we need the backwards walk to hit a node
	// whose depth is LESS than d before finding the ancestor at depth d.
	// This happens for d=1 if we walk back and hit a depth-0 node first.
	//
	// Let's build a case: sibling items at depth 0 followed by a nested node.
	items2 := []Item{
		testItem{"module.a.module.b.aws_instance.one"},
		testItem{"module.a.aws_instance.root_item"},
	}
	tr2 := New(items2)
	tr2.ExpandAll()

	opts := RenderOpts{Width: 80, Height: 30}
	output := tr2.Render(opts)
	if output == "" {
		t.Fatal("expected non-empty output")
	}

	// Verify the deep node renders without issues
	if !strings.Contains(output, "aws_instance.one") {
		t.Fatal("expected deep leaf to appear in output")
	}
}

func TestAdjustViewport_WhenCursorBelowViewport_ShouldScrollDown(t *testing.T) {
	items := make([]Item, 10)
	for i := range items {
		items[i] = testItem{addr: "item_" + string(rune('a'+i))}
	}
	tr := New(items)

	// Move cursor to position 8, keep viewOffset at 0, height 5
	tr.MoveToStart()
	for i := 0; i < 8; i++ {
		tr.MoveDown()
	}
	// viewOffset is still 0, cursor at 8, height 5
	// cursor(8) >= viewOffset(0) + height(5) should trigger scroll
	tr.adjustViewport(5)
	// Expected: viewOffset = cursor - height + 1 = 8 - 5 + 1 = 4
	if tr.viewOffset != 4 {
		t.Fatalf("expected viewOffset 4, got %d", tr.viewOffset)
	}
}

func TestAdjustViewport_WhenViewOffsetExceedsMaxAfterTreeShrinks_ShouldClampToMax(t *testing.T) {
	items := make([]Item, 10)
	for i := range items {
		items[i] = testItem{addr: "item_" + string(rune('a'+i))}
	}
	tr := New(items)

	// Simulate: viewOffset was high (e.g., from scrolling), then tree shrinks
	// but cursor is still within bounds.
	// cursor=4, viewOffset=3, height=5 → cursor NOT below viewport (4 < 3+5=8)
	// But if flattened shrinks to 5 items: maxOffset = 5-5 = 0
	// viewOffset(3) > maxOffset(0) → TRUE → clamp!
	tr.cursor = 4
	tr.viewOffset = 3
	// Shrink flattened to 5 items
	tr.flattened = tr.flattened[:5]

	tr.adjustViewport(5)
	if tr.viewOffset != 0 {
		t.Fatalf("expected viewOffset clamped to 0 (maxOffset), got %d", tr.viewOffset)
	}
}

func TestViewOffset_WhenViewOffsetExceedsMaxAfterShrink_ShouldClamp(t *testing.T) {
	items := make([]Item, 10)
	for i := range items {
		items[i] = testItem{addr: "item_" + string(rune('a'+i))}
	}
	tr := New(items)

	// Set up: cursor at 4, viewOffset at 3
	// height = 5, so cursor(4) < viewOffset(3)+height(5)=8 → no scroll down
	// cursor(4) >= viewOffset(3) → no scroll up
	// Then shrink: maxOffset = 5 - 5 = 0; viewOffset(3) > 0 → clamp
	tr.cursor = 4
	tr.viewOffset = 3
	tr.flattened = tr.flattened[:5]

	offset := tr.ViewOffset(5)
	if offset != 0 {
		t.Fatalf("expected ViewOffset to clamp to 0 after shrink, got %d", offset)
	}
}

func TestGetAncestorContinuations_DepthLessThanDBreak(t *testing.T) {
	// The n.Depth < d break in getAncestorContinuations is a defensive guard
	// that fires when walking backwards we hit a node at a shallower depth than
	// what we're searching for. We exercise it by manipulating flattened directly.
	items := []Item{
		testItem{"module.a.module.b.aws_instance.one"},
	}
	tr := New(items)
	tr.ExpandAll()

	// Inject a fake node into flattened that creates a depth gap.
	// Insert a depth-0 node just before a depth-2 leaf to force the break.
	fakeShallow := &Node{Kind: KindBranch, Label: "fake", Path: "fake", Depth: 0, IsLast: false}
	deepLeaf := &Node{Kind: KindLeaf, Label: "test", Path: "test.leaf", Depth: 3, IsLast: true}

	tr.flattened = append(tr.flattened, fakeShallow, deepLeaf)

	// Call getAncestorContinuations on deepLeaf (depth 3)
	// Walking back from deepLeaf's idx, for d=2, we'll hit fakeShallow (depth 0) which is < 2
	result := tr.getAncestorContinuations(deepLeaf)
	if len(result) != 3 {
		t.Fatalf("expected 3 depth levels, got %d", len(result))
	}
}

func TestRenderRows_WhenTreeIsEmpty_ShouldReturnNil(t *testing.T) {
	tr := New([]Item{})
	rows := tr.RenderRows(RenderOpts{Width: 80, Height: 5})
	if rows != nil {
		t.Fatalf("expected nil for empty tree, got %v", rows)
	}
}

func TestRenderRows_WhenHeightIsZero_ShouldReturnAllRows(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
		testItem{"aws_s3_bucket.three"},
		testItem{"aws_s3_bucket.four"},
	}
	tr := New(items)

	rows := tr.RenderRows(RenderOpts{Width: 80, Height: 0})
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

func TestRenderRows_WhenHeightIsLimited_ShouldReturnWindowedSlice(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.a"},
		testItem{"aws_s3_bucket.b"},
		testItem{"aws_s3_bucket.c"},
		testItem{"aws_s3_bucket.d"},
		testItem{"aws_s3_bucket.e"},
	}
	tr := New(items)

	rows := tr.RenderRows(RenderOpts{Width: 80, Height: 3})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows with limited height, got %d", len(rows))
	}
	if !strings.Contains(rows[0], "aws_s3_bucket.a") {
		t.Fatalf("expected first row to contain aws_s3_bucket.a, got %q", rows[0])
	}
	if !strings.Contains(rows[2], "aws_s3_bucket.c") {
		t.Fatalf("expected third row to contain aws_s3_bucket.c, got %q", rows[2])
	}
}

func TestRenderRows_WhenCursorAtEnd_ShouldReturnScrolledWindow(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.a"},
		testItem{"aws_s3_bucket.b"},
		testItem{"aws_s3_bucket.c"},
		testItem{"aws_s3_bucket.d"},
		testItem{"aws_s3_bucket.e"},
	}
	tr := New(items)
	tr.MoveToEnd()

	rows := tr.RenderRows(RenderOpts{Width: 80, Height: 3})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !strings.Contains(rows[2], "aws_s3_bucket.e") {
		t.Fatalf("expected last row to contain aws_s3_bucket.e, got %q", rows[2])
	}
	if strings.Contains(rows[0], "aws_s3_bucket.a") {
		t.Fatal("expected first item to be scrolled out of view")
	}
}

func TestRenderRows_ShouldNotApplyTruncationOrSelectedStyle(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.very_long_name_that_exceeds_width"},
	}
	tr := New(items)

	truncateCalled := false
	selectedCalled := false
	rows := tr.RenderRows(RenderOpts{
		Width:  10,
		Height: 5,
		TruncateRow: func(s string, width int) string {
			truncateCalled = true
			return s[:5]
		},
		SelectedStyle: func(s string, width int) string {
			selectedCalled = true
			return ">>>" + s
		},
	})

	if truncateCalled {
		t.Fatal("expected TruncateRow to NOT be called by RenderRows")
	}
	if selectedCalled {
		t.Fatal("expected SelectedStyle to NOT be called by RenderRows")
	}
	if !strings.Contains(rows[0], "very_long_name_that_exceeds_width") {
		t.Fatalf("expected full untruncated content, got %q", rows[0])
	}
}

func TestRenderRows_WhenHeightExceedsItems_ShouldReturnAllRows(t *testing.T) {
	items := []Item{
		testItem{"aws_s3_bucket.one"},
		testItem{"aws_s3_bucket.two"},
	}
	tr := New(items)

	rows := tr.RenderRows(RenderOpts{Width: 80, Height: 100})
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows when height exceeds item count, got %d", len(rows))
	}
}
