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
		{"ShouldShowPin", HintSetPin, "Space", "pin"},
		{"ShouldShowFilter", HintSetFilter, "/", "filter"},
		{"ShouldShowTreeDefault", HintSetTree, "^t", "flat"},
		{"ShouldShowCollapseExpand", HintSetCollapse, "[/]", "collapse/expand"},
		{"ShouldShowWrapDefault", HintSetWrap, "^w", "wrap(off)"},
		{"ShouldShowRefresh", HintSetRefresh, "r", "refresh"},
		{"ShouldShowRetry", HintSetRetry, "r", "retry"},
		{"ShouldShowDelete", HintSetDelete, "d", "delete"},
		{"ShouldShowEdit", HintSetEdit, "e", "edit"},
		{"ShouldShowApply", HintSetApply, "a", "apply"},
		{"ShouldShowNew", HintSetNew, "n", "new"},
		{"ShouldShowUnlock", HintSetUnlock, "u", "force-unlock"},
		{"ShouldShowCancel", HintSetCancel, "Esc", "cancel"},
		{"ShouldShowBack", HintSetBack, "q", "back"},
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
			HintSetBack | HintSetInspect,
			[]string{"inspect", "back"},
		},
		{
			"ShouldOrderRegardlessOfBitCombinationOrder",
			HintSetBack | HintSetFilter | HintSetInspect | HintSetPin,
			[]string{"inspect", "pin", "filter", "back"},
		},
		{
			"ShouldMatchStateListPattern",
			HintSetInspect | HintSetPin | HintSetFilter | HintSetTree | HintSetBack,
			[]string{"inspect", "pin", "filter", "flat", "back"},
		},
		{
			"ShouldMatchStateDetailPattern",
			HintSetCancel | HintSetWrap | HintSetPin | HintSetDelete | HintSetEdit,
			[]string{"pin", "wrap(off)", "delete", "edit", "cancel"},
		},
		{
			"ShouldMatchPlanDonePattern",
			HintSetInspect | HintSetPin | HintSetApply | HintSetRefresh | HintSetBack,
			[]string{"inspect", "pin", "refresh", "apply", "back"},
		},
		{
			"ShouldMatchPlanErrorPattern",
			HintSetRetry | HintSetBack,
			[]string{"retry", "back"},
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
		{"ShouldShowTreeWhenTreeModeTrue", HintSetTree, HintSetOpts{TreeMode: true}, "tree"},
		{"ShouldShowFlatWhenTreeModeFalse", HintSetTree, HintSetOpts{TreeMode: false}, "flat"},
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
	set := HintSetInspect | HintSetBack
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
	set := HintSetInspect | HintSetBack
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
	if !found["flat"] {
		t.Error("expected Tree default to be 'flat'")
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
		{"ShouldReturnTrueForSingleBitPresent", HintSetInspect | HintSetBack, HintSetInspect, true},
		{"ShouldReturnTrueForMultipleBitsPresent", HintSetInspect | HintSetBack | HintSetFilter, HintSetInspect | HintSetBack, true},
		{"ShouldReturnTrueForSameSet", HintSetInspect | HintSetBack, HintSetInspect | HintSetBack, true},
		{"ShouldReturnTrueForZeroSubset", HintSetInspect, HintSet(0), true},
		{"ShouldReturnFalseForMissingBit", HintSetInspect | HintSetBack, HintSetFilter, false},
		{"ShouldReturnFalseForPartialSubset", HintSetInspect | HintSetBack, HintSetInspect | HintSetFilter, false},
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
		HintSetSelect | HintSetConfirm | HintSetPin | HintSetFilter |
		HintSetTree | HintSetCollapse | HintSetWrap | HintSetRefresh |
		HintSetRetry | HintSetDelete | HintSetEdit | HintSetApply |
		HintSetNew | HintSetUnlock | HintSetCancel | HintSetBack

	hints := all.Hints()
	if len(hints) != 17 {
		t.Fatalf("expected 17 hints, got %d", len(hints))
	}
}

func TestHintSet_WhenDynamicOptsWithCombination_ShouldApplyToCorrectHints(t *testing.T) {
	set := HintSetInspect | HintSetTree | HintSetWrap | HintSetBack
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
	if !found["tree"] {
		t.Error("expected 'tree' hint (TreeMode=true)")
	}
	if !found["wrap(on)"] {
		t.Error("expected 'wrap(on)' hint (WrapMode=true)")
	}
	if !found["inspect"] {
		t.Error("expected 'inspect' hint")
	}
	if !found["back"] {
		t.Error("expected 'back' hint")
	}
}

func TestHintSet_WhenPinnedWithDynamicOpts_ShouldAppendAfterAll(t *testing.T) {
	set := HintSetInspect | HintSetTree | HintSetBack
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

func hintsDescs(hints []KeyHint) []string {
	descs := make([]string, len(hints))
	for i, h := range hints {
		descs[i] = h.Description
	}
	return descs
}
