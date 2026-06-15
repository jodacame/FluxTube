package api

import "net/http"

// registerDiscovery wires the headless discovery endpoints. The full keyless
// provider implementation is added in a later iteration; until then the routes
// respond with 501 so clients can detect availability.
func (s *Server) registerDiscovery(mux *http.ServeMux) {
	notImpl := func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, http.StatusNotImplemented, "discovery not available")
	}
	mux.HandleFunc("GET /api/discover/search", notImpl)
	mux.HandleFunc("GET /api/discover/trending", notImpl)
	mux.HandleFunc("GET /api/discover/channel/{channelId}", notImpl)
	mux.HandleFunc("GET /api/discover/channel/{channelId}/videos", notImpl)
	mux.HandleFunc("GET /api/discover/playlist/{playlistId}", notImpl)
	mux.HandleFunc("GET /api/discover/video/{id}", notImpl)
	mux.HandleFunc("GET /api/discover/related/{id}", notImpl)
	mux.HandleFunc("POST /api/discover/recommended", notImpl)
}
