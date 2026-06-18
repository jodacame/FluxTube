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

// clearMusic deletes all saved music files and their library entries.
func (s *Server) clearMusic(w http.ResponseWriter, r *http.Request) {
	removed, err := s.eng.ClearMusic()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not clear music")
		return
	}
	for _, e := range s.store.ListEntries() {
		if e.Kind == "music" {
			_ = s.store.DeleteEntry(e.ID)
		}
	}
	s.hub.broadcast(event{Type: "state"})
	writeJSON(w, http.StatusOK, map[string]int{"removed": removed})
}
