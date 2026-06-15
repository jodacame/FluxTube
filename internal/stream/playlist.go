package stream

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jodacame/fluxtube/internal/extractor"
)

// pickVideo selects the best video format at or below maxH (0 = best). The
// list is expected sorted by height descending. If every format exceeds the
// cap, the smallest one is returned.
func pickVideo(list []extractor.VideoFormat, maxH int) extractor.VideoFormat {
	if len(list) == 0 {
		return extractor.VideoFormat{}
	}
	if maxH <= 0 {
		return list[0]
	}
	for _, v := range list {
		if v.Height <= maxH {
			return v
		}
	}
	return list[len(list)-1]
}

// audioTrackName returns the stable track identifier used in URLs for an audio
// track (its language, falling back to its format id).
func audioTrackName(a extractor.AudioTrack) string {
	if a.Lang != "" {
		return a.Lang
	}
	return a.ID
}

// buildMaster renders the multivariant master playlist. When separate audio
// tracks exist they are emitted as alternate renditions; otherwise the single
// video rendition (which carries embedded audio) is referenced directly.
func buildMaster(s *session) []byte {
	s.mu.Lock()
	video := s.video
	s.mu.Unlock()

	res := s.res
	separateAudio := len(res.Audio) > 0 && !video.HasAudio

	var b strings.Builder
	b.WriteString("#EXTM3U\n#EXT-X-VERSION:7\n")

	if separateAudio {
		for i, a := range res.Audio {
			name := audioTrackName(a)
			b.WriteString(fmt.Sprintf(
				"#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID=\"aud\",NAME=%q,LANGUAGE=%q,DEFAULT=%s,AUTOSELECT=YES,URI=%q\n",
				displayLang(a.Lang, a.Name), a.Lang, yesNo(i == 0 || a.Default), fmt.Sprintf("r/a-%s/index.m3u8", name)))
		}
	}
	for i, sub := range res.Subs {
		b.WriteString(fmt.Sprintf(
			"#EXT-X-MEDIA:TYPE=SUBTITLES,GROUP-ID=\"subs\",NAME=%q,LANGUAGE=%q,DEFAULT=NO,AUTOSELECT=%s,FORCED=NO,URI=%q\n",
			displayLang(sub.Lang, sub.Name), sub.Lang, yesNo(i == 0), fmt.Sprintf("subs/%s.m3u8", sub.Lang)))
	}

	inf := "#EXT-X-STREAM-INF:BANDWIDTH=" + strconv.Itoa(bandwidth(video, res))
	if video.Width > 0 && video.Height > 0 {
		inf += fmt.Sprintf(",RESOLUTION=%dx%d", video.Width, video.Height)
	}
	if codecs := codecString(video, res, separateAudio); codecs != "" {
		inf += fmt.Sprintf(",CODECS=%q", codecs)
	}
	if separateAudio {
		inf += ",AUDIO=\"aud\""
	}
	if len(res.Subs) > 0 {
		inf += ",SUBTITLES=\"subs\""
	}
	b.WriteString(inf + "\n")
	b.WriteString("r/video/index.m3u8\n")
	return []byte(b.String())
}

// buildSubtitlePlaylist returns a single-segment VOD playlist for a WebVTT track.
func buildSubtitlePlaylist(lang string, duration int) []byte {
	if duration <= 0 {
		duration = 1
	}
	return []byte(fmt.Sprintf(
		"#EXTM3U\n#EXT-X-VERSION:7\n#EXT-X-TARGETDURATION:%d\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXTINF:%d.0,\n../sub/%s.vtt\n#EXT-X-ENDLIST\n",
		duration, duration, lang))
}

func bandwidth(video extractor.VideoFormat, res *extractor.Resolved) int {
	bw := video.Bitrate * 1000
	if len(res.Audio) > 0 {
		bw += res.Audio[0].Bitrate * 1000
	}
	if bw <= 0 {
		bw = 1_000_000
	}
	return bw
}

// codecString produces a best-effort RFC 6381 codecs attribute.
func codecString(video extractor.VideoFormat, res *extractor.Resolved, separateAudio bool) string {
	var parts []string
	switch video.Codec {
	case "h264":
		parts = append(parts, "avc1.640028")
	case "vp9":
		parts = append(parts, "vp09.00.50.08")
	case "av1":
		parts = append(parts, "av01.0.08M.08")
	}
	if separateAudio && len(res.Audio) > 0 {
		switch res.Audio[0].Codec {
		case "opus":
			parts = append(parts, "opus")
		case "aac":
			parts = append(parts, "mp4a.40.2")
		}
	} else if video.HasAudio {
		parts = append(parts, "mp4a.40.2")
	}
	return strings.Join(parts, ",")
}

// playlistHas reports whether the rendition playlist already lists a segment,
// which means ffmpeg has finished writing it.
func playlistHas(playlistPath, seg string) bool {
	data, err := os.ReadFile(playlistPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), seg)
}

func displayLang(lang, name string) string {
	if name != "" {
		return name
	}
	if lang != "" {
		return lang
	}
	return "default"
}

func yesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}
