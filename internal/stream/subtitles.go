package stream

import (
	"context"
	"io"
	"net/http"
	"strings"
)

// serveVTT fetches a subtitle source and serves it as WebVTT. YouTube's
// timedtext endpoint returns valid WebVTT when fmt=vtt is requested; if the
// payload lacks the required header we prepend it defensively.
func serveVTT(ctx context.Context, w http.ResponseWriter, ua, src string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		http.Error(w, "bad subtitle source", http.StatusBadGateway)
		return nil
	}
	req.Header.Set("User-Agent", ua)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "subtitle upstream error", http.StatusBadGateway)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "subtitle unavailable", http.StatusBadGateway)
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		http.Error(w, "subtitle read error", http.StatusBadGateway)
		return nil
	}
	text := normalizeVTT(string(body))
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.WriteString(w, text)
	return nil
}

// normalizeVTT ensures a WEBVTT header and strips per-cue positioning settings
// (align/position/line/size) that YouTube adds, which otherwise push captions
// off-centre. Without those settings players default to bottom-centre.
func normalizeVTT(text string) string {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "-->"); idx >= 0 {
			// Keep "<start> --> <end>" and drop any trailing cue settings.
			start := strings.TrimSpace(line[:idx])
			rest := strings.TrimSpace(line[idx+3:])
			fields := strings.Fields(rest)
			end := ""
			if len(fields) > 0 {
				end = fields[0]
			}
			lines[i] = start + " --> " + end
		}
	}
	out := strings.Join(lines, "\n")
	if !strings.HasPrefix(strings.TrimSpace(out), "WEBVTT") {
		out = "WEBVTT\n\n" + out
	}
	return out
}
