package stream

import (
	"context"
	"io"
	"net/http"
)

// proxyProgressive streams a progressive (muxed) source through, forwarding the
// client's Range header so seeking works natively, and passing the upstream
// status, content-type, length and range headers back.
func proxyProgressive(ctx context.Context, w http.ResponseWriter, r *http.Request, ua, src string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		http.Error(w, "bad source", http.StatusBadGateway)
		return nil
	}
	req.Header.Set("User-Agent", ua)
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return nil
	}
	defer resp.Body.Close()

	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	if w.Header().Get("Accept-Ranges") == "" {
		w.Header().Set("Accept-Ranges", "bytes")
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
	return nil
}
