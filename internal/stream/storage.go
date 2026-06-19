package stream

import (
	"io/fs"
	"path/filepath"
	"syscall"
	"time"
)

// storageTTL bounds how often disk usage is recomputed, so frequent status-bar
// polls don't rescan a music store that may hold thousands of files.
const storageTTL = 30 * time.Second

// Storage reports disk usage of the music store plus the free/total space of
// the underlying filesystem.
type Storage struct {
	MusicBytes int64 `json:"musicBytes"`
	MusicCount int   `json:"musicCount"`
	FreeBytes  int64 `json:"freeBytes"`
	TotalBytes int64 `json:"totalBytes"`
}

// Storage returns current usage, cached for storageTTL.
func (e *Engine) Storage() Storage {
	e.storageMu.Lock()
	if time.Since(e.storageAt) < storageTTL {
		s := e.storageCache
		e.storageMu.Unlock()
		return s
	}
	e.storageMu.Unlock()

	dir := e.MusicDir()
	count, bytes := musicUsage(dir)
	s := Storage{MusicBytes: bytes, MusicCount: count}

	var st syscall.Statfs_t
	if syscall.Statfs(dir, &st) == nil {
		s.FreeBytes = int64(st.Bavail) * int64(st.Bsize)
		s.TotalBytes = int64(st.Blocks) * int64(st.Bsize)
	}

	e.storageMu.Lock()
	e.storageCache = s
	e.storageAt = time.Now()
	e.storageMu.Unlock()
	return s
}

// musicUsage walks the (sharded) music directory and totals the stored files.
func musicUsage(dir string) (count int, bytes int64) {
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(d.Name()) != ".m4a" {
			return nil
		}
		if info, err := d.Info(); err == nil {
			count++
			bytes += info.Size()
		}
		return nil
	})
	return count, bytes
}
