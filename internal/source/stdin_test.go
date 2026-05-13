package source

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestStdinProvider_WhenCreated_ShouldReturnCorrectScheme(t *testing.T) {
	p := &StdinProvider{}
	if got := p.Scheme(); got != "stdin" {
		t.Errorf("Scheme() = %q, want %q", got, "stdin")
	}
}

func TestStdinProvider_WhenReadConcurrently_ShouldReturnSameData(t *testing.T) {
	const payload = `{"resource_changes": [{"address": "aws_instance.web"}]}`
	const goroutines = 10

	p := &StdinProvider{}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
	})

	go func() {
		w.WriteString(payload)
		w.Close()
	}()

	ctx := context.Background()
	var wg sync.WaitGroup
	results := make([][]byte, goroutines)
	errs := make([]error, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = p.Read(ctx, "-")
		}(i)
	}
	wg.Wait()

	for i := 0; i < goroutines; i++ {
		if errs[i] != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, errs[i])
			continue
		}
		if string(results[i]) != payload {
			t.Errorf("goroutine %d: got %q, want %q", i, results[i], payload)
		}
	}
}

func TestStdinProvider_WhenStdinEmpty_ShouldReturnError(t *testing.T) {
	p := &StdinProvider{}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
	})

	_, err = p.Read(context.Background(), "-")
	if err == nil {
		t.Fatal("expected error for empty stdin")
	}
	if !strings.Contains(err.Error(), "stdin is empty") {
		t.Errorf("error should mention 'stdin is empty', got: %s", err.Error())
	}
}
