package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jodacame/fluxtube/internal/config"
	"github.com/jodacame/fluxtube/internal/extractor"
)

// videoDTO is a library entry plus computed live state.
type videoDTO struct {
	config.Entry
	Active     bool   `json:"active"`
	State      string `json:"state"` // "idle" | "streaming"
	AudioReady bool   `json:"audioReady"`
}

func (s *Server) toDTO(e config.Entry) videoDTO {
	active := s.eng.Active(e.ID)
	state := "idle"
	if active {
		state = "streaming"
	}
	return videoDTO{Entry: e, Active: active, State: state, AudioReady: s.eng.HasAudioFile(e.ID)}
}

// addVideo registers a video in the library using cheap oEmbed metadata.
func (s *Server) addVideo(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Music bool   `json:"music"`
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

	entry := config.Entry{ID: id, AddedAt: time.Now().Unix(), Kind: "video"}
	if meta, err := s.ex.Meta(r.Context(), id); err == nil {
		entry.Title = meta.Title
		entry.Channel = meta.Channel
		entry.ChannelID = meta.ChannelID
		entry.Thumbnail = meta.Thumbnail
	}
	if body.Music {
		entry.Kind = "music"
	}
	if err := s.store.AddEntry(entry); err != nil {
		writeErr(w, http.StatusInternalServerError, "store error")
		return
	}
	// Prepare the persistent audio file in the background so it is ready and
	// never re-downloaded.
	if entry.Kind == "music" {
		go func() { _, _ = s.eng.AudioFile(context.Background(), id) }()
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
	// Resolving (playing) does not add to the library; it only refreshes the
	// metadata of an entry the user explicitly saved.
	if entry, ok := s.store.GetEntry(id); ok {
		entry.Title = res.Title
		entry.Channel = res.Channel
		entry.ChannelID = res.ChannelID
		entry.Thumbnail = res.Thumbnail
		entry.Duration = res.Duration
		_ = s.store.AddEntry(entry)
	} else if res.Music && s.store.Get().Music.AutoSave {
		// Intelligent auto-save: a played video detected as music is saved as
		// music and its audio persisted, with no rule needed.
		s.autoSaveMusic(res)
	}

	writeJSON(w, http.StatusOK, res)
}

// autoSaveMusic stores a detected song as music and persists its audio.
func (s *Server) autoSaveMusic(res *extractor.Resolved) {
	entry := config.Entry{
		ID: res.ID, Title: res.Title, Channel: res.Channel, ChannelID: res.ChannelID,
		Thumbnail: res.Thumbnail, Duration: res.Duration, Kind: "music", AddedAt: time.Now().Unix(),
	}
	_ = s.store.AddEntry(entry)
	go func() { _, _ = s.eng.AudioFile(context.Background(), res.ID) }()
	s.hub.broadcast(event{Type: "added", ID: res.ID})
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
