package tree

import (
	"sort"
	"strings"
)

// PinState represents the pin/selection state of a node.
type PinState int

const (
	PinNone    PinState = iota // no children pinned
	PinPartial                 // some children pinned
	PinFull                    // all children pinned (or leaf is pinned)
)

// Item is implemented by anything that can appear in a tree.
type Item interface {
	Address() string
}

// NodeKind distinguishes branch nodes from leaf nodes.
type NodeKind int

const (
	KindBranch NodeKind = iota
	KindLeaf
)

// Node represents a visible row in the flattened tree.
type Node struct {
	Kind       NodeKind
	Label      string
	ModulePath string // full module path; branches only
	Depth      int
	Expanded   bool
	Count      int  // descendant leaf count (branches only)
	IsLast     bool // last child of its parent (for connector rendering)
	Item       Item // non-nil for leaves only

	// branch is the subtree this row heads. Non-nil for branches only, so a
	// row can answer questions about its descendants without being re-located
	// by path.
	branch *treeNode
}

// Address returns the resource address of a leaf row, taken from the Item the
// row already carries. A branch row heads a module rather than a resource and
// has no address, so it returns "" — callers acting on a single resource must
// read that as "nothing to act on", and callers wanting the module itself want
// ModulePath.
func (n *Node) Address() string {
	if n.Kind == KindLeaf {
		return n.Item.Address()
	}
	return ""
}

// Tree is an interactive tree navigator built from flat addressed items.
type Tree struct {
	root          *treeNode
	flattened     []*Node
	cursor        int
	viewOffset    int
	expanded      map[string]bool
	splitFunc     func(string) []string
	preserveOrder bool
}

// treeNode is the internal recursive structure used for building.
type treeNode struct {
	label    string
	path     string
	children []*treeNode
	items    []Item // leaf items directly under this node
}

// Option configures a Tree.
type Option func(*Tree)

// WithSplitFunc sets a custom function for splitting addresses into segments.
func WithSplitFunc(fn func(string) []string) Option {
	return func(t *Tree) { t.splitFunc = fn }
}

// WithPreserveOrder disables sorting so items stay in insertion order.
func WithPreserveOrder() Option {
	return func(t *Tree) { t.preserveOrder = true }
}

// New creates a tree from flat items.
func New(items []Item, opts ...Option) *Tree {
	t := &Tree{
		expanded:  make(map[string]bool),
		splitFunc: SplitTerraform,
	}
	for _, opt := range opts {
		opt(t)
	}
	t.SetItems(items)
	return t
}

// SetItems replaces the tree data and rebuilds. Preserves expansion state.
func (t *Tree) SetItems(items []Item) {
	t.root = &treeNode{path: ""}
	for _, item := range items {
		segments := t.splitFunc(item.Address())
		t.insert(segments, item)
	}
	t.sortNodes(t.root)
	t.flatten()
}

func (t *Tree) insert(segments []string, item Item) {
	current := t.root
	pathSoFar := ""
	for i, seg := range segments {
		if i == len(segments)-1 {
			current.items = append(current.items, item)
		} else {
			if pathSoFar != "" {
				pathSoFar += "." + seg
			} else {
				pathSoFar = seg
			}
			child := t.findChild(current, seg)
			if child == nil {
				child = &treeNode{label: seg, path: pathSoFar}
				current.children = append(current.children, child)
			}
			current = child
		}
	}
}

func (t *Tree) findChild(parent *treeNode, label string) *treeNode {
	for _, c := range parent.children {
		if c.label == label {
			return c
		}
	}
	return nil
}

func (t *Tree) sortNodes(node *treeNode) {
	if t.preserveOrder {
		return
	}
	sort.SliceStable(node.children, func(i, j int) bool {
		return node.children[i].label < node.children[j].label
	})
	sort.SliceStable(node.items, func(i, j int) bool {
		return node.items[i].Address() < node.items[j].Address()
	})
	for _, c := range node.children {
		t.sortNodes(c)
	}
}

