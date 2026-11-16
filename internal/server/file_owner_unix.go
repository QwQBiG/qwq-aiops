//go:build unix

package server

import (
	"os"
	"syscall"
)

// An atomic replacement must retain ownership: group-readable configuration
// files can become inaccessible if they inherit the temporary file's group.
func preserveFileOwner(file *os.File, original os.FileInfo) error {
	want := original.Sys().(*syscall.Stat_t)
	current, err := file.Stat()
	if err != nil {
		return err
	}
	got := current.Sys().(*syscall.Stat_t)
	if got.Uid == want.Uid && got.Gid == want.Gid {
		return nil
	}
	return file.Chown(int(want.Uid), int(want.Gid))
}
