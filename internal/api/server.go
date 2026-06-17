// Package api exposes the REST + WebSocket surface and serves the embedded UI.
// One process, one port: API, UI, web client and stream all share it.
package api

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/jodacame/fluxtube/internal/config"
	"github.com/jodacame/fluxtube/internal/discovery"
	"github.com/jodacame/fluxtube/internal/extractor"
	"github.com/jodacame/fluxtube/internal/stream"
)

// Version is the running build version.
var Version = "dev"

// Server wires the store, extractor, streaming engine and UI into a handler.
type Server struct {
	store     *config.Store
	ex        *extractor.Extractor
	eng       *stream.Engine
	discovery *discovery.Service
	ui        fs.FS
	hub       *hub
	start     time.Time
}

// New builds the API server.
func New(store *config.Store, ex *extractor.Extractor, eng *stream.Engine, disc *discovery.Service, ui fs.FS) *Server {
	s := &Server{
		store:     store,
		ex:        ex,
		eng:       eng,
		discovery: disc,
		ui:        ui,
		hub:       newHub(),
		start:     time.Now(),
	}
	go s.hub.run()
	go s.broadcastLoop()
	return s
}

// Handler returns the root HTTP handler with routing, auth and CORS.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Library / sessions.
	mux.HandleFunc("POST /api/videos", s.addVideo)
	mux.HandleFunc("GET /api/videos", s.listVideos)
	mux.HandleFunc("GET /api/videos/{id}", s.getVideo)
	mux.HandleFunc("POST /api/videos/{id}/stop", s.stopVideo)
	mux.HandleFunc("DELETE /api/videos/{id}", s.deleteVideo)

	// Settings & rules.
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PUT /api/settings", s.putSettings)
	mux.HandleFunc("GET /api/rules", s.getRules)
	mux.HandleFunc("PUT /api/rules", s.putRules)

	// System.
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/storage", s.storage)
	mux.HandleFunc("GET /api/events", s.hub.serveWS)

	// Discovery (registered separately so the feature can evolve independently).
	s.registerDiscovery(mux)

	// Streaming.
	mux.HandleFunc("GET /stream/{id}/master.m3u8", s.streamMaster)
	mux.HandleFunc("GET /stream/{id}/r/{track}/{file}", s.streamRendition)
	mux.HandleFunc("GET /stream/{id}/subs/{file}", s.streamSubPlaylist)
	mux.HandleFunc("GET /stream/{id}/sub/{file}", s.streamSubVTT)
	mux.HandleFunc("GET /stream/{id}/progressive", s.streamProgressive)
	mux.HandleFunc("GET /stream/{id}/audio", s.streamAudio)

	// UI / web client (SPA fallback).
	mux.Handle("/", s.spaHandler())

	return s.withCORS(s.withAuth(mux))
}

// --- middleware ---

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := s.store.Get().APIToken
		// Stream and UI stay open so saved player URLs work without credentials;
		// only /api/* is guarded when a token is configured.
		if token == "" || !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if got := bearer(r); got == token {
			next.ServeHTTP(w, r)
			return
		}
		writeErr(w, http.StatusUnauthorized, "unauthorized")
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if v, ok := strings.CutPrefix(h, "Bearer "); ok {
		return v
	}
	return ""
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
