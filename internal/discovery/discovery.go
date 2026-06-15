// Package discovery provides a headless, keyless catalog over YouTube (search,
// trending, channels, playlists, related and recommendations).
//
// It stores nothing persistently: results are served live through a provider
// (yt-dlp by default) and held only in a short-lived in-memory TTL cache, with
// single-flight de-duplication, to keep upstream requests minimal.
package discovery

import (
	"context"
	"sync"
	"time"
)

// VideoItem is a catalog entry for a video.
type VideoItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Channel   string `json:"channel"`
	ChannelID string `json:"channelId"`
	Thumbnail string `json:"thumbnail"`
	Duration  int    `json:"duration"`
	Views     int64  `json:"views"`
}

// ChannelItem is catalog information about a channel.
type ChannelItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Thumbnail   string `json:"thumbnail"`
	Subscribers int64  `json:"subscribers"`
	Description string `json:"description,omitempty"`
}

// Page is a paginated list of videos.
type Page struct {
	Videos   []VideoItem   `json:"videos"`
	Channels []ChannelItem `json:"channels,omitempty"`
	NextPage string        `json:"nextPage,omitempty"`
}

// Provider is a catalog backend. Implementations must be keyless.
type Provider interface {
	Search(ctx context.Context, query, kind string, limit int) (Page, error)
	Trending(ctx context.Context, region string) (Page, error)
	Channel(ctx context.Context, channelID string) (ChannelItem, error)
	ChannelVideos(ctx context.Context, channelID, tab string, page int) (Page, error)
	Playlist(ctx context.Context, playlistID string, page int) (Page, error)
	Related(ctx context.Context, videoID string) (Page, error)
}

// Service wraps a provider with an ephemeral TTL cache and de-duplication.
type Service struct {
	p   Provider
	ttl time.Duration

	mu       sync.Mutex
	items    map[string]cacheItem
	inflight map[string]*flight
}

type cacheItem struct {
	val any
	exp time.Time
}

type flight struct {
	wg  sync.WaitGroup
	val any
	err error
}

// NewService creates a discovery service.
func NewService(p Provider, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	s := &Service{
		p:        p,
		ttl:      ttl,
		items:    map[string]cacheItem{},
		inflight: map[string]*flight{},
	}
	// Drop expired entries periodically so memory stays bounded.
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			now := time.Now()
			s.mu.Lock()
			for k, v := range s.items {
				if now.After(v.exp) {
					delete(s.items, k)
				}
			}
			s.mu.Unlock()
		}
	}()
	return s
}

// do runs fn for key, serving a cached value when fresh and coalescing
// concurrent identical requests.
func (s *Service) do(key string, fn func() (any, error)) (any, error) {
	s.mu.Lock()
	if it, ok := s.items[key]; ok && time.Now().Before(it.exp) {
		s.mu.Unlock()
		return it.val, nil
	}
	if fl, ok := s.inflight[key]; ok {
		s.mu.Unlock()
		fl.wg.Wait()
		return fl.val, fl.err
	}
	fl := &flight{}
	fl.wg.Add(1)
	s.inflight[key] = fl
	s.mu.Unlock()

	fl.val, fl.err = fn()

	s.mu.Lock()
	delete(s.inflight, key)
	if fl.err == nil {
		s.items[key] = cacheItem{val: fl.val, exp: time.Now().Add(s.ttl)}
	}
	s.mu.Unlock()
	fl.wg.Done()
	return fl.val, fl.err
}
