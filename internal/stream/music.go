package stream

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// AudioFile returns the path to a persistent, universally-playable audio file
// for a video, producing it once if needed. The result is AAC in an MP4 (m4a)
// container with a front-loaded index (faststart) so any player can stream and
// seek it. Because it is stored persistently, a song is never downloaded twice.
func (e *Engine) AudioFile(ctx context.Context, id string) (string, error) {
	path := filepath.Join(e.opt.MusicDir, id+".m4a")
	if fileExists(path) {
		return path, nil
	}

	// Serialise preparation; re-check after acquiring the lock.
	e.audioMu.Lock()
	defer e.audioMu.Unlock()
	if fileExists(path) {
		return path, nil
	}

	res, err := e.ex.Resolve(ctx, id)
	if err != nil {
		return "", err
	}
	// Prefer the highest-bitrate AAC track (lossless copy, universally playable);
	// only if no AAC exists, transcode the best available audio to AAC.
	src, copyCodec := res.BestAAC()
	if !copyCodec {
		best, ok := res.BestAudio()
		if !ok {
			return "", errors.New("no audio track available")
		}
		src = best
	}

	tmp := path + ".tmp"
	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-user_agent", e.opt.UserAgent, "-i", src.URL, "-vn"}
	if copyCodec {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-c:a", "aac", "-b:a", "192k")
	}
	args = append(args, "-movflags", "+faststart", "-f", "mp4", tmp)

	if err := exec.CommandContext(ctx, e.opt.FFmpegPath, args...).Run(); err != nil {
		_ = os.Remove(tmp)
		return "", errors.New("audio preparation failed")
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

// ServeAudio serves the persistent audio file with range support.
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
	return fileExists(filepath.Join(e.opt.MusicDir, id+".m4a"))
}

// AudioActive reports whether a music track was served recently enough to be
// considered currently playing.
func (e *Engine) AudioActive(id string) bool {
	e.accessMu.Lock()
	t, ok := e.audioAccess[id]
	e.accessMu.Unlock()
	return ok && time.Since(t) < audioActiveWindow
}

// SetMusicDir updates the persistent music directory at runtime.
func (e *Engine) SetMusicDir(dir string) {
	if dir == "" {
		return
	}
	e.audioMu.Lock()
	e.opt.MusicDir = dir
	e.audioMu.Unlock()
	_ = os.MkdirAll(dir, 0o755)
}

// MusicDir returns the current persistent music directory.
func (e *Engine) MusicDir() string {
	e.audioMu.Lock()
	defer e.audioMu.Unlock()
	return e.opt.MusicDir
}
