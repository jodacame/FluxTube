package discovery

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Search returns catalog results for a query. When music is true, results are
// biased toward official song audio/video (artist "- Topic" channels, etc.).
func (s *Service) Search(ctx context.Context, query, kind string, limit int, music bool) (Page, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	key := fmt.Sprintf("search|%s|%s|%d|%t", query, kind, limit, music)
	v, err := s.do(key, func() (any, error) { return s.p.Search(ctx, query, kind, limit, music) })
	return asPage(v), err
}

// Trending returns trending videos for a region (empty = default).
func (s *Service) Trending(ctx context.Context, region string) (Page, error) {
	v, err := s.do("trending|"+region, func() (any, error) { return s.p.Trending(ctx, region) })
	return asPage(v), err
}

// Channel returns channel metadata.
func (s *Service) Channel(ctx context.Context, channelID string) (ChannelItem, error) {
	v, err := s.do("channel|"+channelID, func() (any, error) { return s.p.Channel(ctx, channelID) })
	if err != nil {
		return ChannelItem{}, err
	}
	if c, ok := v.(ChannelItem); ok {
		return c, nil
	}
	return ChannelItem{}, nil
}

// ChannelVideos returns a page of a channel's videos.
func (s *Service) ChannelVideos(ctx context.Context, channelID, tab string, page int) (Page, error) {
	key := fmt.Sprintf("chvids|%s|%s|%d", channelID, tab, page)
	v, err := s.do(key, func() (any, error) { return s.p.ChannelVideos(ctx, channelID, tab, page) })
	return asPage(v), err
}

// Playlist returns a page of a playlist's videos.
func (s *Service) Playlist(ctx context.Context, playlistID string, page int) (Page, error) {
	key := fmt.Sprintf("playlist|%s|%d", playlistID, page)
	v, err := s.do(key, func() (any, error) { return s.p.Playlist(ctx, playlistID, page) })
	return asPage(v), err
}

// Related returns videos related to a video (best-effort, keyless).
func (s *Service) Related(ctx context.Context, videoID string) (Page, error) {
	v, err := s.do("related|"+videoID, func() (any, error) { return s.p.Related(ctx, videoID) })
	return asPage(v), err
}

// RecommendRequest is the client-provided state for a stateless recommended feed.
type RecommendRequest struct {
	Seeds    []string `json:"seeds"`    // recently watched video ids (used to exclude)
	Channels []string `json:"channels"` // followed channel ids
	Limit    int      `json:"limit"`
}

// Recommended builds a stateless feed from followed channels' latest uploads,
// falling back to trending. The client owns its history/follows; FluxTube stores
// nothing — this mirrors the NewPipe/FreeTube model.
func (s *Service) Recommended(ctx context.Context, req RecommendRequest) (Page, error) {
	limit := req.Limit
	if limit <= 0 || limit > 60 {
		limit = 30
	}
	exclude := map[string]bool{}
	for _, id := range req.Seeds {
		exclude[id] = true
	}

	seen := map[string]bool{}
	var videos []VideoItem
	for _, ch := range req.Channels {
		page, err := s.ChannelVideos(ctx, ch, "videos", 1)
		if err != nil {
			continue
		}
		for _, v := range page.Videos {
			if exclude[v.ID] || seen[v.ID] {
				continue
			}
			seen[v.ID] = true
			videos = append(videos, v)
		}
	}

	// Interleave-ish: keep highest view counts first as a light ranking signal.
	sort.SliceStable(videos, func(i, j int) bool { return videos[i].Views > videos[j].Views })

	if len(videos) < limit {
		if tr, err := s.Trending(ctx, ""); err == nil {
			for _, v := range tr.Videos {
				if exclude[v.ID] || seen[v.ID] {
					continue
				}
				seen[v.ID] = true
				videos = append(videos, v)
			}
		}
	}
	if len(videos) > limit {
		videos = videos[:limit]
	}
	return Page{Videos: videos}, nil
}

func asPage(v any) Page {
	if p, ok := v.(Page); ok {
		return p
	}
	return Page{}
}

// NormalizeChannelID accepts a raw id or @handle and returns a channel path
// component usable in a channel URL.
func NormalizeChannelID(s string) string {
	s = strings.TrimSpace(s)
	return s
}
