package sdk

import (
	"testing"
)

func TestHintSet_WhenZeroValue_ShouldReturnEmptySlice(t *testing.T) {
	hints := HintSet(0).Hints()
	if len(hints) != 0 {
		t.Fatalf("expected empty slice, got %d hints", len(hints))
	}
}

func TestHintSet_WhenSingleHint_ShouldReturnCorrectKeyHint(t *testing.T) {
	tests := []struct {
		name     string
		set      HintSet
		wantKey  string
		wantDesc string
	}{
		{"ShouldShowInspect", HintSetInspect, "Enter", "inspect"},
		{"ShouldShowSelect", HintSetSelect, "Enter", "select"},
		{"ShouldShowConfirm", HintSetConfirm, "Enter", "confirm"},
		{"ShouldShowFilter", HintSetFilter, "/", "filter"},
		{"ShouldShowPin", HintSetPin, "Space", "pin"},
		{"ShouldShowTreeDefault", HintSetTree, "^t", "tree"},
		{"ShouldShowCollapse", HintSetCollapse, "[", "collapse"},
		{"ShouldShowExpand", HintSetExpand, "]", "expand"},
		{"ShouldShowWrapDefault", HintSetWrap, "^w", "wrap(off)"},
		{"ShouldShowRefresh", HintSetRefresh, "^r", "refresh"},
		{"ShouldShowRetry", HintSetRetry, "^r", "retry"},
		{"ShouldShowClearPins", HintSetClearPins, "^u", "unpin all"},
		{"ShouldShowCancel", HintSetCancel, "Esc", "cancel"},
		{"ShouldShowQuit", HintSetQuit, "q", "quit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := tt.set.Hints()
			if len(hints) != 1 {
				t.Fatalf("expected 1 hint, got %d", len(hints))
			}
			if hints[0].Key != tt.wantKey {
				t.Errorf("expected key %q, got %q", tt.wantKey, hints[0].Key)
			}
			if hints[0].Description != tt.wantDesc {
				t.Errorf("expected description %q, got %q", tt.wantDesc, hints[0].Description)
			}
		})
	}
}

func TestHintSet_WhenCombined_ShouldProduceFixedOrder(t *testing.T) {
	tests := []struct {
		name      string
		set       HintSet
		wantDescs []string
	}{
		{
			"ShouldOrderInspectBeforeBack",
			HintSetQuit | HintSetInspect,
			[]string{"inspect", "quit"},
		},
		{
			"ShouldOrderRegardlessOfBitCombinationOrder",
			HintSetQuit | HintSetFilter | HintSetInspect | HintSetPin,
			[]string{"inspect", "filter", "pin", "quit"},
		},
		{
			"ShouldMatchStateListPattern",
			HintSetInspect | HintSetPin | HintSetFilter | HintSetTree | HintSetQuit,
			[]string{"inspect", "filter", "pin", "tree", "quit"},
		},
		{
			"ShouldMatchStateDetailPattern",
			HintSetCancel | HintSetWrap | HintSetPin,
			[]string{"pin", "wrap(off)", "cancel"},
		},
		{
			"ShouldMatchPlanDonePattern",
			HintSetInspect | HintSetPin | HintSetRefresh | HintSetQuit,
			[]string{"inspect", "pin", "refresh", "quit"},
		},
		{
			"ShouldMatchPlanErrorPattern",
			HintSetRetry | HintSetQuit,
			[]string{"retry", "quit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := tt.set.Hints()
			if len(hints) != len(tt.wantDescs) {
				t.Fatalf("expected %d hints, got %d: %v", len(tt.wantDescs), len(hints), hintsDescs(hints))
			}
			for i, wantDesc := range tt.wantDescs {
				if hints[i].Description != wantDesc {
					t.Errorf("position %d: expected %q, got %q (full order: %v)",
						i, wantDesc, hints[i].Description, hintsDescs(hints))
				}
			}
		})
	}
}

func TestHintSet_WhenDynamicHintsWithOpts_ShouldReflectState(t *testing.T) {
	tests := []struct {
		name     string
		set      HintSet
		opts     HintSetOpts
		wantDesc string
	}{
		{"ShouldShowFlatWhenTreeModeTrue", HintSetTree, HintSetOpts{TreeMode: true}, "flat"},
		{"ShouldShowTreeWhenTreeModeFalse", HintSetTree, HintSetOpts{TreeMode: false}, "tree"},
		{"ShouldShowWrapOnWhenWrapModeTrue", HintSetWrap, HintSetOpts{WrapMode: true}, "wrap(on)"},
		{"ShouldShowWrapOffWhenWrapModeFalse", HintSetWrap, HintSetOpts{WrapMode: false}, "wrap(off)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := tt.set.Hints(tt.opts)
			if len(hints) != 1 {
				t.Fatalf("expected 1 hint, got %d", len(hints))
			}
			if hints[0].Description != tt.wantDesc {
				t.Errorf("expected description %q, got %q", tt.wantDesc, hints[0].Description)
			}
		})
	}
}

func TestHintSet_WhenPinnedOptSet_ShouldAppendPinnedIndicator(t *testing.T) {
	set := HintSetInspect | HintSetQuit
	hints := set.Hints(HintSetOpts{Pinned: true})

	if len(hints) < 3 {
		t.Fatalf("expected at least 3 hints (inspect + back + pinned), got %d", len(hints))
	}
	last := hints[len(hints)-1]
	if last.Description != "[pinned]" {
		t.Errorf("expected last hint to be [pinned], got %q", last.Description)
	}
	if last.Key != "" {
		t.Errorf("expected pinned indicator to have empty key, got %q", last.Key)
	}
}

