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
