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

func TestPins_Toggle_AddWhenAbsent(t *testing.T) {
	original := Pins{"a"}
	got := original.Toggle("b")
	if !reflect.DeepEqual(got, Pins{"a", "b"}) {
		t.Errorf("Toggle add: got %v, want [a b]", got)
	}
	if !reflect.DeepEqual(original, Pins{"a"}) {
		t.Errorf("receiver mutated: %v", original)
	}
}

func TestPins_Toggle_RemoveWhenPresent(t *testing.T) {
	original := Pins{"a", "b", "c"}
	got := original.Toggle("b")
	if !reflect.DeepEqual(got, Pins{"a", "c"}) {
		t.Errorf("Toggle remove: got %v, want [a c]", got)
	}
	if !reflect.DeepEqual(original, Pins{"a", "b", "c"}) {
		t.Errorf("receiver mutated: %v", original)
	}
}

func TestPins_Toggle_OnEmpty(t *testing.T) {
	var original Pins
	got := original.Toggle("x")
	if !reflect.DeepEqual(got, Pins{"x"}) {
		t.Errorf("Toggle on empty: got %v, want [x]", got)
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
