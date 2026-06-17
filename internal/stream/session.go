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

// sourceFor returns the source URL, its bound User-Agent, and whether it exists
// for a track name.
func (s *session) sourceFor(track string) (url, ua string, ok bool) {
	if track == "video" {
		if s.video.URL != "" {
			return s.video.URL, s.video.UA, true
		}
		return "", "", false
	}
	if lang, found := strings.CutPrefix(track, "a-"); found {
		for _, a := range s.res.Audio {
			if audioTrackName(a) == lang {
				return a.URL, a.UA, true
			}
		}
	}
	return "", "", false
}

// segAhead is how many segments past the production front a request may be
// before we restart the generator at the requested position (a forward seek).
const segAhead = 10

// rendition runs ffmpeg to remux one source into fMP4 HLS segments. The
// generator can be (re)started at any segment offset so seeking beyond the
// produced buffer works: we serve our own full VOD playlist and produce the
// requested region on demand.
type rendition struct {
	dir     string
	src     string
	ua      string
	mu      sync.Mutex
	cancel  context.CancelFunc
	baseSeg int
	started bool
}

// ensureRendition returns (creating if needed) the rendition for a track.
func (s *session) ensureRendition(track string) (*rendition, error) {
	src, ua, ok := s.sourceFor(track)
	if !ok {
		return nil, os.ErrNotExist
	}
	s.mu.Lock()
	r := s.renditions[track]
	if r == nil {
		r = &rendition{dir: filepath.Join(s.dir, track), src: src, ua: ua}
		s.renditions[track] = r
		_ = os.MkdirAll(r.dir, 0o755)
	}
	s.mu.Unlock()
	return r, nil
}

// ensureInit makes sure a generator is running so the init segment exists.
func (s *session) ensureInit(track string) (*rendition, error) {
	r, err := s.ensureRendition(track)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	if !r.started {
		s.startGenerator(r, 0)
	}
	r.mu.Unlock()
	return r, nil
}

// ensureSegment makes sure segment n is being (or has been) produced, starting
// or restarting the generator at n when n falls outside the active window.
func (s *session) ensureSegment(track string, n int) (*rendition, error) {
	r, err := s.ensureRendition(track)
	if err != nil {
		return nil, err
	}
	if fileExists(filepath.Join(r.dir, segName(n))) {
		return r, nil
	}
	r.mu.Lock()
	last := maxSegIndex(r.dir)
	if !r.started || n < r.baseSeg || n > last+segAhead {
		// Start one segment earlier so the requested segment is reliably
		// covered even when the keyframe lands just before its boundary.
		base := n - 1
		if base < 0 {
			base = 0
		}
		s.startGenerator(r, base)
	}
	r.mu.Unlock()
	return r, nil
}

// startGenerator (re)starts ffmpeg producing segments from segment base.
// Caller must hold r.mu.
func (s *session) startGenerator(r *rendition, base int) {
	if r.cancel != nil {
		r.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.baseSeg = base
	r.started = true
	go s.runFFmpeg(r, base, ctx)
}

// runFFmpeg remuxes from the segment-aligned offset using stream copy. Source
// timestamps are preserved (-copyts) so segments land at the right point on the
// timeline regardless of where generation started, and -start_number keeps the
// filenames aligned with the global segment index.
func (s *session) runFFmpeg(r *rendition, base int, ctx context.Context) {
	s.e.sem <- struct{}{}
	defer func() { <-s.e.sem }()

	seg := s.e.opt.SegmentSeconds
	ua := r.ua
	if ua == "" {
		ua = s.e.opt.UserAgent
	}
	args := []string{
		"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-user_agent", ua,
	}
	if base > 0 {
		args = append(args, "-ss", itoa(base*seg))
	}
	args = append(args, "-i", r.src, "-map", "0", "-c", "copy")
	if base > 0 {
		// Place the restarted segments at their real position on the timeline
		// (input -ss reset timestamps to 0) without copying odd source ts.
		args = append(args, "-output_ts_offset", itoa(base*seg))
	}
	args = append(args,
		"-f", "hls",
		"-hls_time", itoa(seg),
		"-hls_list_size", "0",
		"-hls_flags", "independent_segments+append_list+omit_endlist",
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4",
		"-start_number", itoa(base),
		"-hls_segment_filename", filepath.Join(r.dir, "seg%05d.m4s"),
		filepath.Join(r.dir, "_ff.m3u8"),
	)
	cmd := exec.CommandContext(ctx, s.e.opt.FFmpegPath, args...)
	if logf, lerr := os.Create(filepath.Join(r.dir, "ffmpeg.log")); lerr == nil {
		cmd.Stderr = logf
		defer logf.Close()
	}
	_ = cmd.Run()
}

func (r *rendition) stop() {
	r.mu.Lock()
	c := r.cancel
	r.mu.Unlock()
	if c != nil {
		c()
	}
}
