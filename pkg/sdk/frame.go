package sdk

import tea "github.com/charmbracelet/bubbletea"

// KeyHint describes a single keybinding shown in the hint bar.
type KeyHint struct {
	Key         string
	Description string
}

// Common hints reusable across plugins.
var (
	HintQuit    = KeyHint{Key: "q", Description: "quit"}
	HintBack    = KeyHint{Key: "Esc", Description: "back"}
	HintCancel  = KeyHint{Key: "Esc", Description: "cancel"}
	HintRefresh = KeyHint{Key: "^r", Description: "refresh"}
	HintRetry   = KeyHint{Key: "^r", Description: "retry"}
	HintFilter  = KeyHint{Key: "/", Description: "filter"}
	HintPin     = KeyHint{Key: "Space", Description: "pin"}
	HintInspect = KeyHint{Key: "Enter", Description: "inspect"}
	HintSelect  = KeyHint{Key: "Enter", Description: "select"}
	HintConfirm = KeyHint{Key: "Enter", Description: "confirm"}
)

// Frame is a composable view layer that lives in a navigation stack.
// Input is always routed to the topmost frame. Each frame renders its
// own view and declares which key hints to show.
type Frame interface {
	// ID returns a short identifier for debugging/logging.
	ID() string

	// Update processes a message. Returns nil to signal pop (back navigation).
	Update(msg tea.Msg) (Frame, tea.Cmd)

	// View renders this frame's content within the given dimensions.
	View(width, height int) string

	// Hints returns the key hints to display while this frame is active.
	Hints() []KeyHint
}

// Stackable is an optional interface plugins implement to use
// frame-based navigation. The app routes key input through the plugin's
// stack instead of calling Update directly.
type Stackable interface {
	Stack() *Stack
}

// FramePushMsg is returned as a tea.Cmd to request pushing a new frame.
type FramePushMsg struct {
	Frame Frame
}

// HintSet is a bitmask of standard hints. Plugins compose with |.
// Hints always render in a consistent fixed order regardless of which bits
// are set — "navigate" is always first if present, "back" is always last, etc.
type HintSet uint32

const (
	HintSetInspect      HintSet = 1 << iota // Enter inspect
	HintSetSelect                           // Enter select
	HintSetConfirm                          // Enter confirm
	HintSetFilter                           // / filter
	HintSetPin                              // Space pin
	HintSetTree                             // ^t flat/tree (dynamic label)
	HintSetCollapse                         // [ collapse
	HintSetExpand                           // ] expand
	HintSetWrap                             // ^w wrap(on/off) (dynamic label)
	HintSetPinnedFilter                     // ^p pinned(on/off) (dynamic label)
	HintSetRefresh                          // ^r refresh
	HintSetRetry                            // ^r retry
	HintSetClearPins                        // ^u unpin all
	HintSetCancel                           // Esc cancel
	HintSetBack                             // Esc back
	HintSetQuit                             // q quit
)

// HintSetOpts provides dynamic state for hints that need it.
type HintSetOpts struct {
	TreeMode     bool // for HintSetTree: true → shows "tree", false → shows "flat"
	WrapMode     bool // for HintSetWrap: true → shows "wrap(on)", false → shows "wrap(off)"
	PinnedFilter bool // for HintSetPinnedFilter: true → shows "pinned(on)", false → shows "pinned(off)"
	Pinned       bool // appends [pinned] indicator at the end
}

// hintDef maps a HintSet bit to its KeyHint representation.
type hintDef struct {
	bit     HintSet
	hint    KeyHint
	dynamic bool // if true, resolved via opts
}

// hintOrder defines the fixed rendering order for all standard UI hints.
// Grouped: navigation → view modes → pin management → escape.
var hintOrder = []hintDef{
	// Navigation
	{bit: HintSetInspect, hint: KeyHint{Key: "Enter", Description: "inspect"}},
	{bit: HintSetSelect, hint: KeyHint{Key: "Enter", Description: "select"}},
	{bit: HintSetConfirm, hint: KeyHint{Key: "Enter", Description: "confirm"}},
	{bit: HintSetFilter, hint: KeyHint{Key: "/", Description: "filter"}},
	{bit: HintSetPin, hint: KeyHint{Key: "Space", Description: "pin"}},
	// View modes
	{bit: HintSetTree, dynamic: true},
	{bit: HintSetCollapse, hint: KeyHint{Key: "[", Description: "collapse"}},
	{bit: HintSetExpand, hint: KeyHint{Key: "]", Description: "expand"}},
	{bit: HintSetWrap, dynamic: true},
	{bit: HintSetPinnedFilter, dynamic: true},
	{bit: HintSetRefresh, hint: KeyHint{Key: "^r", Description: "refresh"}},
	{bit: HintSetRetry, hint: KeyHint{Key: "^r", Description: "retry"}},
	// Pin management
	{bit: HintSetClearPins, hint: KeyHint{Key: "^u", Description: "unpin all"}},
	// Escape
	{bit: HintSetCancel, hint: KeyHint{Key: "Esc", Description: "cancel"}},
	{bit: HintSetBack, hint: KeyHint{Key: "Esc", Description: "back"}},
	{bit: HintSetQuit, hint: KeyHint{Key: "q", Description: "quit"}},
}

// Hints converts a HintSet to a slice of KeyHint in fixed display order.
// The order is always the same regardless of which bits are set.
// Pass HintSetOpts to control dynamic hint labels (tree mode, wrap mode, pinned).
func (h HintSet) Hints(opts ...HintSetOpts) []KeyHint {
	var o HintSetOpts
	if len(opts) > 0 {
		o = opts[0]
	}

	var result []KeyHint
	for _, def := range hintOrder {
		if h&def.bit == 0 {
			continue
		}
		if def.dynamic {
			result = append(result, resolveDynamic(def.bit, o))
		} else {
			result = append(result, def.hint)
		}
	}

	if o.Pinned {
		result = append(result, KeyHint{Description: "[pinned]"})
	}

	return result
}

// Has returns true if all bits in other are set in h.
func (h HintSet) Has(other HintSet) bool {
	return h&other == other
}

// resolveDynamic returns the KeyHint for a dynamic hint bit based on opts.
func resolveDynamic(bit HintSet, opts HintSetOpts) KeyHint {
	switch bit {
	case HintSetTree:
		if opts.TreeMode {
			return KeyHint{Key: "^t", Description: "flat"}
		}
		return KeyHint{Key: "^t", Description: "tree"}
	case HintSetWrap:
		if opts.WrapMode {
			return KeyHint{Key: "^w", Description: "wrap(on)"}
		}
		return KeyHint{Key: "^w", Description: "wrap(off)"}
	case HintSetPinnedFilter:
		if opts.PinnedFilter {
			return KeyHint{Key: "^p", Description: "pinned(on)"}
		}
		return KeyHint{Key: "^p", Description: "pinned(off)"}
	default:
		return KeyHint{}
	}
}
