//go:build !unix

package server

import "os"

func preserveFileOwner(file *os.File, original os.FileInfo) error {
	return nil
}
