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

// ServeRendition serves a generated rendition file (index.m3u8, init.mp4 or a
// segment), starting the generator on demand and waiting briefly for segments.
func (e *Engine) ServeRendition(ctx context.Context, w http.ResponseWriter, r *http.Request, id, track, file string) error {
	if !safeName(track) || !safeName(file) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return nil
	}
	s, err := e.getSession(ctx, id)
	if err != nil {
		return err
	}
	rd, err := s.ensure(track)
	if err != nil {
		http.Error(w, "unknown track", http.StatusNotFound)
		return nil
	}

	path := filepath.Join(rd.dir, file)
	if err := waitFor(rd, file, path, 30*time.Second); err != nil {
		http.Error(w, "not ready", http.StatusGatewayTimeout)
		return nil
	}
	setStreamHeaders(w, file)
	http.ServeFile(w, r, path)
	return nil
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

// waitFor blocks until a rendition file is ready to serve or the generator
// finishes. For segments, readiness means the playlist already lists the file
// (so it is fully written); for other files, existence is enough.
func waitFor(rd *rendition, file, path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	isSeg := strings.HasSuffix(file, ".m4s")
	for {
		if isSeg {
			if playlistHas(filepath.Join(rd.dir, "index.m3u8"), file) {
				return nil
			}
		} else if _, err := os.Stat(path); err == nil {
			return nil
		}
		if rd.isDone() {
			if _, err := os.Stat(path); err == nil {
				return nil
			}
			return ErrNotReady
		}
		if time.Now().After(deadline) {
			return ErrNotReady
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
