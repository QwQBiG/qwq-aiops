package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFileContentIsVerbatimText(t *testing.T) {
	root := useFileRoot(t)
	for name, content := range map[string]string{
		"page.html": "<html><script>alert('test')</script></html>",
		"data.json": "{\n  \"text\": \"中文\", \"count\": 1\n}\n",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		handleFileContent(w, httptest.NewRequest(http.MethodGet, "/api/files/content?path=/"+name, nil))
		if w.Code != 200 || w.Body.String() != content || w.Header().Get("Content-Type") != "text/plain; charset=utf-8" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("unsafe or modified text response for %s: status=%d headers=%v", name, w.Code, w.Header())
		}
		body, _ := json.Marshal(map[string]string{"path": "/" + name, "content": content})
		response := httptest.NewRecorder()
		handleFileSave(response, httptest.NewRequest(http.MethodPost, "/api/files/save", bytes.NewReader(body)))
		got, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || response.Code != 200 || string(got) != content {
			t.Fatalf("text round trip failed: status=%d error=%v", response.Code, err)
		}
	}
}

func TestFileSaveLimitsAndMethods(t *testing.T) {
	root := useFileRoot(t)
	target := filepath.Join(root, "existing.txt")
	if err := os.WriteFile(target, []byte("unchanged"), 0644); err != nil {
		t.Fatal(err)
	}
	tooLarge, _ := json.Marshal(map[string]string{"path": "/existing.txt", "content": strings.Repeat("a", maxEditableFileSize+1)})
	for _, tc := range []struct {
		method, body string
		status       int
	}{
		{http.MethodGet, "", http.StatusMethodNotAllowed},
		{http.MethodPost, string(tooLarge), http.StatusRequestEntityTooLarge},
		{http.MethodPost, `{"path":"/existing.txt","content":"x"} {}`, http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		handleFileSave(w, httptest.NewRequest(tc.method, "/api/files/save", strings.NewReader(tc.body)))
		if w.Code != tc.status {
			t.Errorf("save: got %d, want %d", w.Code, tc.status)
		}
	}
	got, _ := os.ReadFile(target)
	if string(got) != "unchanged" {
		t.Fatalf("rejected request changed the file")
	}
	w := httptest.NewRecorder()
	handleFileContent(w, httptest.NewRequest(http.MethodGet, "/api/files/content?path="+url.QueryEscape("/"), nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("directory accepted as editable text: %d", w.Code)
	}
}

func TestAtomicWritePreservesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permissions are verified on Linux")
	}
	root := t.TempDir()
	for _, mode := range []os.FileMode{0644, 0755} {
		name := filepath.Join(root, mode.String())
		if err := os.WriteFile(name, []byte("old"), mode); err != nil {
			t.Fatal(err)
		}
		if err := atomicWriteFile(name, []byte("new"), 0600); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(name)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("saved file lost permissions: info=%v err=%v", info, err)
		}
	}
}

func TestFileSavePreservesSymlink(t *testing.T) {
	root := useFileRoot(t)
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "enabled.txt")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	body := `{"path":"/enabled.txt","content":"updated"}`
	w := httptest.NewRecorder()
	handleFileSave(w, httptest.NewRequest(http.MethodPost, "/api/files/save", strings.NewReader(body)))
	got, err := os.ReadFile(target)
	info, statErr := os.Lstat(link)
	if w.Code != http.StatusOK || err != nil || statErr != nil || string(got) != "updated" || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink edit failed: status=%d target=%q read=%v stat=%v", w.Code, got, err, statErr)
	}
}
