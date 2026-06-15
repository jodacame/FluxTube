package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jodacame/fluxtube/internal/rules"
	"github.com/jodacame/fluxtube/internal/stream"
)

// streamMaster serves the master playlist, applying rules and the optional
// quality selection from the query string.
func (s *Server) streamMaster(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	res, err := s.ex.Resolve(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	decision := rules.Evaluate(s.store.GetRules(), rules.Subject{
		VideoID: id, Title: res.Title, Channel: res.Channel,
	})
	if decision.Reject {
		writeErr(w, http.StatusForbidden, decision.RejectReason)
		return
	}

	sel := stream.Selection{Height: decision.MaxHeight}
	if q := r.URL.Query().Get("q"); q != "" {
		if h, err := strconv.Atoi(strings.TrimSuffix(q, "p")); err == nil {
			sel.Height = h
		}
	}

	data, err := s.eng.Master(r.Context(), id, sel)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

// streamRendition serves a generated rendition file.
func (s *Server) streamRendition(w http.ResponseWriter, r *http.Request) {
	if err := s.eng.ServeRendition(r.Context(), w, r, r.PathValue("id"), r.PathValue("track"), r.PathValue("file")); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
	}
}

// streamSubPlaylist serves the subtitle media playlist for a language.
func (s *Server) streamSubPlaylist(w http.ResponseWriter, r *http.Request) {
	lang := strings.TrimSuffix(r.PathValue("file"), ".m3u8")
	data, err := s.eng.SubtitlePlaylist(r.Context(), r.PathValue("id"), lang)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = w.Write(data)
}

// streamSubVTT serves the WebVTT subtitle payload for a language.
func (s *Server) streamSubVTT(w http.ResponseWriter, r *http.Request) {
	lang := strings.TrimSuffix(r.PathValue("file"), ".vtt")
	if err := s.eng.Subtitle(r.Context(), w, r, r.PathValue("id"), lang); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
	}
}

// streamProgressive proxies the progressive (muxed) format with Range support.
func (s *Server) streamProgressive(w http.ResponseWriter, r *http.Request) {
	if err := s.eng.Progressive(r.Context(), w, r, r.PathValue("id")); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
	}
}
