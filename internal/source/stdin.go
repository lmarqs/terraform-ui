package source

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// StdinProvider reads bytes from standard input when URI is "-".
// Thread-safe: concurrent calls return the same cached data.
type StdinProvider struct {
	once sync.Once
	data []byte
	err  error
}

func (p *StdinProvider) Scheme() string { return "stdin" }

func (p *StdinProvider) Read(_ context.Context, _ string) ([]byte, error) {
	p.once.Do(func() {
		p.data, p.err = io.ReadAll(os.Stdin)
		if p.err != nil {
			p.err = fmt.Errorf("reading stdin: %w", p.err)
			return
		}
		if len(p.data) == 0 {
			p.data = nil
			p.err = fmt.Errorf("stdin is empty: no data received (pipe input with | or redirect with <)")
		}
	})
	return p.data, p.err
}
