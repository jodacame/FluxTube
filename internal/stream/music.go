package stream

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jodacame/fluxtube/internal/extractor"
)

// shardPath returns the on-disk location for a music file, sharded into a
// sub-directory by the id prefix so a single folder never holds thousands of
// entries (which slows directory listing on most filesystems).
func shardPath(dir, id string) string {
	sub := "_"
	if len(id) >= 2 {
		sub = id[:2]
	}
	return filepath.Join(dir, sub, id+".m4a")
}

// audioPath returns the stored audio file for an id, if present (checking the
// sharded location, then the legacy flat path for backward compatibility).
func (e *Engine) audioPath(id string) (string, bool) {
	dir := e.MusicDir()
	if p := shardPath(dir, id); fileExists(p) {
		return p, true
	}
	if p := filepath.Join(dir, id+".m4a"); fileExists(p) {
		return p, true
	}
	return "", false
}

// AudioFile returns the path to a persistent, universally-playable audio file
// for a video, producing it once if needed. The best AAC track is copied
// losslessly; if YouTube exposes no AAC, the best available audio is transcoded
// to AAC. The result is always AAC in an m4a container (faststart) so it plays
// in any player, and is stored persistently so a song is never fetched twice.
func (e *Engine) AudioFile(ctx context.Context, id string) (string, error) {
	if p, ok := e.audioPath(id); ok {
		return p, nil
	}

	e.audioMu.Lock()
	defer e.audioMu.Unlock()
	if p, ok := e.audioPath(id); ok {
		return p, nil
	}

	// Bound the whole preparation so a stalled fetch can never hold the lock
	// (and block other music) indefinitely — important for background auto-save.
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	path := shardPath(e.MusicDir(), id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	// YouTube CDN URLs occasionally reject a request (transient 403/SABR). Retry
	// a few times with a freshly resolved URL and a short backoff before giving up.
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			e.ex.Drop(id) // force a fresh resolve
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Second):
			}
		}
		res, err := e.ex.Resolve(ctx, id)
		if err != nil {
			lastErr = err
			continue
		}
		if err := e.writeAudio(ctx, res, path); err != nil {
			lastErr = err
			continue
		}
		e.invalidateStorage()
		return path, nil
	}
	return "", lastErr
}

// writeAudio copies the best audio track of a resolved video into an m4a file,
// preferring a lossless AAC copy and falling back to an AAC transcode.
func (e *Engine) writeAudio(ctx context.Context, res *extractor.Resolved, path string) error {
	src, copyCodec := res.BestAAC()
	if !copyCodec {
		best, ok := res.BestAudio()
		if !ok {
			return errors.New("no audio track available")
		}
		src = best
	}

	ua := src.UA
	if ua == "" {
		ua = e.opt.UserAgent
	}
	tmp := path + ".tmp"
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-user_agent", ua, "-i", src.URL, "-vn"}
	if copyCodec {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "192k")
	}
	args = append(args, "-movflags", "+faststart", "-f", "mp4", tmp)

	if err := exec.CommandContext(ctx, e.opt.FFmpegPath, args...).Run(); err != nil {
		_ = os.Remove(tmp)
		return errors.New("audio preparation failed")
	}
	return os.Rename(tmp, path)
}

// ServeAudio serves the persistent best-quality audio file with range support.
func (e *Engine) ServeAudio(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) error {
	path, err := e.AudioFile(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return nil
	}
	e.accessMu.Lock()
	e.audioAccess[id] = time.Now()
	e.accessMu.Unlock()
	w.Header().Set("Content-Type", "audio/mp4")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, path)
	return nil
}

// HasAudioFile reports whether a persistent audio file already exists.
func (e *Engine) HasAudioFile(id string) bool {
	_, ok := e.audioPath(id)
	return ok
}

// SetMusicDir updates the persistent music directory at runtime.
func (e *Engine) SetMusicDir(dir string) {
	if dir == "" {
		return
	}
	e.dirMu.Lock()
	e.opt.MusicDir = dir
	e.dirMu.Unlock()
	_ = os.MkdirAll(dir, 0o755)
}

// MusicDir returns the current persistent music directory. It uses a dedicated
// lock so it is safe to call while audioMu is held (avoids self-deadlock).
func (e *Engine) MusicDir() string {
	e.dirMu.RLock()
	defer e.dirMu.RUnlock()
	return e.opt.MusicDir
}

// ClearMusic deletes every stored music file (across all shards) and returns
// how many were removed.
func (e *Engine) ClearMusic() (int, error) {
	dir := e.MusicDir()
	n, _ := musicUsage(dir) // count before wiping
	if err := clearDir(dir); err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	e.accessMu.Lock()
	e.audioAccess = map[string]time.Time{}
	e.accessMu.Unlock()
	e.invalidateStorage()
	return n, nil
}

// invalidateStorage forces the next Storage call to recompute usage.
func (e *Engine) invalidateStorage() {
	e.storageMu.Lock()
	e.storageAt = time.Time{}
	e.storageMu.Unlock()
}

// AudioActive reports whether a music track was served recently enough to be
// considered currently playing.
func (e *Engine) AudioActive(id string) bool {
	e.accessMu.Lock()
	t, ok := e.audioAccess[id]
	e.accessMu.Unlock()
	return ok && time.Since(t) < audioActiveWindow
}
