package discovery

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// mockProvider counts calls so we can assert caching behaviour.
type mockProvider struct {
	calls    int32
	channels map[string][]VideoItem
}

func (m *mockProvider) Search(ctx context.Context, q, k string, n int, music bool) (Page, error) {
	atomic.AddInt32(&m.calls, 1)
	return Page{Videos: []VideoItem{{ID: "vvvvvvvvvvv", Title: q}}}, nil
}
func (m *mockProvider) Trending(ctx context.Context, region string) (Page, error) {
	return Page{Videos: []VideoItem{{ID: "trendingaaa", Title: "t"}}}, nil
}
func (m *mockProvider) Channel(ctx context.Context, id string) (ChannelItem, error) {
	return ChannelItem{ID: id, Name: "Chan"}, nil
}
func (m *mockProvider) ChannelVideos(ctx context.Context, id, tab string, page int) (Page, error) {
	return Page{Videos: m.channels[id]}, nil
}
func (m *mockProvider) Playlist(ctx context.Context, id string, page int) (Page, error) {
	return Page{}, nil
}
func (m *mockProvider) Related(ctx context.Context, id string) (Page, error) { return Page{}, nil }

func TestServiceCaches(t *testing.T) {
	m := &mockProvider{}
	s := NewService(m, time.Minute)
	for i := 0; i < 5; i++ {
		if _, err := s.Search(context.Background(), "same", "", 10, false); err != nil {
			t.Fatal(err)
		}
	}
	if c := atomic.LoadInt32(&m.calls); c != 1 {
		t.Errorf("provider called %d times, want 1 (cached)", c)
	}
}

func TestRecommendedAggregates(t *testing.T) {
	m := &mockProvider{channels: map[string][]VideoItem{
		"c1": {{ID: "aaaaaaaaaaa", Views: 10}, {ID: "watched0001", Views: 99}},
		"c2": {{ID: "bbbbbbbbbbb", Views: 50}},
	}}
	s := NewService(m, time.Minute)
	page, err := s.Recommended(context.Background(), RecommendRequest{
		Channels: []string{"c1", "c2"},
		Seeds:    []string{"watched0001"}, // already watched, must be excluded
		Limit:    2,                       // exactly fills from channels, no trending fallback
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Videos) != 2 {
		t.Fatalf("got %d videos, want 2 (watched excluded)", len(page.Videos))
	}
	// Highest views first.
	if page.Videos[0].ID != "bbbbbbbbbbb" {
		t.Errorf("ranking wrong, first = %s", page.Videos[0].ID)
	}
	for _, v := range page.Videos {
		if v.ID == "watched0001" {
			t.Error("watched video should be excluded")
		}
	}
}
