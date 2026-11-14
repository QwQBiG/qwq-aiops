package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func useFileRoot(t *testing.T) string {
	t.Helper()
	old := MountPoint
	MountPoint = t.TempDir()
	t.Cleanup(func() { MountPoint = old })
	return MountPoint
}

func TestResolveSafePath(t *testing.T) {
	root := useFileRoot(t)
	for _, input := range []string{"", "/", "/notes/new.txt", "notes/new.txt", "/process/notes", "/devtools"} {
		t.Run(input, func(t *testing.T) {
			got, err := resolveSafePath(input)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := relativeWithin(root, got); err != nil {
				t.Fatalf("resolved outside root: %s", got)
			}
		})
	}
	for _, input := range []string{"../outside", "/../outside", "../../sibling/file", "/proc", "/proc/self", "proc/self", "/sys/kernel", "/dev/null", "/boot", "/notes/../proc/self", "..\\outside", "bad\x00path"} {
		t.Run(input, func(t *testing.T) {
			if got, err := resolveSafePath(input); err == nil {
				t.Fatalf("unsafe path accepted as %s", got)
			}
		})
	}
}

func TestResolveSafePathSymlinks(t *testing.T) {
	root := useFileRoot(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, target := range []string{"proc", "notes"} {
		if err := os.Mkdir(filepath.Join(root, target), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(root, target), filepath.Join(root, target+"-link")); err != nil {
			t.Fatal(err)
		}
	}
	for _, input := range []string{"/escape", "/escape/new/file", "/proc-link/self"} {
		if got, err := resolveSafePath(input); err == nil {
			t.Errorf("symlink escape %s accepted as %s", input, got)
		}
	}
	if _, err := resolveSafePath("/notes-link/new/file"); err != nil {
		t.Fatalf("in-root symlink rejected: %v", err)
	}
}

func TestFileActionsRequirePostAndProtectRoot(t *testing.T) {
	root := useFileRoot(t)
	for _, tc := range []struct {
		method, query string
		want          int
	}{
		{http.MethodGet, "type=delete&path=/", http.StatusMethodNotAllowed},
		{http.MethodPost, "type=delete&path=/", http.StatusForbidden},
		{http.MethodPost, "type=delete&path=.", http.StatusForbidden},
		{http.MethodPost, "type=mkdir&path=../outside", http.StatusForbidden},
		{http.MethodPost, "type=mkdir&path=/notes", http.StatusOK},
		{http.MethodPost, "type=delete&path=/notes", http.StatusOK},
	} {
		w := httptest.NewRecorder()
		handleFileAction(w, httptest.NewRequest(tc.method, "/api/files/action?"+tc.query, nil))
		var body FileResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != tc.want || body.Code != tc.want {
			t.Errorf("%s %s: HTTP=%d body=%d, want %d", tc.method, tc.query, w.Code, body.Code, tc.want)
		}
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("file root was removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "notes")); !os.IsNotExist(err) {
		t.Fatalf("delete did not remove notes: %v", err)
	}
}
