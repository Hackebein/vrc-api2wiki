package mediawiki

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ImageCacheDirFromEnv() string {
	return strings.TrimSpace(os.Getenv("VRC_API2WIKI_IMAGE_CACHE"))
}

type diskImageCache struct {
	dir string
}

func openDiskImageCache(dir string) *diskImageCache {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	return &diskImageCache{dir: dir}
}

func (c *diskImageCache) filePath(sourceRef string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(sourceRef)))
	return filepath.Join(c.dir, hex.EncodeToString(sum[:])+".bin")
}

func (c *diskImageCache) Get(sourceRef string) (data []byte, sha1hex string, ok bool) {
	if c == nil || strings.TrimSpace(sourceRef) == "" {
		return nil, "", false
	}
	path := c.filePath(sourceRef)
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil, "", false
	}
	sum := sha1.Sum(data)
	return data, hex.EncodeToString(sum[:]), true
}

func (c *diskImageCache) Put(sourceRef string, data []byte) error {
	if c == nil || strings.TrimSpace(sourceRef) == "" {
		return nil
	}
	if len(data) == 0 {
		return fmt.Errorf("image cache: empty data for %s", sourceRef)
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("image cache mkdir: %w", err)
	}
	path := c.filePath(sourceRef)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("image cache write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("image cache rename: %w", err)
	}
	return nil
}
