package utils

import "strings"

// Keep command output bounded while continuing to drain the child's pipes.
type commandOutput struct {
	data      []byte
	truncated bool
}

func (b *commandOutput) Write(p []byte) (int, error) {
	remaining := 4000 - len(b.data)
	if len(p) > remaining {
		b.truncated = true
	}
	b.data = append(b.data, p[:min(len(p), remaining)]...)
	return len(p), nil
}

func (b *commandOutput) String() string {
	result := strings.ToValidUTF8(string(b.data), "")
	if b.truncated {
		result += "\n...(Output truncated)"
	}
	return result
}