func TestHintSet_WhenPinnedOptFalse_ShouldNotAppendPinnedIndicator(t *testing.T) {
	set := HintSetInspect | HintSetQuit
	hints := set.Hints(HintSetOpts{Pinned: false})

	for _, h := range hints {
		if h.Description == "[pinned]" {
			t.Fatal("should not include [pinned] indicator when Pinned is false")
		}
	}
}

func TestHintSet_WhenNoOptsProvided_ShouldUseDefaults(t *testing.T) {
	set := HintSetTree | HintSetWrap

	hints := set.Hints()
	if len(hints) != 2 {
		t.Fatalf("expected 2 hints, got %d", len(hints))
	}

	descs := hintsDescs(hints)
	found := map[string]bool{}
	for _, d := range descs {
		found[d] = true
	}
	if !found["tree"] {
		t.Error("expected Tree default to be 'tree'")
	}
	if !found["wrap(off)"] {
		t.Error("expected Wrap default to be 'wrap(off)'")
	}
}

func TestHintSet_Has_WhenSubsetPresent_ShouldReturnTrue(t *testing.T) {
	tests := []struct {
		name   string
		h      HintSet
		other  HintSet
		expect bool
	}{
		{"ShouldReturnTrueForSingleBitPresent", HintSetInspect | HintSetQuit, HintSetInspect, true},
		{"ShouldReturnTrueForMultipleBitsPresent", HintSetInspect | HintSetQuit | HintSetFilter, HintSetInspect | HintSetQuit, true},
		{"ShouldReturnTrueForSameSet", HintSetInspect | HintSetQuit, HintSetInspect | HintSetQuit, true},
		{"ShouldReturnTrueForZeroSubset", HintSetInspect, HintSet(0), true},
		{"ShouldReturnFalseForMissingBit", HintSetInspect | HintSetQuit, HintSetFilter, false},
		{"ShouldReturnFalseForPartialSubset", HintSetInspect | HintSetQuit, HintSetInspect | HintSetFilter, false},
		{"ShouldReturnFalseForZeroHasNonZero", HintSet(0), HintSetInspect, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.h.Has(tt.other)
			if got != tt.expect {
				t.Errorf("HintSet(%d).Has(%d) = %v, want %v", tt.h, tt.other, got, tt.expect)
			}
		})
	}
}

func TestHintSet_WhenAllBitsSet_ShouldProduceAllHints(t *testing.T) {
	all := HintSetInspect |
		HintSetSelect | HintSetConfirm | HintSetFilter | HintSetPin |
		HintSetTree | HintSetCollapse | HintSetExpand | HintSetWrap |
		HintSetPinnedFilter | HintSetRefresh |
		HintSetRetry | HintSetClearPins |
		HintSetCancel | HintSetQuit

	hints := all.Hints()
	if len(hints) != 15 {
		t.Fatalf("expected 15 hints, got %d", len(hints))
	}
}

func TestHintSet_WhenDynamicOptsWithCombination_ShouldApplyToCorrectHints(t *testing.T) {
	set := HintSetInspect | HintSetTree | HintSetWrap | HintSetQuit
	opts := HintSetOpts{TreeMode: true, WrapMode: true}

	hints := set.Hints(opts)
	if len(hints) != 4 {
		t.Fatalf("expected 4 hints, got %d: %v", len(hints), hintsDescs(hints))
	}

	descs := hintsDescs(hints)
	found := map[string]bool{}
	for _, d := range descs {
		found[d] = true
	}
	if !found["flat"] {
		t.Error("expected 'flat' hint (TreeMode=true means show 'flat' to switch back)")
	}
	if !found["wrap(on)"] {
		t.Error("expected 'wrap(on)' hint (WrapMode=true)")
	}
	if !found["inspect"] {
		t.Error("expected 'inspect' hint")
	}
	if !found["quit"] {
		t.Error("expected .quit. hint")
	}
}

func TestHintSet_WhenPinnedWithDynamicOpts_ShouldAppendAfterAll(t *testing.T) {
	set := HintSetInspect | HintSetTree | HintSetQuit
	opts := HintSetOpts{TreeMode: true, Pinned: true}

	hints := set.Hints(opts)
	if len(hints) != 4 {
		t.Fatalf("expected 4 hints (inspect + tree + back + pinned), got %d: %v", len(hints), hintsDescs(hints))
	}

	last := hints[len(hints)-1]
	if last.Description != "[pinned]" {
		t.Errorf("expected last hint to be [pinned], got %q", last.Description)
	}
}

func TestHintSet_WhenPinnedFilterDynamic_ShouldReflectState(t *testing.T) {
	tests := []struct {
		name     string
		opts     HintSetOpts
		wantDesc string
	}{
		{"ShouldShowPinnedOnWhenTrue", HintSetOpts{PinnedFilter: true}, "pinned(on)"},
		{"ShouldShowPinnedOffWhenFalse", HintSetOpts{PinnedFilter: false}, "pinned(off)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hints := HintSetPinnedFilter.Hints(tt.opts)
			if len(hints) != 1 {
				t.Fatalf("expected 1 hint, got %d", len(hints))
			}
			if hints[0].Key != "^p" {
				t.Errorf("Key = %q, want %q", hints[0].Key, "^p")
			}
			if hints[0].Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", hints[0].Description, tt.wantDesc)
			}
		})
	}
}

func TestResolveDynamic_WhenUnknownBit_ShouldReturnEmptyHint(t *testing.T) {
	hint := resolveDynamic(HintSet(0), HintSetOpts{})
	if hint.Key != "" {
		t.Errorf("expected empty key, got %q", hint.Key)
	}
	if hint.Description != "" {
		t.Errorf("expected empty description, got %q", hint.Description)
	}
}

func hintsDescs(hints []KeyHint) []string {
	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Description
	}
	return descs
}
