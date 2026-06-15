package stream

import (
	"fmt"
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

// segName formats the on-disk/segment filename for a global segment index.
func segName(n int) string { return fmt.Sprintf("seg%05d.m4s", n) }

// parseSegName extracts the segment index from a segment filename.
func parseSegName(file string) (int, bool) {
	s := strings.TrimSuffix(strings.TrimPrefix(file, "seg"), ".m4s")
	if s == file || s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// maxSegIndex returns the highest segment index currently present in dir, or -1.
func maxSegIndex(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return -1
	}
	max := -1
	for _, e := range entries {
		if n, ok := parseSegName(e.Name()); ok && n > max {
			max = n
		}
	}
	return max
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
