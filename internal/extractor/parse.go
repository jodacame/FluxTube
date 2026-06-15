package extractor

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// rawInfo mirrors the subset of yt-dlp's JSON we consume.
type rawInfo struct {
	ID                string                   `json:"id"`
	Title             string                   `json:"title"`
	Channel           string                   `json:"channel"`
	ChannelID         string                   `json:"channel_id"`
	Uploader          string                   `json:"uploader"`
	Duration          float64                  `json:"duration"`
	Thumbnail         string                   `json:"thumbnail"`
	Description       string                   `json:"description"`
	Formats           []rawFormat              `json:"formats"`
	Subtitles         map[string][]rawSubtitle `json:"subtitles"`
	AutomaticCaptions map[string][]rawSubtitle `json:"automatic_captions"`
}

type rawFormat struct {
	FormatID     string  `json:"format_id"`
	URL          string  `json:"url"`
	Ext          string  `json:"ext"`
	Protocol     string  `json:"protocol"`
	VCodec       string  `json:"vcodec"`
	ACodec       string  `json:"acodec"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	FPS          float64 `json:"fps"`
	TBR          float64 `json:"tbr"`
	ABR          float64 `json:"abr"`
	VBR          float64 `json:"vbr"`
	Language     string  `json:"language"`
	FormatNote   string  `json:"format_note"`
	DynamicRange string  `json:"dynamic_range"`
}

type rawSubtitle struct {
	Ext  string `json:"ext"`
	URL  string `json:"url"`
	Name string `json:"name"`
}

func has(codec string) bool { return codec != "" && codec != "none" }

// parseInfo converts raw yt-dlp info into a Resolved result. allowedAuto limits
// which automatically generated caption languages are exposed.
func parseInfo(r *rawInfo, allowedAuto map[string]bool) *Resolved {
	res := &Resolved{
		Meta: Meta{
			ID:          r.ID,
			Title:       r.Title,
			Channel:     firstNonEmpty(r.Channel, r.Uploader),
			ChannelID:   r.ChannelID,
			Duration:    int(r.Duration + 0.5),
			Thumbnail:   r.Thumbnail,
			Description: r.Description,
		},
	}

	for _, f := range r.Formats {
		// Skip storyboards and non-progressive streaming protocols; we remux
		// from direct (range-capable) https sources.
		if f.Protocol != "https" && f.Protocol != "http" {
			continue
		}
		switch {
		case has(f.VCodec) && has(f.ACodec):
			res.Progressive = append(res.Progressive, VideoFormat{
				ID: f.FormatID, Width: f.Width, Height: f.Height,
				FPS: int(f.FPS + 0.5), Codec: shortCodec(f.VCodec), Ext: f.Ext,
				Bitrate: int(f.TBR + 0.5), HasAudio: true, URL: f.URL,
				Label: videoLabel(f),
			})
		case has(f.VCodec):
			res.Video = append(res.Video, VideoFormat{
				ID: f.FormatID, Width: f.Width, Height: f.Height,
				FPS: int(f.FPS + 0.5), Codec: shortCodec(f.VCodec), Ext: f.Ext,
				Bitrate: bitrate(f.VBR, f.TBR), HDR: isHDR(f.DynamicRange),
				URL: f.URL, Label: videoLabel(f),
			})
		case has(f.ACodec):
			res.Audio = append(res.Audio, AudioTrack{
				ID: f.FormatID, Lang: f.Language, Name: audioName(f),
				Codec: shortCodec(f.ACodec), Ext: f.Ext,
				Bitrate: bitrate(f.ABR, f.TBR), URL: f.URL,
			})
		}
	}

	dedupeAudioByLang(res)
	res.Subs = collectSubs(r, allowedAuto)

	// Highest quality first.
	sort.SliceStable(res.Video, func(i, j int) bool { return res.Video[i].Height > res.Video[j].Height })
	sort.SliceStable(res.Progressive, func(i, j int) bool { return res.Progressive[i].Height > res.Progressive[j].Height })
	sort.SliceStable(res.Audio, func(i, j int) bool { return res.Audio[i].Bitrate > res.Audio[j].Bitrate })
	return res
}

// dedupeAudioByLang keeps the highest-bitrate audio per language and marks a
// default track.
func dedupeAudioByLang(res *Resolved) {
	best := map[string]AudioTrack{}
	var order []string
	for _, a := range res.Audio {
		key := a.Lang
		if cur, ok := best[key]; !ok || a.Bitrate > cur.Bitrate {
			if !ok {
				order = append(order, key)
			}
			best[key] = a
		}
	}
	var out []AudioTrack
	for i, k := range order {
		t := best[k]
		if i == 0 {
			t.Default = true
		}
		out = append(out, t)
	}
	res.Audio = out
}

// collectSubs merges manual and automatic captions, preferring WebVTT and
// avoiding duplicate languages (manual wins over automatic). Manual subtitles
// are always kept; automatic captions are limited to allowedAuto (plus the
// channel's original-language track) to avoid exposing ~150 languages.
func collectSubs(r *rawInfo, allowedAuto map[string]bool) []SubTrack {
	seen := map[string]bool{}
	var out []SubTrack
	add := func(lang string, list []rawSubtitle, auto bool) {
		if seen[lang] || len(list) == 0 {
			return
		}
		s := pickSubtitle(list)
		if s.URL == "" {
			return
		}
		seen[lang] = true
		out = append(out, SubTrack{Lang: lang, Name: subName(s, lang), Auto: auto, URL: vttURL(s.URL)})
	}
	for lang, list := range r.Subtitles {
		add(lang, list, false)
	}
	for lang, list := range r.AutomaticCaptions {
		if !allowedAuto[lang] && !allowedAuto[baseLang(lang)] && !strings.HasSuffix(lang, "-orig") {
			continue
		}
		add(lang, list, true)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Auto != out[j].Auto {
			return !out[i].Auto // manual first
		}
		return out[i].Lang < out[j].Lang
	})
	return out
}

// pickSubtitle prefers a vtt entry, otherwise the first usable one.
func pickSubtitle(list []rawSubtitle) rawSubtitle {
	for _, s := range list {
		if s.Ext == "vtt" {
			return s
		}
	}
	return list[0]
}

// vttURL forces the timedtext format to WebVTT so players can render it.
func vttURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	if q.Has("fmt") || strings.Contains(u.Host, "youtube.com") {
		q.Set("fmt", "vtt")
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func bitrate(primary, fallback float64) int {
	if primary > 0 {
		return int(primary + 0.5)
	}
	return int(fallback + 0.5)
}

func isHDR(dr string) bool {
	dr = strings.ToUpper(dr)
	return dr != "" && dr != "SDR"
}

func shortCodec(c string) string {
	c = strings.ToLower(c)
	switch {
	case strings.HasPrefix(c, "avc1"), strings.HasPrefix(c, "h264"):
		return "h264"
	case strings.HasPrefix(c, "vp9"), strings.HasPrefix(c, "vp09"):
		return "vp9"
	case strings.HasPrefix(c, "av01"):
		return "av1"
	case strings.HasPrefix(c, "mp4a"):
		return "aac"
	case strings.HasPrefix(c, "opus"):
		return "opus"
	}
	if i := strings.IndexByte(c, '.'); i > 0 {
		return c[:i]
	}
	return c
}

func videoLabel(f rawFormat) string {
	label := fmt.Sprintf("%dp", f.Height)
	if f.FPS >= 50 {
		label += fmt.Sprintf("%d", int(f.FPS+0.5))
	}
	if isHDR(f.DynamicRange) {
		label += " HDR"
	}
	return label
}

func audioName(f rawFormat) string {
	if f.Language != "" {
		return f.Language
	}
	if f.FormatNote != "" {
		return f.FormatNote
	}
	return "audio"
}

func subName(s rawSubtitle, lang string) string {
	if s.Name != "" {
		return s.Name
	}
	return lang
}

// baseLang returns the primary subtag of a language code (e.g. "es" for "es-419").
func baseLang(lang string) string {
	if i := strings.IndexAny(lang, "-_"); i > 0 {
		return lang[:i]
	}
	return lang
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
