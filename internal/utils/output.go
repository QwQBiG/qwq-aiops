package utils

import (
	"strings"
	"sync"
)

// Keep command output bounded while continuing to drain the child's pipes.
type CommandOutput struct {
	mu        sync.Mutex
	data      []byte
	truncated bool
}

func (b *CommandOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := 4000 - len(b.data)
	if len(p) > remaining {
		b.truncated = true
	}
	b.data = append(b.data, p[:min(len(p), remaining)]...)
	return len(p), nil
}

func (b *CommandOutput) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := strings.ToValidUTF8(string(b.data), "")
	if b.truncated {
		result += "\n...(Output truncated)"
	}
	return result
}
