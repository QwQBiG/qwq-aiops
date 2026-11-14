package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveSafePath maps a slash-separated API path into MountPoint. Both the
// requested path and its existing symlink targets must stay inside that root.
func resolveSafePath(userPath string) (string, error) {
	if strings.ContainsAny(userPath, "\\\x00") || filepath.VolumeName(userPath) != "" {
		return "", fmt.Errorf("access denied: invalid path")
	}
	rel := filepath.Clean(filepath.FromSlash(strings.TrimPrefix(userPath, "/")))
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("access denied: path escape detected")
	}
	if isBlockedPath(rel) {
		return "", fmt.Errorf("access denied: path is in blocklist")
	}
	root, err := filepath.Abs(MountPoint)
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(root, rel)
	if _, err := relativeWithin(root, candidate); err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolved, err := resolveExistingAncestor(candidate)
	if err != nil {
		return "", err
	}
	resolvedRel, err := relativeWithin(resolvedRoot, resolved)
	if err != nil {
		return "", err
	}
	if isBlockedPath(resolvedRel) {
		return "", fmt.Errorf("access denied: symlink target is in blocklist")
	}
	return candidate, nil
}

func relativeWithin(root, target string) (string, error) {
	rel, err := filepath.Rel(root, target)
	if err != nil || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("access denied: path escape detected")
	}
	return rel, nil
}

func isBlockedPath(rel string) bool {
	for _, blocked := range BlockList {
		blocked = filepath.FromSlash(strings.TrimPrefix(blocked, "/"))
		if rel == blocked || strings.HasPrefix(rel, blocked+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// Resolve the nearest existing ancestor so creation of a new file or directory
// receives the same symlink checks as access to an existing file.
func resolveExistingAncestor(target string) (string, error) {
	if _, err := os.Lstat(target); err == nil {
		return filepath.EvalSymlinks(target)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	parent := filepath.Dir(target)
	if parent == target {
		return "", fmt.Errorf("access denied: no existing ancestor")
	}
	resolved, err := resolveExistingAncestor(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolved, filepath.Base(target)), nil
}
