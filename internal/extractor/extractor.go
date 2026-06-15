package extractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Options configures an Extractor.
type Options struct {
	YtDlpPath    string        // path/name of the yt-dlp binary
	CookiesFile  string        // optional Netscape cookies file to unlock restricted videos
	ExtractorArg string        // optional extra yt-dlp --extractor-args value
	CacheTTL     time.Duration // how long a resolve stays fresh (before CDN URLs expire)
	NegTTL       time.Duration // how long a failure is remembered
	Timeout      time.Duration // per-resolve timeout
	AutoSubLangs []string      // languages for which automatic captions are exposed (manual subs always kept)
}

func (o *Options) withDefaults() {
	if o.YtDlpPath == "" {
		o.YtDlpPath = "yt-dlp"
	}
	if o.CacheTTL == 0 {
		o.CacheTTL = 5 * time.Hour
	}
	if o.NegTTL == 0 {
		o.NegTTL = 30 * time.Second
	}
	if o.Timeout == 0 {
		o.Timeout = 90 * time.Second
	}
	if o.AutoSubLangs == nil {
		// A video can expose ~150 auto-caption languages; expose only a sane
		// default set so playlists stay usable. Manual subtitles are always kept.
		o.AutoSubLangs = []string{"en", "es"}
	}
}

// Extractor resolves YouTube videos. It is safe for concurrent use.
type Extractor struct {
	opt   Options
	cache *cache
	http  *http.Client
}

// New creates an Extractor with the given options.
func New(opt Options) *Extractor {
	opt.withDefaults()
	return &Extractor{
		opt:   opt,
		cache: newCache(),
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

// SetCookies updates the cookies file used for subsequent resolves.
func (e *Extractor) SetCookies(path string) { e.opt.CookiesFile = path }

var (
	// ErrInvalidID is returned when a value cannot be parsed into a video ID.
	ErrInvalidID = errors.New("invalid youtube video id")

	idRe  = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	urlRe = regexp.MustCompile(`(?:v=|youtu\.be/|/shorts/|/embed/|/v/)([A-Za-z0-9_-]{11})`)
)

// NormalizeID accepts a bare video ID or any common YouTube URL and returns the
// canonical 11-character video ID.
func NormalizeID(input string) (string, error) {
	s := strings.TrimSpace(input)
	if idRe.MatchString(s) {
		return s, nil
	}
	if m := urlRe.FindStringSubmatch(s); m != nil {
		return m[1], nil
	}
	return "", ErrInvalidID
}

// oEmbed is the subset of the YouTube oEmbed response we use.
type oEmbed struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// Meta returns cheap catalog metadata via oEmbed without a full resolve.
func (e *Extractor) Meta(ctx context.Context, id string) (*Meta, error) {
	if !idRe.MatchString(id) {
		return nil, ErrInvalidID
	}
	endpoint := "https://www.youtube.com/oembed?format=json&url=" +
		url.QueryEscape("https://www.youtube.com/watch?v="+id)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oembed: status %d", resp.StatusCode)
	}
	var o oEmbed
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return nil, err
	}
	m := &Meta{
		ID:        id,
		Title:     o.Title,
		Channel:   o.AuthorName,
		Thumbnail: o.ThumbnailURL,
	}
	if cid := lastPath(o.AuthorURL); cid != "" {
		m.ChannelID = cid
	}
	return m, nil
}

// Resolve returns the full resolve result for a video, using the cache,
// single-flight and negative cache to minimise upstream requests.
func (e *Extractor) Resolve(ctx context.Context, id string) (*Resolved, error) {
	if !idRe.MatchString(id) {
		return nil, ErrInvalidID
	}
	if r, ok := e.cache.get(id); ok {
		return r, nil
	}
	if err, ok := e.cache.negGet(id); ok {
		return nil, err
	}

	// Single-flight: only one resolve per id runs at a time.
	e.cache.mu.Lock()
	if c, ok := e.cache.inflight[id]; ok {
		e.cache.mu.Unlock()
		c.wg.Wait()
		return c.res, c.err
	}
	c := &call{}
	c.wg.Add(1)
	e.cache.inflight[id] = c
	e.cache.mu.Unlock()

	c.res, c.err = e.runResolve(ctx, id)

	e.cache.mu.Lock()
	delete(e.cache.inflight, id)
	e.cache.mu.Unlock()
	c.wg.Done()

	if c.err != nil {
		e.cache.negPut(id, c.err, e.opt.NegTTL)
		return nil, c.err
	}
	e.cache.put(id, c.res, e.opt.CacheTTL)
	return c.res, nil
}

// Drop clears any cached state for a video (used when it is removed).
func (e *Extractor) Drop(id string) { e.cache.drop(id) }

// runResolve invokes yt-dlp and parses its JSON output.
func (e *Extractor) runResolve(ctx context.Context, id string) (*Resolved, error) {
	ctx, cancel := context.WithTimeout(ctx, e.opt.Timeout)
	defer cancel()

	args := []string{"-J", "--no-warnings", "--no-playlist", "--no-progress"}
	if e.opt.CookiesFile != "" {
		args = append(args, "--cookies", e.opt.CookiesFile)
	}
	if e.opt.ExtractorArg != "" {
		args = append(args, "--extractor-args", e.opt.ExtractorArg)
	}
	args = append(args, "--", "https://www.youtube.com/watch?v="+id)

	cmd := exec.CommandContext(ctx, e.opt.YtDlpPath, args...)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("yt-dlp failed: %s", firstLine(ee.Stderr))
		}
		return nil, fmt.Errorf("yt-dlp: %w", err)
	}

	var raw rawInfo
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse yt-dlp output: %w", err)
	}
	allowedAuto := map[string]bool{}
	for _, l := range e.opt.AutoSubLangs {
		allowedAuto[l] = true
	}
	res := parseInfo(&raw, allowedAuto)
	if len(res.Video) == 0 && len(res.Progressive) == 0 {
		return nil, errors.New("no playable formats available")
	}
	now := time.Now()
	res.ResolvedAt = now
	res.ExpiresAt = now.Add(e.opt.CacheTTL)
	return res, nil
}

func lastPath(u string) string {
	u = strings.TrimRight(u, "/")
	if i := strings.LastIndex(u, "/"); i >= 0 {
		return u[i+1:]
	}
	return ""
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return "unknown error"
	}
	return s
}
