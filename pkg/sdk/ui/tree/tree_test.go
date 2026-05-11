package tree

import (
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
