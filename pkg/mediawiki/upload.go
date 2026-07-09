package mediawiki

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// apiUploadRequest performs a multipart/form-data POST against the MediaWiki
// API, reusing the same User-Agent, CF-bypass header, and cookie jar as
// apiRequest.
func (c *MediaWikiClient) apiUploadRequest(params map[string]string, filename string, file []byte) (map[string]any, error) {
	params["format"] = "json"

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range params {
		if err := writer.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("write form field %s: %w", k, err)
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(file); err != nil {
		return nil, fmt.Errorf("write file data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}
	contentType := writer.FormDataContentType()
	payload := buf.Bytes()

	return c.doRequest(func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.apiURL, bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("User-Agent", c.userAgent)
		if c.headerName != "" && c.headerValue != "" {
			req.Header.Set(c.headerName, c.headerValue)
		}
		return req, nil
	})
}

// GetFileSHA1 returns the SHA1 hex digest of the current revision of
// File:<filename> on the wiki, or "" if the file does not exist.
func (c *MediaWikiClient) GetFileSHA1(filename string) (string, error) {
	params := map[string]string{
		"action": "query",
		"titles": "File:" + filename,
		"prop":   "imageinfo",
		"iiprop": "sha1",
	}
	result, err := c.apiRequest(params)
	if err != nil {
		return "", fmt.Errorf("get imageinfo for %s: %w", filename, err)
	}
	query, ok := result["query"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response structure: missing query")
	}
	pages, ok := query["pages"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response structure: missing pages")
	}
	for _, page := range pages {
		pageMap, _ := page.(map[string]any)
		if pageMap == nil {
			continue
		}
		if _, missing := pageMap["missing"]; missing {
			return "", nil
		}
		imageinfo, _ := pageMap["imageinfo"].([]any)
		if len(imageinfo) == 0 {
			return "", nil
		}
		info, _ := imageinfo[0].(map[string]any)
		sha, _ := info["sha1"].(string)
		return sha, nil
	}
	return "", nil
}

// UploadFile uploads image bytes as File:<filename>, skipping the upload when
// the wiki already has a file with an identical SHA1. Returns true when an
// upload was performed.
func (c *MediaWikiClient) UploadFile(filename string, data []byte) (bool, error) {
	existingSHA1, err := c.GetFileSHA1(filename)
	if err != nil {
		return false, fmt.Errorf("check existing file %s: %w", filename, err)
	}
	sum := sha1.Sum(data)
	newSHA1 := hex.EncodeToString(sum[:])
	if existingSHA1 != "" && strings.EqualFold(existingSHA1, newSHA1) {
		if c.offline && c.logger != nil {
			c.logger.Info("offline: skip file (sha1 unchanged on wiki)", "filename", filename)
		}
		return false, nil
	}

	if c.offline {
		if err := os.MkdirAll(c.outputDir, 0o755); err != nil {
			return false, fmt.Errorf("ensure output dir: %w", err)
		}
		path := c.imageFilePath(filename)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return false, fmt.Errorf("write image file: %w", err)
		}
		if c.logger != nil {
			if existingSHA1 == "" {
				c.logger.Info("offline: would upload new file", "filename", filename, "file", path, "bytes", len(data))
			} else {
				c.logger.Info("offline: would upload file (sha1 mismatch)", "filename", filename, "file", path, "bytes", len(data))
			}
		}
		return true, nil
	}

	err = c.withCSRFWriteRetry(func(csrf string) error {
		params := map[string]string{
			"action":         "upload",
			"filename":       filename,
			"comment":        BuildEditSummary("File:"+filename, ""),
			"token":          csrf,
			"ignorewarnings": "true",
		}
		result, err := c.apiUploadRequest(params, filename, data)
		if err != nil {
			return fmt.Errorf("upload request failed: %w", err)
		}
		upload, ok := result["upload"].(map[string]any)
		if !ok {
			return fmt.Errorf("invalid upload response structure")
		}
		if r, _ := upload["result"].(string); r != "Success" {
			return fmt.Errorf("upload failed: %s", r)
		}
		if c.logger != nil {
			c.logger.Info("wiki upload success", "filename", filename, "bytes", len(data))
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// imageFilePath returns the offline output path for an uploaded image,
// keeping the real file extension (unlike page files which get .md).
func (c *MediaWikiClient) imageFilePath(filename string) string {
	dir := c.outputDir
	if strings.TrimSpace(dir) == "" {
		dir = "./wiki-output"
	}
	name := sanitizeFilename(filename)
	name = strings.TrimSuffix(name, ".md")
	return filepath.Join(dir, name)
}
