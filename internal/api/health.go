package api

import (
	"net/http"
	"time"
)

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version":        Version,
		"status":         "ok",
		"uptimeSeconds":  int(time.Since(s.start).Seconds()),
		"activeSessions": s.eng.ActiveCount(),
	})
}

// storage reports music/cache usage and free disk space.
func (s *Server) storage(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.eng.Storage())
}
