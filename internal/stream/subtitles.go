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
	text := string(body)
	if !strings.HasPrefix(strings.TrimSpace(text), "WEBVTT") {
		text = "WEBVTT\n\n" + text
	}
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.WriteString(w, text)
	return nil
}
