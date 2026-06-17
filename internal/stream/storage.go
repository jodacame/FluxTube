package stream

import (
	"os"
	"strings"
	"syscall"
)

// Storage reports disk usage of the music store and segment cache, plus the
// free/total space of the underlying filesystem.
type Storage struct {
	MusicBytes int64 `json:"musicBytes"`
	MusicCount int   `json:"musicCount"`
	CacheBytes int64 `json:"cacheBytes"`
	FreeBytes  int64 `json:"freeBytes"`
	TotalBytes int64 `json:"totalBytes"`
}

// Storage computes current storage usage.
func (e *Engine) Storage() Storage {
	dir := e.MusicDir()
	var s Storage
	if entries, err := os.ReadDir(dir); err == nil {
		for _, en := range entries {
			if en.IsDir() || !strings.HasSuffix(en.Name(), ".m4a") {
				continue
			}
			if info, err := en.Info(); err == nil {
				s.MusicBytes += info.Size()
				s.MusicCount++
			}
		}
	}
	s.CacheBytes = dirSize(e.opt.CacheRoot)

	var st syscall.Statfs_t
	if syscall.Statfs(dir, &st) == nil {
		s.FreeBytes = int64(st.Bavail) * int64(st.Bsize)
		s.TotalBytes = int64(st.Blocks) * int64(st.Bsize)
	}
	return s
}
