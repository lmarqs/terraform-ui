package sdk

import (
	"reflect"
	"testing"
)

func TestPins_Count(t *testing.T) {
	tests := []struct {
		name string
		pins Pins
		want int
	}{
		{"nil", nil, 0},
		{"empty", Pins{}, 0},
		{"populated", Pins{"a", "b"}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pins.Count(); got != tt.want {
				t.Errorf("Count() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPins_HasAny(t *testing.T) {
	tests := []struct {
		name string
		pins Pins
		want bool
	}{
		{"nil", nil, false},
		{"empty", Pins{}, false},
		{"populated", Pins{"a"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pins.HasAny(); got != tt.want {
				t.Errorf("HasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPins_Contains(t *testing.T) {
	tests := []struct {
		name    string
		pins    Pins
		address string
		want    bool
	}{
		{"nil receiver", nil, "a", false},
		{"empty", Pins{}, "a", false},
		{"present", Pins{"a", "b"}, "b", true},
		{"absent", Pins{"a", "b"}, "c", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pins.Contains(tt.address); got != tt.want {
				t.Errorf("Contains(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}

func TestPins_Toggle_WhenGivenGroups(t *testing.T) {
	tests := []struct {
		name      string
		pins      Pins
		addresses []string
		want      Pins
	}{
		{
			name:      "single absent address is added",
			pins:      Pins{"a"},
			addresses: []string{"b"},
			want:      Pins{"a", "b"},
		},
		{
			name:      "single present address is removed",
			pins:      Pins{"a", "b", "c"},
			addresses: []string{"b"},
			want:      Pins{"a", "c"},
		},
		{
			name:      "address added to an empty set",
			pins:      nil,
			addresses: []string{"x"},
			want:      Pins{"x"},
		},
		{
			name:      "wholly unpinned group is added in order",
			pins:      Pins{"a"},
			addresses: []string{"b", "c"},
			want:      Pins{"a", "b", "c"},
		},
		{
			name:      "wholly pinned group is removed",
			pins:      Pins{"a", "b", "c"},
			addresses: []string{"b", "c"},
			want:      Pins{"a"},
		},
		{
			name:      "partly pinned group adds only what is missing",
			pins:      Pins{"b"},
			addresses: []string{"a", "b", "c"},
			want:      Pins{"b", "a", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.pins.Clone()
			got := tt.pins.Toggle(tt.addresses...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Toggle(%v) = %v, want %v", tt.addresses, got, tt.want)
			}
			if !reflect.DeepEqual(tt.pins, before) {
				t.Errorf("receiver mutated: %v, want %v", tt.pins, before)
			}
		})
	}
}

func TestPins_Clone_NilStaysNil(t *testing.T) {
	var original Pins
	got := original.Clone()
	if got != nil {
		t.Errorf("Clone(nil) = %v, want nil", got)
	}
}

func TestPins_Clone_IsIndependent(t *testing.T) {
	original := Pins{"a", "b"}
	clone := original.Clone()
	if !reflect.DeepEqual(clone, original) {
		t.Fatalf("Clone() = %v, want %v", clone, original)
	}
	clone[0] = "X"
	if original[0] != "a" {
		t.Errorf("mutating clone affected original: %v", original)
	}
}
