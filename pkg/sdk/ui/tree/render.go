package tree

import (
	"fmt"
	"strings"
)

// RenderOpts controls visual rendering.
type RenderOpts struct {
	// Width is the available content width.
	Width int
	// Height is the maximum number of visible rows.
	Height int
	// RenderLeaf formats a leaf node line. If nil, uses Label.
	RenderLeaf func(node *Node, pinned bool) string
	// RenderBranch formats a branch node line. If nil, uses "Label (count)".
	RenderBranch func(node *Node, pinned bool) string
	// PinIndicator is the string shown for pinned items. Default: "* ".
	// Deprecated: use PinIndicators for checkbox-style rendering.
	PinIndicator string
	// PinIndicators provides state-specific pin indicators (none/partial/full).
	// If set, overrides PinIndicator.
	PinIndicators *PinIndicators
	// SelectedStyle wraps the selected row. If nil, no highlight.
	SelectedStyle func(s string, width int) string
}

// PinIndicators defines the visual indicators for each pin state.
type PinIndicators struct {
	None    string // shown when not pinned (e.g. "[ ] ")
	Full    string // shown when fully pinned (e.g. "[*] ")
	Partial string // shown when partially pinned — branches only (e.g. "[-] ")
}

// Render returns the visible lines of the tree within the viewport.
func (t *Tree) Render(opts RenderOpts) string {
	if len(t.flattened) == 0 {
		return ""
	}

	height := opts.Height
	if height <= 0 {
		height = len(t.flattened)
	}

	startIdx := 0
	if t.cursor >= height {
		startIdx = t.cursor - height + 1
	}
	endIdx := startIdx + height
	if endIdx > len(t.flattened) {
		endIdx = len(t.flattened)
	}

	// Build ancestor connector state for tree lines
	var lines []string
	for i := startIdx; i < endIdx; i++ {
		node := t.flattened[i]
		line := t.renderNode(node, opts)
		if i == t.cursor && opts.SelectedStyle != nil {
			line = opts.SelectedStyle(line, opts.Width)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (t *Tree) renderNode(node *Node, opts RenderOpts) string {
	prefix := t.buildConnectors(node)

	state := t.NodePinState(node.Path)
	pinInd := t.pinIndicatorFor(state, node.Kind, opts)
	pinned := state == PinFull

	var content string
	if node.Kind == KindBranch {
		if opts.RenderBranch != nil {
			content = opts.RenderBranch(node, pinned)
		} else {
			indicator := "▶"
			if node.Expanded {
				indicator = "▼"
			}
			content = fmt.Sprintf("%s %s (%d)", indicator, node.Label, node.Count)
		}
	} else {
		if opts.RenderLeaf != nil {
			content = opts.RenderLeaf(node, pinned)
		} else {
			content = node.Label
		}
	}

	return pinInd + prefix + content
}

func (t *Tree) pinIndicatorFor(state PinState, kind NodeKind, opts RenderOpts) string {
	if opts.PinIndicators != nil {
		switch state {
		case PinFull:
			return opts.PinIndicators.Full
		case PinPartial:
			if kind == KindBranch {
				return opts.PinIndicators.Partial
			}
			return opts.PinIndicators.None
		default:
			return opts.PinIndicators.None
		}
	}
	// Legacy single-indicator mode
	pinIndicator := opts.PinIndicator
	if pinIndicator == "" {
		pinIndicator = "* "
	}
	if state == PinFull {
		return pinIndicator
	}
	return "  "
}

func (t *Tree) buildConnectors(node *Node) string {
	if node.Depth == 0 {
		return ""
	}

	var connector string
	if node.IsLast {
		connector = "└─"
	} else {
		connector = "├─"
	}

	ancestors := t.getAncestorContinuations(node)
	var prefix strings.Builder
	for i := 0; i < node.Depth-1; i++ {
		if i < len(ancestors) && ancestors[i] {
			prefix.WriteString("│ ")
		} else {
			prefix.WriteString("  ")
		}
	}

	return prefix.String() + connector
}

// getAncestorContinuations determines which ancestor levels have continuing siblings.
func (t *Tree) getAncestorContinuations(node *Node) []bool {
	result := make([]bool, node.Depth)
	idx := t.indexOf(node)
	// For each ancestor depth level, walk backwards to find the ancestor node
	// and check if it's NOT the last child (meaning the vertical line continues).
	for d := 0; d < node.Depth; d++ {
		for i := idx - 1; i >= 0; i-- {
			n := t.flattened[i]
			if n.Depth == d {
				result[d] = !n.IsLast
				break
			}
			if n.Depth < d {
				break
			}
		}
	}
	return result
}

func (t *Tree) indexOf(node *Node) int {
	for i, n := range t.flattened {
		if n == node {
			return i
		}
	}
	return -1
}
