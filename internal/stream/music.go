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

// audioPath returns the stored audio file for an id, if present.
func (e *Engine) audioPath(id string) (string, bool) {
	p := filepath.Join(e.MusicDir(), id+".m4a")
	if fileExists(p) {
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

	path := filepath.Join(e.MusicDir(), id+".m4a")

	// YouTube CDN URLs occasionally reject a request (transient 403/SABR). Retry
	// once with a freshly resolved URL before giving up.
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		res, err := e.ex.Resolve(ctx, id)
		if err != nil {
			return "", err
		}
		if err := e.writeAudio(ctx, res, path); err != nil {
			lastErr = err
			e.ex.Drop(id) // force a fresh resolve on the next attempt
			continue
		}
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

// ClearMusic deletes every stored music file and returns how many were removed.
func (e *Engine) ClearMusic() (int, error) {
	dir := e.MusicDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, en := range entries {
		if !en.IsDir() && filepath.Ext(en.Name()) == ".m4a" {
			if os.Remove(filepath.Join(dir, en.Name())) == nil {
				n++
			}
		}
	}
	e.accessMu.Lock()
	e.audioAccess = map[string]time.Time{}
	e.accessMu.Unlock()
	return n, nil
}

// AudioActive reports whether a music track was served recently enough to be
// considered currently playing.
func (e *Engine) AudioActive(id string) bool {
	e.accessMu.Lock()
	t, ok := e.audioAccess[id]
	e.accessMu.Unlock()
	return ok && time.Since(t) < audioActiveWindow
}
