package vcc

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const (
	CreatorCompanionClientName     = "creator-companion"
	CreatorCompanionBetaClientName = "creator-companion-beta"
	QuickLauncherClientName        = "quick-launcher"

	newsURL              = "https://vcc.docs.vrchat.com/news/"
	githubReleasesURL    = "https://api.github.com/repos/vrchat-community/creator-companion/releases?per_page=20"
	portableZipName      = "web_vcc_%s_Release_Portable.zip"
	portableZipURL       = "https://github.com/vrchat-community/creator-companion/releases/download/%s/" + portableZipName
	quickLauncherZipPath = "Tools/VRC Quick Launcher/VRC Quick Launcher.exe"

	userAgent = "vrc-api2wiki (https://github.com/Hackebein/vrc-api2wiki)"
)

var (
	releaseTitlePattern = regexp.MustCompile(`(?i)Release\s+(v?\d+\.\d+\.\d+(?:[-.][A-Za-z0-9.]+)?)`)
	semverPattern       = regexp.MustCompile(`(?i)^v?\d+\.\d+\.\d+(?:[-.][A-Za-z0-9.]+)?$`)
)

func FetchCreatorCompanion(httpClient *http.Client) (*steam.ClientBuild, error) {
	body, err := getBytes(httpClient, newsURL, 60*time.Second)
	if err != nil {
		return nil, err
	}
	version, err := parseStableVersion(body)
	if err != nil {
		return nil, err
	}
	return toolBuild(version, CreatorCompanionClientName), nil
}

func FetchCreatorCompanionBeta(httpClient *http.Client) (*steam.ClientBuild, error) {
	body, err := getBytes(httpClient, githubReleasesURL, 60*time.Second)
	if err != nil {
		return nil, err
	}
	version, err := parseNewestPrereleaseTag(body)
	if err != nil {
		return nil, err
	}
	return toolBuild(version, CreatorCompanionBetaClientName), nil
}

func FetchQuickLauncher(httpClient *http.Client) (*steam.ClientBuild, error) {
	body, err := getBytes(httpClient, githubReleasesURL, 60*time.Second)
	if err != nil {
		return nil, err
	}
	tag, zipURL, err := parsePortableZipRelease(body)
	if err != nil {
		return nil, err
	}
	if zipURL == "" {
		zipURL = fmt.Sprintf(portableZipURL, tag, tag)
	}
	zipBody, err := getBytes(httpClient, zipURL, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("creator companion portable zip: %w", err)
	}
	version, err := quickLauncherVersionFromZip(zipBody)
	if err != nil {
		return nil, err
	}
	cb := toolBuild(version, QuickLauncherClientName)
	cb.RawMatch = tag
	return cb, nil
}

func toolBuild(version, name string) *steam.ClientBuild {
	return &steam.ClientBuild{
		Version:   version,
		Branch:    name,
		FetchedAt: time.Now().UTC(),
	}
}

func parseStableVersion(newsHTML []byte) (string, error) {
	matches := releaseTitlePattern.FindAllSubmatch(newsHTML, -1)
	for _, m := range matches {
		v := normalizeVersion(string(m[1]))
		if isPrereleaseVersion(v) {
			continue
		}
		return v, nil
	}
	return "", fmt.Errorf("vcc docs news: no stable Release version found")
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func parseNewestPrereleaseTag(body []byte) (string, error) {
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("github releases: %w", err)
	}
	for _, r := range releases {
		if !r.Prerelease {
			continue
		}
		v := normalizeVersion(r.TagName)
		if !semverPattern.MatchString(v) {
			continue
		}
		return v, nil
	}
	return "", fmt.Errorf("github releases: no prerelease found")
}

func parsePortableZipRelease(body []byte) (tag, zipURL string, err error) {
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", "", fmt.Errorf("github releases: %w", err)
	}
	wantSuffix := "_Release_Portable.zip"
	for _, r := range releases {
		v := normalizeVersion(r.TagName)
		if !semverPattern.MatchString(v) {
			continue
		}
		expected := fmt.Sprintf(portableZipName, v)
		for _, a := range r.Assets {
			if a.Name == expected || (strings.HasPrefix(a.Name, "web_vcc_") && strings.HasSuffix(a.Name, wantSuffix)) {
				return v, a.BrowserDownloadURL, nil
			}
		}
		// Prefer prereleases that ship the portable zip even when assets are omitted from the payload.
		if r.Prerelease {
			return v, "", nil
		}
	}
	return "", "", fmt.Errorf("github releases: no portable zip release found")
}

func quickLauncherVersionFromZip(zipBody []byte) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipBody), int64(len(zipBody)))
	if err != nil {
		return "", fmt.Errorf("portable zip: %w", err)
	}
	for _, f := range zr.File {
		if f.Name != quickLauncherZipPath {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		version, err := productVersionFromPE(data)
		if err != nil {
			return "", fmt.Errorf("quick launcher exe: %w", err)
		}
		return version, nil
	}
	return "", fmt.Errorf("portable zip: %q not found", quickLauncherZipPath)
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	return v
}

func isPrereleaseVersion(v string) bool {
	lower := strings.ToLower(v)
	return strings.Contains(lower, "beta") ||
		strings.Contains(lower, "alpha") ||
		strings.Contains(lower, "-rc") ||
		strings.Contains(lower, ".rc") ||
		strings.Contains(lower, "preview")
}

func getBytes(httpClient *http.Client, url string, timeout time.Duration) ([]byte, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	} else if httpClient.Timeout == 0 || httpClient.Timeout < timeout {
		// Copy client so callers with a short default timeout still allow large downloads.
		cloned := *httpClient
		cloned.Timeout = timeout
		httpClient = &cloned
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/html, */*")
	if strings.Contains(url, "api.github.com") {
		if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d: %s", url, resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
