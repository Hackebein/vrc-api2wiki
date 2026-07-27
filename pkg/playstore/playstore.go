package playstore

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const (
	VRChatPlayStorePackageID = "com.vrchat.mobile.playstore"
	PlayStoreClientName      = "android-google-play"
	playStoreListingURL      = "https://play.google.com/store/apps/details?id=" + VRChatPlayStorePackageID + "&hl=en&gl=us"
	playStoreUserAgent       = "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"
)

func FetchVRChatPlayStoreBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	req, err := http.NewRequest(http.MethodGet, playStoreListingURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", playStoreUserAgent)
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
		return nil, fmt.Errorf("play store HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	cb, err := steam.ExtractHighestBuildFromBytes(body, PlayStoreClientName)
	if err != nil {
		return nil, fmt.Errorf("play store listing: %w", err)
	}
	return cb, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
