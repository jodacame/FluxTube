package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const pageSize = 30

// YtDlpProvider implements Provider using yt-dlp in flat-playlist mode, which is
// lightweight (ids and titles, no per-video resolve) and keyless.
type YtDlpProvider struct {
	Bin          string
	CookiesFile  string
	ExtractorArg string
	Timeout      time.Duration
	http         *http.Client
}

// NewYtDlpProvider creates a provider backed by yt-dlp.
func NewYtDlpProvider(bin, cookies, extractorArg string) *YtDlpProvider {
	if bin == "" {
		bin = "yt-dlp"
	}
	return &YtDlpProvider{
		Bin:          bin,
		CookiesFile:  cookies,
		ExtractorArg: extractorArg,
		Timeout:      45 * time.Second,
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

type flatThumb struct {
	URL string `json:"url"`
}

type flatEntry struct {
	ID         string      `json:"id"`
	Title      string      `json:"title"`
	Channel    string      `json:"channel"`
	ChannelID  string      `json:"channel_id"`
	Uploader   string      `json:"uploader"`
	Duration   float64     `json:"duration"`
	ViewCount  int64       `json:"view_count"`
	Thumbnails []flatThumb `json:"thumbnails"`
	IEKey      string      `json:"ie_key"`
}

type flatResult struct {
	Channel       string      `json:"channel"`
	ChannelID     string      `json:"channel_id"`
	Uploader      string      `json:"uploader"`
	Title         string      `json:"title"`
	Description   string      `json:"description"`
	FollowerCount int64       `json:"channel_follower_count"`
	Thumbnails    []flatThumb `json:"thumbnails"`
	Entries       []flatEntry `json:"entries"`
}

// run executes yt-dlp in flat-playlist mode against a target and parses output.
func (p *YtDlpProvider) run(ctx context.Context, target string, items string) (*flatResult, error) {
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()

	args := []string{"-J", "--flat-playlist", "--no-warnings", "--no-progress"}
	if items != "" {
		args = append(args, "--playlist-items", items)
	}
	if p.CookiesFile != "" {
		args = append(args, "--cookies", p.CookiesFile)
	}
	if p.ExtractorArg != "" {
		args = append(args, "--extractor-args", p.ExtractorArg)
	}
	args = append(args, "--", target)

	out, err := exec.CommandContext(ctx, p.Bin, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("discovery query failed")
	}
	var res flatResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("discovery parse error")
	}
	return &res, nil
}

func (p *YtDlpProvider) Search(ctx context.Context, query, kind string, limit int, music bool) (Page, error) {
	spec := fmt.Sprintf("ytsearch%d:%s", limit, query)
	res, err := p.run(ctx, spec, "")
	if err != nil {
		return Page{}, err
	}
	videos := toVideos(res.Entries)
	if music {
		videos = rankMusic(videos)
	}
	return Page{Videos: videos}, nil
}

// rankMusic reorders results to surface official song audio/video first, the
// way a music app would: artist topic channels and official uploads.
func rankMusic(videos []VideoItem) []VideoItem {
	score := func(v VideoItem) int {
		s := 0
		ch := strings.ToLower(v.Channel)
		title := strings.ToLower(v.Title)
		if strings.HasSuffix(ch, "- topic") || strings.Contains(ch, "vevo") {
			s += 3
		}
		if strings.Contains(title, "official audio") {
			s += 2
		}
		if strings.Contains(title, "official") || strings.Contains(title, "audio") {
			s++
		}
		return s
	}
	sort.SliceStable(videos, func(i, j int) bool { return score(videos[i]) > score(videos[j]) })
	return videos
}

func (p *YtDlpProvider) Trending(ctx context.Context, region string) (Page, error) {
	res, err := p.run(ctx, "https://www.youtube.com/feed/trending", fmt.Sprintf("1-%d", pageSize))
	if err != nil {
		return Page{}, err
	}
	return Page{Videos: toVideos(res.Entries)}, nil
}

func (p *YtDlpProvider) Channel(ctx context.Context, channelID string) (ChannelItem, error) {
	res, err := p.run(ctx, channelURL(channelID, "videos"), "1-1")
	if err != nil {
		return ChannelItem{}, err
	}
	return ChannelItem{
		ID:          firstNonEmpty(res.ChannelID, channelID),
		Name:        firstNonEmpty(res.Channel, res.Uploader, strings.TrimPrefix(res.Title, "")),
		Thumbnail:   pickThumb(res.Thumbnails),
		Subscribers: res.FollowerCount,
		Description: res.Description,
	}, nil
}

func (p *YtDlpProvider) ChannelVideos(ctx context.Context, channelID, tab string, page int) (Page, error) {
	if tab != "shorts" && tab != "streams" {
		tab = "videos"
	}
	res, err := p.run(ctx, channelURL(channelID, tab), pageRange(page))
	if err != nil {
		return Page{}, err
	}
	return Page{Videos: toVideos(res.Entries), NextPage: nextPage(page, len(res.Entries))}, nil
}

func (p *YtDlpProvider) Playlist(ctx context.Context, playlistID string, page int) (Page, error) {
	res, err := p.run(ctx, "https://www.youtube.com/playlist?list="+url.QueryEscape(playlistID), pageRange(page))
	if err != nil {
		return Page{}, err
	}
	return Page{Videos: toVideos(res.Entries), NextPage: nextPage(page, len(res.Entries))}, nil
}

// Related returns more videos from the same creator (a keyless best-effort,
// since YouTube's watch-next "related" requires the internal API).
func (p *YtDlpProvider) Related(ctx context.Context, videoID string) (Page, error) {
	channel := p.channelOf(ctx, videoID)
	if channel == "" {
		return Page{}, nil
	}
	page, err := p.ChannelVideos(ctx, channel, "videos", 1)
	if err != nil {
		return Page{}, err
	}
	out := page.Videos[:0]
	for _, v := range page.Videos {
		if v.ID != videoID {
			out = append(out, v)
		}
	}
	page.Videos = out
	page.NextPage = ""
	return page, nil
}

// channelOf resolves a video's channel id cheaply via oEmbed.
func (p *YtDlpProvider) channelOf(ctx context.Context, videoID string) string {
	endpoint := "https://www.youtube.com/oembed?format=json&url=" +
		url.QueryEscape("https://www.youtube.com/watch?v="+videoID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	resp, err := p.http.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var o struct {
		AuthorURL string `json:"author_url"`
	}
	if json.NewDecoder(resp.Body).Decode(&o) != nil {
		return ""
	}
	if i := strings.LastIndex(strings.TrimRight(o.AuthorURL, "/"), "/"); i >= 0 {
		return o.AuthorURL[i+1:]
	}
	return ""
}

// --- helpers ---

func toVideos(entries []flatEntry) []VideoItem {
	out := make([]VideoItem, 0, len(entries))
	for _, e := range entries {
		if e.ID == "" || len(e.ID) != 11 {
			continue // skip channel/playlist rows
		}
		out = append(out, VideoItem{
			ID:        e.ID,
			Title:     e.Title,
			Channel:   firstNonEmpty(e.Channel, e.Uploader),
			ChannelID: e.ChannelID,
			Thumbnail: "https://i.ytimg.com/vi/" + e.ID + "/hqdefault.jpg",
			Duration:  int(e.Duration + 0.5),
			Views:     e.ViewCount,
		})
	}
	return out
}

func channelURL(channelID, tab string) string {
	base := "https://www.youtube.com/"
	if strings.HasPrefix(channelID, "@") {
		base += channelID
	} else if strings.HasPrefix(channelID, "UC") {
		base += "channel/" + channelID
	} else {
		base += "@" + channelID
	}
	return base + "/" + tab
}

func pageRange(page int) string {
	if page < 1 {
		page = 1
	}
	start := (page-1)*pageSize + 1
	return fmt.Sprintf("%d-%d", start, page*pageSize)
}

func nextPage(page, got int) string {
	if got < pageSize {
		return ""
	}
	return fmt.Sprintf("%d", page+1)
}

func pickThumb(thumbs []flatThumb) string {
	if len(thumbs) == 0 {
		return ""
	}
	return thumbs[len(thumbs)-1].URL
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
