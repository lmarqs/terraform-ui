package source

import (
	"context"
	"fmt"
	"io"
	"os"
)

// StdinProvider reads bytes from standard input when URI is "-".
type StdinProvider struct {
	consumed bool
}

func (p *StdinProvider) Scheme() string { return "stdin" }

func (p *StdinProvider) Read(_ context.Context, _ string) ([]byte, error) {
	if p.consumed {
		return nil, fmt.Errorf("stdin already consumed: only one flag can read from stdin per invocation")
	}
	p.consumed = true

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("stdin is empty: no data received (pipe input with | or redirect with <)")
	}
	return data, nil
}
