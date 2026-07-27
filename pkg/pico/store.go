package pico

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const (
	VRChatPicoAppID    = "7288745304105664518"
	PicoClientName     = "android-pico"
	picoStoreDetailURL = "https://store-global.picoxr.com/global/detail/1/" + VRChatPicoAppID
	picoStoreUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

var appVersionPattern = regexp.MustCompile(`"app_version"\s*:\s*"([^"]+)"`)

func FetchVRChatPicoBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, picoStoreDetailURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", picoStoreUserAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

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
		return nil, fmt.Errorf("pico store HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	version, err := extractAppVersion(body)
	if err != nil {
		return nil, err
	}
	cb, err := steam.ExtractBuildFromBytes([]byte(version), PicoClientName)
	if err != nil {
		return nil, fmt.Errorf("parse pico app_version %q: %w", version, err)
	}
	cb.SteamBuildID = VRChatPicoAppID
	return cb, nil
}

func extractAppVersion(body []byte) (string, error) {
	m := appVersionPattern.FindSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("pico store listing: app_version not found")
	}
	version := strings.TrimSpace(string(m[1]))
	if version == "" {
		return "", fmt.Errorf("pico store listing: empty app_version")
	}
	return version, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
