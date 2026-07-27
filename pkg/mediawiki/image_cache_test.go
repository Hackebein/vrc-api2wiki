package mediawiki

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"testing"
)

func TestDiskImageCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := openDiskImageCache(dir)
	ref := "file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@2"
	payload := []byte("fake-image-bytes")
	if err := c.Put(ref, payload); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, sha, ok := c.Get(ref)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("bytes mismatch")
	}
	sum := sha1.Sum(payload)
	if sha != hex.EncodeToString(sum[:]) {
		t.Fatalf("sha %q", sha)
	}
	if !strings.HasPrefix(c.filePath(ref), dir) {
		t.Fatalf("path outside cache dir: %s", c.filePath(ref))
	}
	if _, _, ok := c.Get("file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@3"); ok {
		t.Fatal("version bump must miss cache")
	}
}

func TestDiskImageCacheDisabled(t *testing.T) {
	var c *diskImageCache
	if err := c.Put("x@1", []byte("a")); err != nil {
		t.Fatalf("nil Put: %v", err)
	}
	if _, _, ok := c.Get("x@1"); ok {
		t.Fatal("nil Get should miss")
	}
}
