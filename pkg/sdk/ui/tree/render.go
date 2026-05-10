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
	PinIndicator string
	// SelectedStyle wraps the selected row. If nil, no highlight.
	SelectedStyle func(s string, width int) string
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

	pinInd := "  "
	pinIndicator := opts.PinIndicator
	if pinIndicator == "" {
		pinIndicator = "* "
	}
	if t.pinned[node.Path] {
		pinInd = pinIndicator
	}

	var content string
	if node.Kind == KindBranch {
		if opts.RenderBranch != nil {
			content = opts.RenderBranch(node, t.pinned[node.Path])
		} else {
			indicator := "▶"
			if node.Expanded {
				indicator = "▼"
			}
			content = fmt.Sprintf("%s %s (%d)", indicator, node.Label, node.Count)
		}
	} else {
		if opts.RenderLeaf != nil {
			content = opts.RenderLeaf(node, t.pinned[node.Path])
		} else {
			content = node.Label
		}
	}

	return pinInd + prefix + content
}

func (t *Tree) buildConnectors(node *Node) string {
	if node.Depth == 0 {
		return ""
	}

	// Build connector for current level
	var connector string
	if node.IsLast {
		connector = "└── "
	} else {
		connector = "├── "
	}

	// Build ancestor connectors by walking up the flattened list
	ancestors := t.getAncestorContinuations(node)
	var prefix strings.Builder
	for i := 0; i < node.Depth-1; i++ {
		if i < len(ancestors) && ancestors[i] {
			prefix.WriteString("│   ")
		} else {
			prefix.WriteString("    ")
		}
	}

	return prefix.String() + connector
}

// getAncestorContinuations determines which ancestor levels have continuing siblings.
func (t *Tree) getAncestorContinuations(node *Node) []bool {
	result := make([]bool, node.Depth)
	// Walk backwards through flattened to find ancestors
	for i := t.indexOf(node) - 1; i >= 0; i-- {
		ancestor := t.flattened[i]
		if ancestor.Depth < node.Depth-1 {
			break
		}
		if ancestor.Depth < node.Depth && !ancestor.IsLast {
			if ancestor.Depth < len(result) {
				result[ancestor.Depth] = true
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
