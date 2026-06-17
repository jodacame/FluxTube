package extractor

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeID(t *testing.T) {
	cases := map[string]string{
		"dQw4w9WgXcQ": "dQw4w9WgXcQ",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=10s": "dQw4w9WgXcQ",
		"https://youtu.be/dQw4w9WgXcQ":                      "dQw4w9WgXcQ",
		"https://www.youtube.com/shorts/dQw4w9WgXcQ":        "dQw4w9WgXcQ",
		"https://www.youtube.com/embed/dQw4w9WgXcQ":         "dQw4w9WgXcQ",
	}
	for in, want := range cases {
		got, err := NormalizeID(in)
		if err != nil || got != want {
			t.Errorf("NormalizeID(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := NormalizeID("not a video"); err == nil {
		t.Error("expected error for invalid input")
	}
}

// adaptiveJSON simulates a full yt-dlp response with separate video/audio
// tracks plus subtitles, which restricted environments do not return.
const adaptiveJSON = `{
  "id":"abc12345678","title":"Sample","channel":"Chan","channel_id":"UC123","duration":100.4,
  "thumbnail":"https://t/x.jpg",
  "formats":[
    {"format_id":"sb0","protocol":"mhtml","vcodec":"none","acodec":"none"},
    {"format_id":"18","protocol":"https","ext":"mp4","vcodec":"avc1.42001E","acodec":"mp4a.40.2","height":360,"width":640,"tbr":500},
    {"format_id":"248","protocol":"https","ext":"webm","vcodec":"vp9","acodec":"none","height":1080,"width":1920,"fps":30,"vbr":2500,"dynamic_range":"SDR"},
    {"format_id":"701","protocol":"https","ext":"mp4","vcodec":"av01.0.12M.10","acodec":"none","height":2160,"width":3840,"fps":60,"vbr":12000,"dynamic_range":"HDR10"},
    {"format_id":"251","protocol":"https","ext":"webm","vcodec":"none","acodec":"opus","abr":160,"language":"en"},
    {"format_id":"251-1","protocol":"https","ext":"webm","vcodec":"none","acodec":"opus","abr":70,"language":"en"},
    {"format_id":"139","protocol":"https","ext":"m4a","vcodec":"none","acodec":"mp4a.40.5","abr":48,"language":"es"}
  ],
  "subtitles":{"en":[{"ext":"vtt","url":"https://www.youtube.com/api/timedtext?lang=en&fmt=srv1","name":"English"}]},
  "automatic_captions":{"en":[{"ext":"vtt","url":"https://x/auto-en"}],"fr":[{"ext":"vtt","url":"https://x/auto-fr"}]}
}`

func TestParseInfoAdaptive(t *testing.T) {
	var raw rawInfo
	if err := json.Unmarshal([]byte(adaptiveJSON), &raw); err != nil {
		t.Fatal(err)
	}
	res := parseInfo(&raw, map[string]bool{"en": true, "fr": true})

	if res.Duration != 100 {
		t.Errorf("duration = %d, want 100", res.Duration)
	}
	if len(res.Video) != 2 {
		t.Fatalf("video formats = %d, want 2", len(res.Video))
	}
	if res.Video[0].Height != 2160 || res.Video[0].Codec != "av1" || !res.Video[0].HDR {
		t.Errorf("top video unexpected: %+v", res.Video[0])
	}
	if len(res.Progressive) != 1 || res.Progressive[0].Height != 360 {
		t.Errorf("progressive unexpected: %+v", res.Progressive)
	}
	// Two languages, en deduped to the 160k track and marked default.
	if len(res.Audio) != 2 {
		t.Fatalf("audio tracks = %d, want 2", len(res.Audio))
	}
	if res.Audio[0].Lang != "en" || res.Audio[0].Bitrate != 160 || !res.Audio[0].Default {
		t.Errorf("default audio unexpected: %+v", res.Audio[0])
	}
	// en manual wins over en auto; fr auto added. Total 2 subs.
	if len(res.Subs) != 2 {
		t.Fatalf("subs = %d, want 2", len(res.Subs))
	}
	if res.Subs[0].Lang != "en" || res.Subs[0].Auto {
		t.Errorf("first sub should be manual en: %+v", res.Subs[0])
	}
	if !strings.Contains(res.Subs[0].URL, "fmt=vtt") {
		t.Errorf("subtitle url not forced to vtt: %s", res.Subs[0].URL)
	}

	// Best-track selection scans all (non-deduped) audio formats.
	if aac, ok := res.BestAAC(); !ok || aac.Codec != "aac" || aac.Bitrate != 48 {
		t.Errorf("BestAAC = %+v ok=%v, want aac 48k", aac, ok)
	}
	if best, ok := res.BestAudio(); !ok || best.Bitrate != 160 {
		t.Errorf("BestAudio = %+v ok=%v, want 160k", best, ok)
	}
}

func TestLanguages(t *testing.T) {
	r := &Resolved{Audio: []AudioTrack{{Lang: "en"}, {Lang: "es"}, {Lang: "en"}}}
	if got := r.Languages(); len(got) != 2 {
		t.Errorf("languages = %v, want 2 distinct", got)
	}
}
