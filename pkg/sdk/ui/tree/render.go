package tree

import "fmt"

// RenderOpts controls visual rendering.
type RenderOpts struct {
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
	// Pinned reports whether a resource address is pinned. Supplied per render
	// from the caller's Context, which owns Pins (ADR-0018). If nil, no row is
	// pinned.
	Pinned func(address string) bool
}

// PinIndicators defines the visual indicators for each pin state.
type PinIndicators struct {
	None    string // shown when not pinned (e.g. "[ ] ")
	Full    string // shown when fully pinned (e.g. "[*] ")
	Partial string // shown when partially pinned — branches only (e.g. "[-] ")
}

// RenderRows returns every row of the tree as a slice, without truncation,
// windowing or cursor highlighting. Those concerns are owned by the caller
// (typically a ContentPanel).
func (t *Tree) RenderRows(opts RenderOpts) []string {
	if len(t.flattened) == 0 {
		return nil
	}
	rows := make([]string, 0, len(t.flattened))
	for _, node := range t.flattened {
		rows = append(rows, t.renderNode(node, opts))
	}
	return rows
}

func (t *Tree) renderNode(node *Node, opts RenderOpts) string {
	prefix := t.buildConnectors(node)

	state := t.PinStateOf(node, opts.Pinned)
	pinInd := t.pinIndicatorFor(state, opts)
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

// pinIndicatorFor maps a row's pin state to its indicator. Only a branch row
// can be Partial — PinStateOf derives a leaf from its single address — so no
// leaf case is needed here.
func (t *Tree) pinIndicatorFor(state PinState, opts RenderOpts) string {
	if opts.PinIndicators != nil {
		switch state {
		case PinFull:
			return opts.PinIndicators.Full
		case PinPartial:
			return opts.PinIndicators.Partial
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
	if node.IsLast {
		return node.prefix + "└─"
	}
	return node.prefix + "├─"
}
