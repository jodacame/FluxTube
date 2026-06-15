package main

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler serves the embedded single-page app, falling back to index.html
// for client-side routes. When the UI has not been built yet (empty dist) it
// returns a small placeholder so the binary still runs.
func spaHandler(ui fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(ui))
	hasIndex := func() bool {
		_, err := fs.Stat(ui, "index.html")
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
		if _, err := fs.Stat(ui, p); err != nil {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			http.ServeFileFS(w, r2, ui, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
