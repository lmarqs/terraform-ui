package tree

import (
	"strings"
	"testing"
)

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
