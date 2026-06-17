// Package extractor resolves YouTube videos into stream sources and metadata.
//
// It is intentionally frugal with upstream requests: cheap metadata comes from
// oEmbed, the expensive full resolve is deferred until playback and then cached
// with a TTL, de-duplicated via single-flight, and guarded by a negative cache
// with backoff. Bytes are later fetched directly from the media CDN by the
// remuxer, so serving a stream does not trigger additional resolves.
package extractor

import "time"

// Meta is the lightweight catalog information for a video.
type Meta struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Channel     string `json:"channel"`
	ChannelID   string `json:"channelId"`
	Duration    int    `json:"duration"` // seconds
	Thumbnail   string `json:"thumbnail"`
	Description string `json:"description,omitempty"`
	Music       bool   `json:"music"` // auto-detected as a song / music track
}

// VideoFormat is a single video-only (or progressive) rendition.
type VideoFormat struct {
	ID       string `json:"id"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	FPS      int    `json:"fps"`
	Codec    string `json:"codec"`
	Ext      string `json:"ext"`
	Bitrate  int    `json:"bitrate"` // kbps
	HDR      bool   `json:"hdr"`
	HasAudio bool   `json:"hasAudio"` // true for progressive formats
	URL      string `json:"-"`        // CDN URL, never exposed to clients
	Label    string `json:"label"`    // e.g. "1080p60 HDR"
}

// AudioTrack is a single audio-only rendition for one language.
type AudioTrack struct {
	ID      string `json:"id"`
	Lang    string `json:"lang"`
	Name    string `json:"name"`
	Codec   string `json:"codec"`
	Ext     string `json:"ext"`
	Bitrate int    `json:"bitrate"` // kbps
	Default bool   `json:"default"`
	URL     string `json:"-"`
}

// SubTrack is a subtitle/caption track for one language.
type SubTrack struct {
	Lang string `json:"lang"`
	Name string `json:"name"`
	Auto bool   `json:"auto"` // automatically generated captions
	URL  string `json:"-"`    // WebVTT source URL
}

// Resolved is the full result of resolving a video for playback.
type Resolved struct {
	Meta
	Video       []VideoFormat `json:"video"`
	Audio       []AudioTrack  `json:"audio"`
	AllAudio    []AudioTrack  `json:"-"` // every audio format (not deduped); for best-track selection
	Subs        []SubTrack    `json:"subs"`
	Progressive []VideoFormat `json:"progressive"`
	ResolvedAt  time.Time     `json:"resolvedAt"`
	ExpiresAt   time.Time     `json:"expiresAt"`
}

// BestAAC returns the highest-bitrate AAC audio track (universally playable),
// or false if none is available.
func (r *Resolved) BestAAC() (AudioTrack, bool) {
	var best AudioTrack
	found := false
	for _, a := range r.AllAudio {
		if a.Codec == "aac" && (!found || a.Bitrate > best.Bitrate) {
			best, found = a, true
		}
	}
	return best, found
}

// BestAudio returns the highest-bitrate audio track regardless of codec.
func (r *Resolved) BestAudio() (AudioTrack, bool) {
	var best AudioTrack
	found := false
	for _, a := range r.AllAudio {
		if !found || a.Bitrate > best.Bitrate {
			best, found = a, true
		}
	}
	return best, found
}

// Languages returns the distinct audio languages available.
func (r *Resolved) Languages() []string {
	seen := map[string]bool{}
	var out []string
	for _, a := range r.Audio {
		if a.Lang != "" && !seen[a.Lang] {
			seen[a.Lang] = true
			out = append(out, a.Lang)
		}
	}
	return out
}
