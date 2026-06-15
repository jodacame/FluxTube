package stream

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jodacame/fluxtube/internal/extractor"
)

// ErrNotReady indicates a requested artifact is still being generated.
var ErrNotReady = errors.New("stream artifact not ready")

// Master writes the multivariant master playlist for a video, ensuring the
// session exists. sel optionally overrides the default video quality by height.
func (e *Engine) Master(ctx context.Context, id string, sel Selection) ([]byte, error) {
	s, err := e.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	s.applySelection(sel)
	return buildMaster(s), nil
}

// ServeRendition serves a rendition artifact: the full VOD media playlist
// (generated, so the client can seek anywhere), the init segment, or a media
// segment (produced on demand, restarting the generator on a forward seek).
func (e *Engine) ServeRendition(ctx context.Context, w http.ResponseWriter, r *http.Request, id, track, file string) error {
	if !safeName(track) || !safeName(file) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return nil
	}
	s, err := e.getSession(ctx, id)
	if err != nil {
		return err
	}

	switch {
	case file == "index.m3u8":
		setStreamHeaders(w, file)
		_, _ = w.Write(buildMediaPlaylist(s.res.Duration, e.opt.SegmentSeconds))
		return nil

	case file == "init.mp4":
		rd, err := s.ensureInit(track)
		if err != nil {
			http.Error(w, "unknown track", http.StatusNotFound)
			return nil
		}
		path := filepath.Join(rd.dir, "init.mp4")
		if !waitFileExists(path, 30*time.Second) {
			http.Error(w, "not ready", http.StatusGatewayTimeout)
			return nil
		}
		setStreamHeaders(w, file)
		http.ServeFile(w, r, path)
		return nil

	default: // segment
		n, ok := parseSegName(file)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return nil
		}
		rd, err := s.ensureSegment(track, n)
		if err != nil {
			http.Error(w, "unknown track", http.StatusNotFound)
			return nil
		}
		path := filepath.Join(rd.dir, file)
		if !waitSegment(rd.dir, file, path, 30*time.Second) {
			http.Error(w, "not ready", http.StatusGatewayTimeout)
			return nil
		}
		setStreamHeaders(w, file)
		http.ServeFile(w, r, path)
		return nil
	}
}

// Subtitle proxies a subtitle track as WebVTT.
func (e *Engine) Subtitle(ctx context.Context, w http.ResponseWriter, r *http.Request, id, lang string) error {
	s, err := e.getSession(ctx, id)
	if err != nil {
		return err
	}
	var url string
	for _, sub := range s.res.Subs {
		if sub.Lang == lang {
			url = sub.URL
			break
		}
	}
	if url == "" {
		http.Error(w, "unknown subtitle", http.StatusNotFound)
		return nil
	}
	return serveVTT(ctx, w, e.opt.UserAgent, url)
}

// SubtitlePlaylist returns a minimal VOD subtitle playlist referencing the VTT.
func (e *Engine) SubtitlePlaylist(ctx context.Context, id, lang string) ([]byte, error) {
	s, err := e.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	dur := s.res.Duration
	return buildSubtitlePlaylist(lang, dur), nil
}

// Progressive proxies the best progressive (muxed) format with Range support.
func (e *Engine) Progressive(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) error {
	s, err := e.getSession(ctx, id)
	if err != nil {
		return err
	}
	if len(s.res.Progressive) == 0 {
		http.Error(w, "no progressive format", http.StatusNotFound)
		return nil
	}
	return proxyProgressive(ctx, w, r, e.opt.UserAgent, s.res.Progressive[0].URL)
}

// Resolved returns the resolved info for a video (cached), starting no session.
func (e *Engine) Resolved(ctx context.Context, id string) (*extractor.Resolved, error) {
	return e.ex.Resolve(ctx, id)
}

// applySelection picks the video rendition matching the requested height.
func (s *session) applySelection(sel Selection) {
	if sel.Height <= 0 || len(s.res.Video) == 0 {
		return
	}
	v := pickVideo(s.res.Video, sel.Height)
	s.mu.Lock()
	s.video = v
	s.mu.Unlock()
}

// Selection captures optional client stream preferences.
type Selection struct {
	Height int // max video height, 0 = best
}

// waitFileExists polls until a file exists or the timeout elapses.
func waitFileExists(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(120 * time.Millisecond)
	}
}

// waitSegment blocks until a segment is fully written, which the generator's
// internal playlist signals by listing it.
func waitSegment(dir, file, path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	ff := filepath.Join(dir, "_ff.m3u8")
	for {
		if playlistHas(ff, file) {
			return true
		}
		if time.Now().After(deadline) {
			// Fall back to plain existence in case the playlist lags.
			_, err := os.Stat(path)
			return err == nil
		}
		time.Sleep(120 * time.Millisecond)
	}
}

func setStreamHeaders(w http.ResponseWriter, file string) {
	switch {
	case strings.HasSuffix(file, ".m3u8"):
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Set("Cache-Control", "no-cache")
	case strings.HasSuffix(file, ".m4s"), strings.HasSuffix(file, ".mp4"):
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
}
