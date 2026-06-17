// Package stream turns a resolved YouTube video into a seekable HTTP stream.
//
// It produces HLS with fragmented-MP4 renditions (one per track) remuxed with
// ffmpeg using stream copy (no re-encoding), so video plus multiple audio and
// subtitle tracks can be selected by the player — like an MKV, but seekable.
// A plain progressive endpoint is offered as a universal fallback.
//
// Sessions are ephemeral: segments live in a bounded on-disk cache that is
// wiped when playback ends or after an idle timeout. The source video is never
// fully downloaded — only the parts a client actually plays.
package stream

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jodacame/fluxtube/internal/extractor"
)

// Options configures the streaming engine.
type Options struct {
	FFmpegPath       string
	CacheRoot        string
	SegmentSeconds   int
	IdleTimeout      time.Duration
	MaxSessions      int
	MaxFFmpeg        int // concurrent ffmpeg processes cap
	UserAgent        string
	GCInterval       time.Duration
	DefaultMaxHeight int    // default video height cap (0 = best available)
	MaxSizeMB        int    // total on-disk cache cap (0 = unbounded)
	MusicDir         string // persistent store for music (audio-only) files
}

func (o *Options) withDefaults() {
	if o.FFmpegPath == "" {
		o.FFmpegPath = "ffmpeg"
	}
	if o.CacheRoot == "" {
		o.CacheRoot = filepath.Join(os.TempDir(), "fluxtube-cache")
	}
	if o.SegmentSeconds == 0 {
		o.SegmentSeconds = 6
	}
	if o.IdleTimeout == 0 {
		o.IdleTimeout = 3 * time.Minute
	}
	if o.MaxSessions == 0 {
		o.MaxSessions = 8
	}
	if o.MaxFFmpeg == 0 {
		o.MaxFFmpeg = 4
	}
	if o.UserAgent == "" {
		o.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"
	}
	if o.GCInterval == 0 {
		o.GCInterval = 30 * time.Second
	}
	if o.DefaultMaxHeight == 0 {
		// Default to 1080p for a sensible quality/efficiency balance on modest
		// hardware; clients can request higher via the master selection.
		o.DefaultMaxHeight = 1080
	}
	if o.MusicDir == "" {
		o.MusicDir = filepath.Join(o.CacheRoot, "..", "music")
	}
}

// Engine manages streaming sessions.
type Engine struct {
	ex  *extractor.Extractor
	opt Options
	sem chan struct{}

	mu       sync.Mutex
	sessions map[string]*session

	audioMu sync.Mutex // serialises music (audio-only) preparation

	accessMu    sync.Mutex
	audioAccess map[string]time.Time // last time each music track was served

	stopGC chan struct{}
}

// audioActiveWindow is how long after the last audio request a music track is
// still considered "now playing" (covers a song buffered in one fetch).
const audioActiveWindow = 5 * time.Minute

// New creates a streaming engine bound to an extractor.
func New(ex *extractor.Extractor, opt Options) (*Engine, error) {
	opt.withDefaults()
	if err := os.MkdirAll(opt.CacheRoot, 0o755); err != nil {
		return nil, fmt.Errorf("cache root: %w", err)
	}
	// Clean any stale segment cache from a previous run.
	_ = clearDir(opt.CacheRoot)
	// Music is persistent — only ensure the directory exists.
	_ = os.MkdirAll(opt.MusicDir, 0o755)

	e := &Engine{
		ex:          ex,
		opt:         opt,
		sem:         make(chan struct{}, opt.MaxFFmpeg),
		sessions:    map[string]*session{},
		audioAccess: map[string]time.Time{},
		stopGC:      make(chan struct{}),
	}
	go e.gcLoop()
	return e, nil
}

// Close stops background work and wipes the cache.
func (e *Engine) Close() {
	close(e.stopGC)
	e.mu.Lock()
	for id, s := range e.sessions {
		s.close()
		delete(e.sessions, id)
	}
	e.mu.Unlock()
	_ = clearDir(e.opt.CacheRoot)
}

// session returns an existing session or creates one, resolving the video.
func (e *Engine) getSession(ctx context.Context, id string) (*session, error) {
	e.mu.Lock()
	if s, ok := e.sessions[id]; ok {
		s.touch()
		e.mu.Unlock()
		return s, nil
	}
	// Enforce the active-session cap by evicting the least-recently-used one.
	if len(e.sessions) >= e.opt.MaxSessions {
		e.evictLRULocked()
	}
	e.mu.Unlock()

	res, err := e.ex.Resolve(ctx, id)
	if err != nil {
		return nil, err
	}
	s := newSession(e, id, res)

	e.mu.Lock()
	defer e.mu.Unlock()
	if existing, ok := e.sessions[id]; ok { // lost a race
		s.close()
		existing.touch()
		return existing, nil
	}
	e.sessions[id] = s
	return s, nil
}

// Stop tears down a session without removing extractor cache.
func (e *Engine) Stop(id string) {
	e.mu.Lock()
	s, ok := e.sessions[id]
	if ok {
		delete(e.sessions, id)
	}
	e.mu.Unlock()
	if ok {
		s.close()
	}
}

// Active reports whether a session is currently live.
func (e *Engine) Active(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.sessions[id]
	return ok
}

// ActiveCount returns the number of live sessions.
func (e *Engine) ActiveCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.sessions)
}

// ActiveStreams returns metadata for every live (HLS) streaming session, so the
// UI can show what is streaming even when it was never saved.
func (e *Engine) ActiveStreams() []extractor.Meta {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]extractor.Meta, 0, len(e.sessions))
	for _, s := range e.sessions {
		out = append(out, s.res.Meta)
	}
	return out
}

func (e *Engine) evictLRULocked() {
	var oldest *session
	var oldestID string
	for id, s := range e.sessions {
		if oldest == nil || s.lastAccess().Before(oldest.lastAccess()) {
			oldest, oldestID = s, id
		}
	}
	if oldest != nil {
		delete(e.sessions, oldestID)
		go oldest.close()
	}
}

func (e *Engine) gcLoop() {
	t := time.NewTicker(e.opt.GCInterval)
	defer t.Stop()
	for {
		select {
		case <-e.stopGC:
			return
		case <-t.C:
			e.gcOnce()
			e.enforceDiskCap()
		}
	}
}

// enforceDiskCap evicts least-recently-used sessions until the on-disk cache is
// under the configured size cap. The currently active session is never evicted,
// so playback is not interrupted.
func (e *Engine) enforceDiskCap() {
	if e.opt.MaxSizeMB <= 0 {
		return
	}
	limit := int64(e.opt.MaxSizeMB) << 20
	for dirSize(e.opt.CacheRoot) > limit {
		e.mu.Lock()
		if len(e.sessions) <= 1 {
			e.mu.Unlock()
			return
		}
		var oldest *session
		var oid string
		for id, s := range e.sessions {
			if oldest == nil || s.lastAccess().Before(oldest.lastAccess()) {
				oldest, oid = s, id
			}
		}
		delete(e.sessions, oid)
		e.mu.Unlock()
		oldest.close()
	}
}

func (e *Engine) gcOnce() {
	now := time.Now()
	var dead []*session
	e.mu.Lock()
	for id, s := range e.sessions {
		if now.Sub(s.lastAccess()) > e.opt.IdleTimeout {
			dead = append(dead, s)
			delete(e.sessions, id)
		}
	}
	e.mu.Unlock()
	for _, s := range dead {
		s.close()
	}
}
