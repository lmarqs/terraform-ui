package ui

// ExpandSet tracks which indices in a list are expanded (showing detail).
type ExpandSet struct {
	expanded map[int]bool
}

// NewExpandSet creates an empty expand set.
func NewExpandSet() *ExpandSet {
	return &ExpandSet{expanded: make(map[int]bool)}
}

// Toggle flips the expanded state of the given index.
func (e *ExpandSet) Toggle(idx int) {
	if e.expanded[idx] {
		delete(e.expanded, idx)
	} else {
		e.expanded[idx] = true
	}
}

// IsExpanded returns whether the given index is currently expanded.
func (e *ExpandSet) IsExpanded(idx int) bool {
	return e.expanded[idx]
}

// CollapseAll resets all items to collapsed.
func (e *ExpandSet) CollapseAll() {
	e.expanded = make(map[int]bool)
}
