package server

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFrontendValidationRejectsMissingEntryAssets(t *testing.T) {
	files := fstest.MapFS{
		"dist/index.html":          {Data: []byte(`<!doctype html><html><script type="module" src="/assets/missing.js"></script><link rel="stylesheet" href="/assets/app.css"></html>`)},
		"dist/assets/unrelated.js": {Data: []byte("console.log('unrelated')")},
		"dist/assets/app.css":      {Data: []byte("body{}")},
	}
	result := NewFrontendManagerWithFS(files, "dist").ValidateResources()
	if result.Valid || !strings.Contains(strings.Join(result.Errors, " "), "missing.js") {
		t.Fatalf("missing startup asset passed resource validation: %+v", result)
	}
}

type unreadableAssetFS struct{ fs.FS }

func (f unreadableAssetFS) Open(name string) (fs.File, error) {
	if name == "dist/assets/app.js" {
		return nil, fs.ErrPermission
	}
	return f.FS.Open(name)
}

func TestFrontendValidationReportsUnreadableAssets(t *testing.T) {
	files := fstest.MapFS{
		"dist/index.html":     {Data: []byte(`<!doctype html><html><script src="/assets/app.js"></script></html>`)},
		"dist/assets/app.js":  {Data: []byte("console.log('app')")},
		"dist/assets/app.css": {Data: []byte("body{}")},
	}
	result := NewFrontendManagerWithFS(unreadableAssetFS{files}, "dist").ValidateResources()
	if result.Valid || len(result.Stats.CorruptedFiles) != 1 || result.Stats.CorruptedFiles[0] != "assets/app.js" {
		t.Fatalf("unreadable asset was not reported: %+v", result)
	}
}

func TestFrontendValidationAcceptsLocalURLsWithQuery(t *testing.T) {
	files := fstest.MapFS{
		"dist/index.html":     {Data: []byte(`<!doctype html><html><script src='./assets/app.js?v=1#bundle'></script><link rel='stylesheet' href='/assets/app.css'></html>`)},
		"dist/assets/app.js":  {Data: []byte("console.log('app')")},
		"dist/assets/app.css": {Data: []byte("body{}")},
	}
	if result := NewFrontendManagerWithFS(files, "dist").ValidateResources(); !result.Valid {
		t.Fatalf("valid local startup resources were rejected: %+v", result)
	}
}

func TestFrontendValidationRejectsUnreadableAssetTree(t *testing.T) {
	result := NewFrontendManagerWithFS(fstest.MapFS{}, "missing").ValidateResources()
	if result.Valid || !strings.Contains(strings.Join(result.Errors, " "), "Cannot inspect frontend resources") {
		t.Fatalf("missing resource directory passed validation: %+v", result)
	}
}
