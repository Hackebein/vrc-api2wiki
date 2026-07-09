package vrchat

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// ErrNotFound is returned by DownloadImage when the server answers 404.
var ErrNotFound = errors.New("image not found")

var contentTypeExtensions = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
	"image/gif":  "gif",
}

func imageExtension(contentType, rawURL string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		if ext, ok := contentTypeExtensions[mediaType]; ok {
			return ext, nil
		}
	}
	if u, err := url.Parse(rawURL); err == nil {
		ext := strings.TrimPrefix(strings.ToLower(path.Ext(u.Path)), ".")
		switch ext {
		case "png", "jpg", "jpeg", "webp", "gif":
			if ext == "jpeg" {
				ext = "jpg"
			}
			return ext, nil
		}
	}
	return "", fmt.Errorf("cannot determine image extension from content type %q or url %q", contentType, rawURL)
}

// DownloadImage fetches image bytes from a VRChat file URL (redirects to the
// CDN are followed by the HTTP client) and returns the bytes plus the file
// extension derived from the Content-Type header or URL path.
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
