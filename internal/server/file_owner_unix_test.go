//go:build unix

package server

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestAtomicWritePreservesFileOwner(t *testing.T) {
	groups, err := os.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	group := -1
	for _, candidate := range groups {
		if candidate != os.Getegid() {
			group = candidate
			break
		}
	}
	if group == -1 {
		t.Skip("a supplementary group is required to exercise ownership preservation")
	}
	name := filepath.Join(t.TempDir(), "configuration.txt")
	if err := os.WriteFile(name, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chown(name, -1, group); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := atomicWriteFile(name, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	want, got := before.Sys().(*syscall.Stat_t), after.Sys().(*syscall.Stat_t)
	if got.Uid != want.Uid || got.Gid != want.Gid {
		t.Fatalf("saved file owner changed: uid:gid=%d:%d, want %d:%d", got.Uid, got.Gid, want.Uid, want.Gid)
	}
}
