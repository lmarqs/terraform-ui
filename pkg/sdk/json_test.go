package sdk

import (
	"strings"
	"testing"
)

func TestMarshalJSON_ShouldReturnIndentedJSONWithNewline(t *testing.T) {
	v := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{Name: "test", Age: 42}

	data := MarshalJSON(v)
	s := string(data)

	if !strings.Contains(s, `"name": "test"`) {
		t.Errorf("missing name field, got: %s", s)
	}
	if !strings.Contains(s, `"age": 42`) {
		t.Errorf("missing age field, got: %s", s)
	}
	if s[len(s)-1] != '\n' {
		t.Error("should end with newline")
	}
}

func TestMarshalJSON_WhenUnmarshalable_ShouldPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MarshalJSON should panic on unmarshalable value")
		}
	}()
	MarshalJSON(make(chan int))
}
