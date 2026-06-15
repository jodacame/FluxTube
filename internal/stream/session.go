package stream

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jodacame/fluxtube/internal/extractor"
)

// session holds the live streaming state for one video, shared by all clients
// watching it. Renditions (video and per-language audio) are generated lazily.
type session struct {
	e   *Engine
	id  string
	res *extractor.Resolved
	dir string

	mu         sync.Mutex
	renditions map[string]*rendition
	accessed   time.Time
	closed     bool

	video extractor.VideoFormat // selected video rendition
}

func newSession(e *Engine, id string, res *extractor.Resolved) *session {
	s := &session{
		e:          e,
		id:         id,
		res:        res,
		dir:        filepath.Join(e.opt.CacheRoot, id),
		renditions: map[string]*rendition{},
		accessed:   time.Now(),
	}
	switch {
	case len(res.Video) > 0:
		s.video = pickVideo(res.Video, e.opt.DefaultMaxHeight)
	case len(res.Progressive) > 0:
		// Restricted environments may only expose a progressive format; use it
		// as a single muxed rendition (it already carries audio).
		s.video = res.Progressive[0]
	}
	_ = os.MkdirAll(s.dir, 0o755)
	return s
}

func (s *session) touch() {
	s.mu.Lock()
	s.accessed = time.Now()
	s.mu.Unlock()
}

func (s *session) lastAccess() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.accessed
}

func (s *session) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	rs := s.renditions
	s.renditions = map[string]*rendition{}
	s.mu.Unlock()

	for _, r := range rs {
		r.stop()
	}
	_ = os.RemoveAll(s.dir)
}

// sourceFor returns the source URL and whether it exists for a track name.
func (s *session) sourceFor(track string) (string, bool) {
	if track == "video" {
		if s.video.URL != "" {
			return s.video.URL, true
		}
		return "", false
	}
	if lang, ok := strings.CutPrefix(track, "a-"); ok {
		for _, a := range s.res.Audio {
			if audioTrackName(a) == lang {
				return a.URL, true
			}
		}
	}
	return "", false
}

// rendition lazily runs ffmpeg to remux a single source into an fMP4 HLS
// rendition under the session directory.
type rendition struct {
	dir    string
	once   sync.Once
	mu     sync.Mutex
	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   bool
	err    error
}

// ensure starts the generator for a track if not already running.
func (s *session) ensure(track string) (*rendition, error) {
	src, ok := s.sourceFor(track)
	if !ok {
		return nil, os.ErrNotExist
	}
	s.mu.Lock()
	r := s.renditions[track]
	if r == nil {
		r = &rendition{dir: filepath.Join(s.dir, track)}
		s.renditions[track] = r
	}
	s.mu.Unlock()

	r.once.Do(func() {
		_ = os.MkdirAll(r.dir, 0o755)
		go s.runFFmpeg(r, src)
	})
	return r, nil
}

// runFFmpeg remuxes the source into fMP4 HLS segments with stream copy.
func (s *session) runFFmpeg(r *rendition, src string) {
	// Respect the global ffmpeg concurrency cap.
	s.e.sem <- struct{}{}
	defer func() { <-s.e.sem }()

	ctx, cancel := context.WithCancel(context.Background())
	args := []string{
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-user_agent", s.e.opt.UserAgent,
		"-i", src,
		"-map", "0", "-c", "copy",
		"-f", "hls",
		"-hls_time", itoa(s.e.opt.SegmentSeconds),
		"-hls_list_size", "0",
		// EVENT (not VOD) so ffmpeg writes the playlist incrementally as each
		// segment is produced; it is appended-only and gains ENDLIST on
		// completion, which players treat as a seekable VOD.
		"-hls_playlist_type", "event",
		"-hls_segment_type", "fmp4",
		"-hls_flags", "independent_segments",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", filepath.Join(r.dir, "seg%05d.m4s"),
		filepath.Join(r.dir, "index.m3u8"),
	}
	cmd := exec.CommandContext(ctx, s.e.opt.FFmpegPath, args...)
	if logf, lerr := os.Create(filepath.Join(r.dir, "ffmpeg.log")); lerr == nil {
		cmd.Stderr = logf
		defer logf.Close()
	}
	r.mu.Lock()
	r.cmd = cmd
	r.cancel = cancel
	r.mu.Unlock()

	err := cmd.Run()

	r.mu.Lock()
	r.done = true
	r.err = err
	r.mu.Unlock()
}

func (r *rendition) isDone() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.done
}

func (r *rendition) stop() {
	r.mu.Lock()
	c := r.cancel
	r.mu.Unlock()
	if c != nil {
		c()
	}
}
