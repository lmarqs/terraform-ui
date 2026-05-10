package sdk

import "testing"

type mockCountable struct {
	filtered int
	total    int
}

func (m *mockCountable) Count() (int, int) {
	return m.filtered, m.total
}

func TestCountable_WhenImplemented_ShouldReturnCounts(t *testing.T) {
	tests := []struct {
		name       string
		filtered   int
		total      int
		wantFilter int
		wantTotal  int
	}{
		{"AllShown", 30, 1549, 30, 1549},
		{"NoneFiltered", 1549, 1549, 1549, 1549},
		{"Empty", 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Countable = &mockCountable{filtered: tt.filtered, total: tt.total}
			f, total := c.Count()
			if f != tt.wantFilter {
				t.Errorf("filtered = %d, want %d", f, tt.wantFilter)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestCountable_WhenNotImplemented_ShouldNotSatisfyInterface(t *testing.T) {
	type nonCountable struct{}
	var i interface{} = &nonCountable{}
	if _, ok := i.(Countable); ok {
		t.Error("nonCountable should not satisfy Countable interface")
	}
}
