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
	Kind     NodeKind
	Label    string
	Path     string // full module path for branches, full address for leaves
	Depth    int
	Expanded bool
	Count    int  // descendant leaf count (branches only)
	IsLast   bool // last child of its parent (for connector rendering)
	Item     Item // non-nil for leaves only
}

// Tree is an interactive tree navigator built from flat addressed items.
type Tree struct {
	root          *treeNode
	flattened     []*Node
	cursor        int
	viewOffset    int
	expanded      map[string]bool
	pinned        map[string]bool
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
		expanded: make(map[string]bool),
		pinned:   make(map[string]bool),
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
			Kind:     KindBranch,
			Label:    child.label,
			Path:     child.path,
			Depth:    depth,
			Expanded: expanded,
			Count:    t.countLeaves(child),
			IsLast:   isLast,
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
			Path:   item.Address(),
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
		t.expanded[n.Path] = !t.expanded[n.Path]
		t.flatten()
	}
}

func (t *Tree) ExpandFocused() {
	if n := t.CursorNode(); n != nil && n.Kind == KindBranch {
		t.expanded[n.Path] = true
		t.flatten()
	}
}

func (t *Tree) CollapseFocused() {
	n := t.CursorNode()
	if n == nil {
		return
	}
	if n.Kind == KindBranch && t.expanded[n.Path] {
		t.expanded[n.Path] = false
		t.flatten()
		return
	}
	// On leaf or collapsed branch: collapse parent
	parent := t.parentPath(n.Path)
	if parent != "" {
		t.expanded[parent] = false
		t.flatten()
		// Move cursor to the collapsed parent
		for i, node := range t.flattened {
			if node.Path == parent {
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
			if n.Kind == KindBranch && !t.expanded[n.Path] {
				t.expanded[n.Path] = true
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

func (t *Tree) TogglePin() {
	n := t.CursorNode()
	if n == nil {
		return
	}
	if n.Kind == KindLeaf {
		if t.pinned[n.Path] {
			delete(t.pinned, n.Path)
		} else {
			t.pinned[n.Path] = true
		}
	} else {
		node := t.findNode(t.root, n.Path)
		if node == nil {
			return
		}
		state := t.nodePinState(node)
		pin := state != PinFull
		t.setPinRecursive(node, pin)
	}
	t.flatten()
}

func (t *Tree) findNode(parent *treeNode, path string) *treeNode {
	if parent.path == path {
		return parent
	}
	for _, c := range parent.children {
		if found := t.findNode(c, path); found != nil {
			return found
		}
	}
	return nil
}

func (t *Tree) setPinRecursive(node *treeNode, pin bool) {
	for _, item := range node.items {
		if pin {
			t.pinned[item.Address()] = true
		} else {
			delete(t.pinned, item.Address())
		}
	}
	for _, child := range node.children {
		t.setPinRecursive(child, pin)
	}
}

func (t *Tree) nodePinState(node *treeNode) PinState {
	total := 0
	pinned := 0
	t.countPinState(node, &total, &pinned)
	if total == 0 {
		return PinNone
	}
	if pinned == total {
		return PinFull
	}
	if pinned > 0 {
		return PinPartial
	}
	return PinNone
}

func (t *Tree) countPinState(node *treeNode, total, pinned *int) {
	for _, item := range node.items {
		*total++
		if t.pinned[item.Address()] {
			*pinned++
		}
	}
	for _, child := range node.children {
		t.countPinState(child, total, pinned)
	}
}

// NodePinState returns the pin state for a given path.
func (t *Tree) NodePinState(path string) PinState {
	node := t.findNode(t.root, path)
	if node == nil {
		if t.pinned[path] {
			return PinFull
		}
		return PinNone
	}
	return t.nodePinState(node)
}

func (t *Tree) IsPinned(path string) bool { return t.pinned[path] }

func (t *Tree) SetPinned(paths []string) {
	t.pinned = make(map[string]bool, len(paths))
	for _, p := range paths {
		t.pinned[p] = true
	}
	t.flatten()
}

func (t *Tree) PinnedPaths() []string {
	result := make([]string, 0, len(t.pinned))
	for p := range t.pinned {
		result = append(result, p)
	}
	sort.Strings(result)
	return result
}

// Query methods

func (t *Tree) Cursor() int       { return t.cursor }
func (t *Tree) VisibleCount() int  { return len(t.flattened) }

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
	parts := strings.Split(address, ".")
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
