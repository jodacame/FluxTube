package api

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler serves the embedded SPA, falling back to index.html for client
// routes (e.g. /app/...). When the UI is not built it returns a placeholder.
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.ui))
	hasIndex := func() bool {
		_, err := fs.Stat(s.ui, "index.html")
		return err == nil
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hasIndex() {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html><title>FluxTube</title><h1>FluxTube</h1><p>UI not built yet.</p>"))
			return
		}
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(s.ui, p); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			http.ServeFileFS(w, r2, s.ui, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
