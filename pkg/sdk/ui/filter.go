package ui

import (
	"sort"

	"github.com/junegunn/fzf/src/algo"
	"github.com/junegunn/fzf/src/util"
)

// FuzzyFilter provides fzf-based filtering over a typed collection.
type FuzzyFilter[T any] struct {
	items   []T
	getText func(T) string
	query   string

	// cached results
	scored []scoredItem[T]
}

type scoredItem[T any] struct {
	item  T
	index int
	score int
}

// NewFuzzyFilter creates a filter with the given text accessor function.
func NewFuzzyFilter[T any](getText func(T) string) *FuzzyFilter[T] {
	return &FuzzyFilter[T]{getText: getText}
}

// SetItems replaces the source collection and re-filters.
func (f *FuzzyFilter[T]) SetItems(items []T) {
	f.items = items
	f.filter()
}

// SetQuery updates the search query and re-filters.
func (f *FuzzyFilter[T]) SetQuery(query string) {
	f.query = query
	f.filter()
}

// Query returns the current filter query.
func (f *FuzzyFilter[T]) Query() string {
	return f.query
}

// Results returns matched items sorted by score (best match first).
func (f *FuzzyFilter[T]) Results() []T {
	out := make([]T, len(f.scored))
	sorted := make([]scoredItem[T], len(f.scored))
	copy(sorted, f.scored)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})
	for i, s := range sorted {
		out[i] = s.item
	}
	return out
}

// OriginalOrder returns matched items preserving their original insertion order.
func (f *FuzzyFilter[T]) OriginalOrder() []T {
	out := make([]T, len(f.scored))
	for i, s := range f.scored {
		out[i] = s.item
	}
	return out
}

// IsActive returns true if a non-empty query is set.
func (f *FuzzyFilter[T]) IsActive() bool {
	return f.query != ""
}

// Clear resets the query and shows all items.
func (f *FuzzyFilter[T]) Clear() {
	f.query = ""
	f.filter()
}

// ScoreAt returns the match score of the idx-th item in OriginalOrder results.
func (f *FuzzyFilter[T]) ScoreAt(idx int) int {
	if idx < 0 || idx >= len(f.scored) {
		return 0
	}
	return f.scored[idx].score
}

// Count returns the number of matched items.
func (f *FuzzyFilter[T]) Count() int {
	return len(f.scored)
}

func (f *FuzzyFilter[T]) filter() {
	if f.query == "" {
		f.scored = make([]scoredItem[T], len(f.items))
		for i, item := range f.items {
			f.scored[i] = scoredItem[T]{item: item, index: i, score: 0}
		}
		return
	}

	pattern := []rune(f.query)
	slab := util.MakeSlab(100*1024, 2048)
	f.scored = nil

	for i, item := range f.items {
		input := util.RunesToChars([]rune(f.getText(item)))
		res, _ := algo.FuzzyMatchV2(false, true, true, &input, pattern, false, slab)
		if res.Score > 0 {
			f.scored = append(f.scored, scoredItem[T]{item: item, index: i, score: res.Score})
		}
	}
}
