package vrchat

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var fileIDPattern = regexp.MustCompile(`file_[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// FileIDFromURL extracts a VRChat file_… UUID from an API or CDN URL.
func FileIDFromURL(rawURL string) string {
	return fileIDPattern.FindString(rawURL)
}

// ErrNotFound is returned by DownloadImage when the server answers 404.
var ErrNotFound = errors.New("image not found")

var contentTypeExtensions = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
	"image/gif":  "gif",
}

var (
	pngMagic  = []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	jpegMagic = []byte{0xff, 0xd8, 0xff}
	gif87a    = []byte("GIF87a")
	gif89a    = []byte("GIF89a")
	riffMagic = []byte("RIFF")
	webpMagic = []byte("WEBP")
)

// NormalizeImageExt lowercases and maps "jpeg" → "jpg". Unknown values are
// returned unchanged (callers may still reject them).
func NormalizeImageExt(ext string) string {
	ext = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".")
	if ext == "jpeg" {
		return "jpg"
	}
	return ext
}

// ReplaceImageExt returns filename with its final extension replaced by ext.
// Prefer this when reusing a wiki filename but the real bytes need a different
// type (e.g. preferred "ListingImage Foo.png" for JPEG bytes → ".jpg").
func ReplaceImageExt(filename, ext string) string {
	filename = strings.TrimSpace(filename)
	ext = NormalizeImageExt(ext)
	if filename == "" {
		return filename
	}
	if ext == "" {
		return filename
	}
	if i := strings.LastIndex(filename, "."); i >= 0 && !strings.Contains(filename[i+1:], "/") {
		return filename[:i+1] + ext
	}
	return filename + "." + ext
}

// ImageExtensionFromBytes returns png/jpg/webp/gif based on magic bytes, or
// "" when the payload is not a recognized image. Prefer this over API metadata
// or Content-Type: MediaWiki verifies uploads the same way.
func ImageExtensionFromBytes(data []byte) string {
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], pngMagic):
		return "png"
	case len(data) >= 3 && bytes.Equal(data[:3], jpegMagic):
		return "jpg"
	case len(data) >= 6 && (bytes.Equal(data[:6], gif87a) || bytes.Equal(data[:6], gif89a)):
		return "gif"
	case len(data) >= 12 && bytes.Equal(data[:4], riffMagic) && bytes.Equal(data[8:12], webpMagic):
		return "webp"
	default:
		return ""
	}
}

// ResolveImageExt prefers magic-byte detection, then fallback (API/header/URL).
func ResolveImageExt(data []byte, fallback string) string {
	if ext := ImageExtensionFromBytes(data); ext != "" {
		return ext
	}
	if ext := NormalizeImageExt(fallback); ext == "png" || ext == "jpg" || ext == "webp" || ext == "gif" {
		return ext
	}
	return "png"
}

func imageExtension(contentType, rawURL string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		if ext, ok := contentTypeExtensions[mediaType]; ok {
			return ext, nil
		}
	}
	if u, err := url.Parse(rawURL); err == nil {
		ext := NormalizeImageExt(path.Ext(u.Path))
		switch ext {
		case "png", "jpg", "webp", "gif":
			return ext, nil
		}
	}
	return "", fmt.Errorf("cannot determine image extension from content type %q or url %q", contentType, rawURL)
}

// DownloadImage fetches image bytes from a VRChat file URL (redirects to the
// CDN are followed by the HTTP client) and returns the bytes plus the file
// extension. Magic bytes win over Content-Type and URL path so the wiki
// filename matches MediaWiki's MIME verification.
func (c *Client) DownloadImage(rawURL string) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("download image %s: %w", rawURL, ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("download image %s: HTTP %d", rawURL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read image body: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("download image %s: empty body", rawURL)
	}

	if ext := ImageExtensionFromBytes(data); ext != "" {
		return data, ext, nil
	}
	ext, err := imageExtension(resp.Header.Get("Content-Type"), rawURL)
	if err != nil {
		return nil, "", err
	}
	return data, ext, nil
}

// DownloadYouTubeThumbnail fetches the thumbnail for a YouTube video id,
// preferring maxresdefault and using hqdefault when the video has no
// max-resolution thumbnail (YouTube answers 404 in that case).
func (c *Client) DownloadYouTubeThumbnail(videoID string) ([]byte, string, error) {
	data, ext, err := c.DownloadImage(fmt.Sprintf("https://i.ytimg.com/vi/%s/maxresdefault.jpg", videoID))
	if errors.Is(err, ErrNotFound) {
		data, ext, err = c.DownloadImage(fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", videoID))
	}
	if err != nil {
		return nil, "", fmt.Errorf("youtube thumbnail for %s: %w", videoID, err)
	}
	return data, ext, nil
}
