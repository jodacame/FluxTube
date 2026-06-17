package stream

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// audioExts are the containers a stored music track may use, by preference.
var audioExts = []string{"m4a", "opus"}

// audioContainer maps an audio codec to the lossless container we copy it into
// (no re-encoding), with the matching ffmpeg format and content type.
func audioContainer(codec string) (ext, format, ctype string) {
	switch codec {
	case "opus", "vorbis":
		return "opus", "ogg", "audio/ogg"
	default: // aac and anything else → mp4
		return "m4a", "mp4", "audio/mp4"
	}
}

func contentTypeForPath(p string) string {
	if strings.HasSuffix(p, ".opus") {
		return "audio/ogg"
	}
	return "audio/mp4"
}

// audioPath returns the stored audio file for an id, if present.
func (e *Engine) audioPath(id string) (string, bool) {
	dir := e.MusicDir()
	for _, ext := range audioExts {
		p := filepath.Join(dir, id+"."+ext)
		if fileExists(p) {
			return p, true
		}
	}
	return "", false
}

// AudioFile returns the path to a persistent, best-quality audio file for a
// video, producing it once if needed. The best available audio track is copied
// losslessly (no re-encoding) into its native container — AAC → m4a, Opus → ogg
// — so the saved file is always maximum quality. Stored persistently, a song is
// never downloaded twice.
func (e *Engine) AudioFile(ctx context.Context, id string) (string, error) {
	if p, ok := e.audioPath(id); ok {
		return p, nil
	}

	e.audioMu.Lock()
	defer e.audioMu.Unlock()
	if p, ok := e.audioPath(id); ok {
		return p, nil
	}

	res, err := e.ex.Resolve(ctx, id)
	if err != nil {
		return "", err
	}
	best, ok := res.BestAudio()
	if !ok {
		return "", errors.New("no audio track available")
	}

	ext, format, _ := audioContainer(best.Codec)
	path := filepath.Join(e.MusicDir(), id+"."+ext)
	tmp := path + ".tmp"

	args := []string{"-nostdin", "-hide_banner", "-loglevel", "error", "-y",
		"-user_agent", e.opt.UserAgent, "-i", best.URL, "-vn", "-c:a", "copy"}
	if format == "mp4" {
		args = append(args, "-movflags", "+faststart")
	}
	args = append(args, "-f", format, tmp)

	if err := exec.CommandContext(ctx, e.opt.FFmpegPath, args...).Run(); err != nil {
		_ = os.Remove(tmp)
		return "", errors.New("audio preparation failed")
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
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
	w.Header().Set("Content-Type", contentTypeForPath(path))
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

// AudioActive reports whether a music track was served recently enough to be
// considered currently playing.
func (e *Engine) AudioActive(id string) bool {
	e.accessMu.Lock()
	t, ok := e.audioAccess[id]
	e.accessMu.Unlock()
	return ok && time.Since(t) < audioActiveWindow
}
