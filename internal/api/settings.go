package api

import (
	"encoding/json"
	"net/http"

	"github.com/jodacame/fluxtube/internal/config"
	"github.com/jodacame/fluxtube/internal/rules"
)

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	cfg := s.store.Get()
	cfg.APIToken = maskToken(cfg.APIToken) // never echo the real token back
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) putSettings(w http.ResponseWriter, r *http.Request) {
	var cfg config.Settings
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	// Preserve the stored token unless a new (unmasked) one is supplied.
	if cfg.APIToken == "" || cfg.APIToken == maskToken(s.store.Get().APIToken) {
		cfg.APIToken = s.store.Get().APIToken
	}
	if err := s.store.PutSettings(cfg); err != nil {
		writeErr(w, http.StatusInternalServerError, "store error")
		return
	}
	s.applySettings(cfg)
	out := cfg
	out.APIToken = maskToken(out.APIToken)
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getRules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.GetRules())
}

func (s *Server) putRules(w http.ResponseWriter, r *http.Request) {
	var list []rules.Rule
	if err := json.NewDecoder(r.Body).Decode(&list); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	for i, rule := range list {
		if !rule.Valid() {
			writeErr(w, http.StatusBadRequest, "invalid rule at index "+itoa(i))
			return
		}
	}
	if err := s.store.PutRules(list); err != nil {
		writeErr(w, http.StatusInternalServerError, "store error")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// applySettings propagates relevant live settings to the extractor and engine.
func (s *Server) applySettings(cfg config.Settings) {
	s.ex.SetCookies(cfg.YouTube.CookiesFile)
	s.eng.SetMusicDir(cfg.Music.Dir)
}

func maskToken(t string) string {
	if t == "" {
		return ""
	}
	return "********"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
