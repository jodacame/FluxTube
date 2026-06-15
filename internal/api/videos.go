package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jodacame/fluxtube/internal/config"
	"github.com/jodacame/fluxtube/internal/extractor"
)

// videoDTO is a library entry plus computed live state.
type videoDTO struct {
	config.Entry
	Active bool   `json:"active"`
	State  string `json:"state"` // "idle" | "streaming"
}

func (s *Server) toDTO(e config.Entry) videoDTO {
	active := s.eng.Active(e.ID)
	state := "idle"
	if active {
		state = "streaming"
	}
	return videoDTO{Entry: e, Active: active, State: state}
}

// addVideo registers a video in the library using cheap oEmbed metadata.
func (s *Server) addVideo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	input := body.ID
	if input == "" {
		input = body.URL
	}
	id, err := extractor.NormalizeID(input)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid youtube id or url")
		return
	}

	entry := config.Entry{ID: id, AddedAt: time.Now().Unix()}
	if meta, err := s.ex.Meta(r.Context(), id); err == nil {
		entry.Title = meta.Title
		entry.Channel = meta.Channel
		entry.ChannelID = meta.ChannelID
		entry.Thumbnail = meta.Thumbnail
	}
	if err := s.store.AddEntry(entry); err != nil {
		writeErr(w, http.StatusInternalServerError, "store error")
		return
	}
	s.hub.broadcast(event{Type: "added", ID: id})
	writeJSON(w, http.StatusOK, s.toDTO(entry))
}

// listVideos returns the library merged with live playback state.
func (s *Server) listVideos(w http.ResponseWriter, r *http.Request) {
	entries := s.store.ListEntries()
	out := make([]videoDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, s.toDTO(e))
	}
	writeJSON(w, http.StatusOK, out)
}

// getVideo resolves full playback info (formats, audio, subtitles) and refreshes
// the stored metadata.
func (s *Server) getVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, err := s.ex.Resolve(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	// Keep the library entry's metadata fresh.
	entry, ok := s.store.GetEntry(id)
	if !ok {
		entry = config.Entry{ID: id, AddedAt: time.Now().Unix()}
	}
	entry.Title = res.Title
	entry.Channel = res.Channel
	entry.ChannelID = res.ChannelID
	entry.Thumbnail = res.Thumbnail
	entry.Duration = res.Duration
	_ = s.store.AddEntry(entry)

	writeJSON(w, http.StatusOK, res)
}

// stopVideo tears down the live session without removing the library entry.
func (s *Server) stopVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.eng.Stop(id)
	s.hub.broadcast(event{Type: "stopped", ID: id})
	w.WriteHeader(http.StatusNoContent)
}

// deleteVideo removes the entry, stops the session and clears caches.
func (s *Server) deleteVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.eng.Stop(id)
	s.ex.Drop(id)
	_ = s.store.DeleteEntry(id)
	s.hub.broadcast(event{Type: "removed", ID: id})
	w.WriteHeader(http.StatusNoContent)
}
