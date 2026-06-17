package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jodacame/fluxtube/internal/discovery"
)

// registerDiscovery wires the headless, keyless discovery endpoints.
func (s *Server) registerDiscovery(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/discover/search", s.discoverSearch)
	mux.HandleFunc("GET /api/discover/trending", s.discoverTrending)
	mux.HandleFunc("GET /api/discover/channel/{channelId}", s.discoverChannel)
	mux.HandleFunc("GET /api/discover/channel/{channelId}/videos", s.discoverChannelVideos)
	mux.HandleFunc("GET /api/discover/playlist/{playlistId}", s.discoverPlaylist)
	mux.HandleFunc("GET /api/discover/video/{id}", s.discoverVideo)
	mux.HandleFunc("GET /api/discover/related/{id}", s.discoverRelated)
	mux.HandleFunc("POST /api/discover/recommended", s.discoverRecommended)
}

func (s *Server) disc() *discovery.Service {
	return s.discovery
}

func (s *Server) discoverSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeErr(w, http.StatusBadRequest, "missing q")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	music := r.URL.Query().Get("music") == "1" || r.URL.Query().Get("music") == "true"
	page, err := s.disc().Search(r.Context(), q, r.URL.Query().Get("type"), limit, music)
	respondPage(w, page, err)
}

func (s *Server) discoverTrending(w http.ResponseWriter, r *http.Request) {
	page, err := s.disc().Trending(r.Context(), r.URL.Query().Get("region"))
	respondPage(w, page, err)
}

func (s *Server) discoverChannel(w http.ResponseWriter, r *http.Request) {
	ch, err := s.disc().Channel(r.Context(), r.PathValue("channelId"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ch)
}

func (s *Server) discoverChannelVideos(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	res, err := s.disc().ChannelVideos(r.Context(), r.PathValue("channelId"), r.URL.Query().Get("tab"), page)
	respondPage(w, res, err)
}

func (s *Server) discoverPlaylist(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	res, err := s.disc().Playlist(r.Context(), r.PathValue("playlistId"), page)
	respondPage(w, res, err)
}

// discoverVideo returns cheap catalog info for a video (no format resolve).
func (s *Server) discoverVideo(w http.ResponseWriter, r *http.Request) {
	meta, err := s.ex.Meta(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (s *Server) discoverRelated(w http.ResponseWriter, r *http.Request) {
	page, err := s.disc().Related(r.Context(), r.PathValue("id"))
	respondPage(w, page, err)
}

func (s *Server) discoverRecommended(w http.ResponseWriter, r *http.Request) {
	var req discovery.RecommendRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	page, err := s.disc().Recommended(r.Context(), req)
	respondPage(w, page, err)
}

func respondPage(w http.ResponseWriter, page discovery.Page, err error) {
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}
