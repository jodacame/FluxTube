package stream

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// clearDir removes all entries inside a directory without removing the dir.
func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		_ = os.RemoveAll(filepath.Join(dir, e.Name()))
	}
	return nil
}

func itoa(n int) string { return strconv.Itoa(n) }

// dirSize returns the total size in bytes of all files under dir.
func dirSize(dir string) int64 {
	var total int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// safeName guards path components against traversal and unexpected separators.
func safeName(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	if strings.ContainsAny(s, "/\\") || strings.Contains(s, "..") {
		return false
	}
	return true
}
