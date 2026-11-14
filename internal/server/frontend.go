package server

import (
	"bytes"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var fingerprintedAsset = regexp.MustCompile(`-[A-Za-z0-9_-]{8,}\.[^.]+$`)

func newFrontendHandler(assets fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		content, err := fs.ReadFile(assets, name)
		if err != nil {
			if name == "favicon.ico" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if IsStaticResource(name) {
				http.NotFound(w, r)
				return
			}
			name = "index.html"
			content, err = fs.ReadFile(assets, name)
			if err != nil {
				http.Error(w, "Frontend assets are unavailable; build the frontend before compiling", http.StatusServiceUnavailable)
				return
			}
		}
		w.Header().Set("Content-Type", GetContentType(name))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		if strings.HasPrefix(name, "assets/") && fingerprintedAsset.MatchString(name) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(content))
	})
}
