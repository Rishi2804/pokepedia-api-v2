package main

import (
	"os"
	"path/filepath"
	"strings"
)

// diskCache is a flat key->file cache under a directory, one file per key.
// It exists so repeated scraper runs and parser iteration cost zero network
// requests once a page has been fetched -- important for Bulbapedia, whose
// Legends: Z-A coverage is still being filled in and which this tool will be
// run against more than once.
type diskCache struct {
	dir string
}

func newDiskCache(dir string) (*diskCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &diskCache{dir: dir}, nil
}

var unsafeFilenameChars = strings.NewReplacer(
	"/", "_", "\\", "_", ":", "_", "*", "_",
	"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
)

func (c *diskCache) path(key string) string {
	return filepath.Join(c.dir, unsafeFilenameChars.Replace(key))
}

func (c *diskCache) get(key string) ([]byte, bool) {
	b, err := os.ReadFile(c.path(key))
	if err != nil {
		return nil, false
	}
	return b, true
}

func (c *diskCache) put(key string, data []byte) error {
	return os.WriteFile(c.path(key), data, 0o644)
}