func (t *Tree) countLeaves(node *treeNode) int {
	count := len(node.items)
	for _, c := range node.children {
		count += t.countLeaves(c)
	}
	return count
}

func (t *Tree) flatten() {
	t.flattened = t.flattened[:0]
	t.walkChildren(t.root, 0)
	if t.cursor >= len(t.flattened) {
		t.cursor = len(t.flattened) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *Tree) walkChildren(node *treeNode, depth int) {
	totalChildren := len(node.children) + len(node.items)
	idx := 0

	for _, child := range node.children {
		idx++
		isLast := idx == totalChildren
		expanded := t.expanded[child.path]
		t.flattened = append(t.flattened, &Node{
			Kind:       KindBranch,
			Label:      child.label,
			ModulePath: child.path,
			Depth:      depth,
			Expanded:   expanded,
			Count:      t.countLeaves(child),
			IsLast:     isLast,
			branch:     child,
		})
		if expanded {
			t.walkChildren(child, depth+1)
		}
	}

	for _, item := range node.items {
		idx++
		isLast := idx == totalChildren
		t.flattened = append(t.flattened, &Node{
			Kind:   KindLeaf,
			Label:  leafLabel(item.Address(), t.splitFunc),
			Depth:  depth,
			IsLast: isLast,
			Item:   item,
		})
	}
}

func leafLabel(address string, splitFn func(string) []string) string {
	segments := splitFn(address)
	if len(segments) == 0 {
		return address
	}
	return segments[len(segments)-1]
}

// Navigation

func (t *Tree) MoveUp() {
	if t.cursor > 0 {
		t.cursor--
	}
}

func (t *Tree) MoveDown() {
	if t.cursor < len(t.flattened)-1 {
		t.cursor++
	}
}

func (t *Tree) MoveToStart() { t.cursor = 0 }

func (t *Tree) MoveToEnd() {
	if len(t.flattened) > 0 {
		t.cursor = len(t.flattened) - 1
	}
}

// Expand/Collapse

func (t *Tree) Toggle() {
	if n := t.CursorNode(); n != nil && n.Kind == KindBranch {
		t.expanded[n.ModulePath] = !t.expanded[n.ModulePath]
		t.flatten()
	}
}

func (t *Tree) ExpandFocused() {
	if n := t.CursorNode(); n != nil && n.Kind == KindBranch {
		t.expanded[n.ModulePath] = true
		t.flatten()
	}
}

func (t *Tree) CollapseFocused() {
	n := t.CursorNode()
	if n == nil {
		return
	}
	if n.Kind == KindBranch && t.expanded[n.ModulePath] {
		t.expanded[n.ModulePath] = false
		t.flatten()
		return
	}
	// On leaf or collapsed branch: collapse parent. Both kinds sit at a dotted
	// position in the hierarchy — a leaf's address shares every segment with
	// the branch above it but the last — so the parent lookup takes whichever
	// this row has.
	position := n.ModulePath
	if n.Kind == KindLeaf {
		position = n.Address()
	}
	parent := t.parentPath(position)
	if parent != "" {
		t.expanded[parent] = false
		t.flatten()
		// Move cursor to the collapsed parent
		for i, node := range t.flattened {
			if node.ModulePath == parent {
				t.cursor = i
				break
			}
		}
	}
}

func (t *Tree) parentPath(path string) string {
	segments := t.splitFunc(path)
	if len(segments) <= 1 {
		return ""
	}
	parentSegments := segments[:len(segments)-1]
	result := ""
	for _, seg := range parentSegments {
		if result != "" {
			result += "." + seg
		} else {
			result = seg
		}
	}
	return result
}

func (t *Tree) ExpandAll() {
	for {
		changed := false
		for _, n := range t.flattened {
			if n.Kind == KindBranch && !t.expanded[n.ModulePath] {
				t.expanded[n.ModulePath] = true
				changed = true
			}
		}
		t.flatten()
		if !changed {
			break
		}
	}
}

func (t *Tree) CollapseAll() {
	t.expanded = make(map[string]bool)
	t.flatten()
	t.cursor = 0
}

// Pinning

// PinAddresses returns the resource addresses a pin toggle on the supplied row
// affects, in the order the rows appear. A leaf row resolves to its own
// address. A branch row resolves to every descendant leaf, expanded or not,
// because a branch has only a ModulePath — not a resource address, and never a
// key in the pinned set. Callers must route a row through this instead of
// assembling addresses themselves, or a branch pin lands on an address no row
// will ever report as pinned.
func (t *Tree) PinAddresses(node *Node) []string {
	if node.Kind == KindLeaf {
		return []string{node.Address()}
	}
	return t.leafAddresses(node.branch, nil)
}

func (t *Tree) leafAddresses(node *treeNode, acc []string) []string {
	for _, child := range node.children {
		acc = t.leafAddresses(child, acc)
	}
	for _, item := range node.items {
		acc = append(acc, item.Address())
	}
	return acc
}

// PinStateOf reports how much of a row is pinned, given the caller's pin
// predicate. The tree holds no pin state of its own — Context owns Pins
// (ADR-0018) and hands them in per render — so a row can never disagree with
// the pin set the way a cached copy could go stale.
//
// A leaf row is None or Full. A branch row is Full only when every descendant
// leaf is pinned, and Partial while some are.
func (t *Tree) PinStateOf(node *Node, pinned func(address string) bool) PinState {
	if pinned == nil {
		return PinNone
	}
	addresses := t.PinAddresses(node)
	count := 0
	for _, a := range addresses {
		if pinned(a) {
			count++
		}
	}
	switch {
	case count == len(addresses):
		return PinFull
	case count > 0:
		return PinPartial
	default:
		return PinNone
	}
}

// Query methods

func (t *Tree) Cursor() int       { return t.cursor }
func (t *Tree) VisibleCount() int { return len(t.flattened) }

// ViewOffset returns the current viewport offset, adjusting if needed for the given height.
func (t *Tree) ViewOffset(height int) int {
	if height <= 0 {
		return 0
	}
	if t.cursor < t.viewOffset {
		t.viewOffset = t.cursor
	}
	if t.cursor >= t.viewOffset+height {
		t.viewOffset = t.cursor - height + 1
	}
	maxOffset := len(t.flattened) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if t.viewOffset > maxOffset {
		t.viewOffset = maxOffset
	}
	if t.viewOffset < 0 {
		t.viewOffset = 0
	}
	return t.viewOffset
}

func (t *Tree) CursorNode() *Node {
	if t.cursor >= 0 && t.cursor < len(t.flattened) {
		return t.flattened[t.cursor]
	}
	return nil
}

func (t *Tree) CursorItem() Item {
	if n := t.CursorNode(); n != nil && n.Kind == KindLeaf {
		return n.Item
	}
	return nil
}

func (t *Tree) Nodes() []*Node { return t.flattened }

// SplitTerraform splits a terraform address into hierarchical segments.
// "module.vpc.module.subnets.aws_subnet.private[0]" ->
//
//	["module.vpc", "module.subnets", "aws_subnet.private[0]"]
func SplitTerraform(address string) []string {
	parts := splitRespectingBrackets(address)
	var segments []string
	i := 0
	for i < len(parts) {
		if parts[i] == "module" && i+1 < len(parts) {
			segments = append(segments, "module."+parts[i+1])
			i += 2
		} else {
			segments = append(segments, strings.Join(parts[i:], "."))
			break
		}
	}
	return segments
}

// splitRespectingBrackets splits on '.' but preserves dots inside bracket-quoted keys.
func splitRespectingBrackets(address string) []string {
	var parts []string
	var current strings.Builder
	inBracket := false
	for _, ch := range address {
		switch {
		case ch == '[':
			inBracket = true
			current.WriteRune(ch)
		case ch == ']':
			inBracket = false
			current.WriteRune(ch)
		case ch == '.' && !inBracket:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
