package viveport

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const (
	VRChatAndroidAppID = "4b5187b8-004c-473c-9219-258c3ea8e255"
	VRChatWindowsAppID = "469fbcbb-bfde-40b5-a7d4-381249d387cd"

	AndroidClientName         = "android-viveport"
	AndroidOpenBetaClientName = "open-beta-android-viveport"
	WindowsClientName         = "windows-viveport"
	WindowsOpenBetaClientName = "open-beta-windows-viveport"

	cmsBaseURL        = "https://www.viveport.com"
	androidCMSPath    = "/api/cms/v4/mobiles/a"
	windowsCMSPath    = "/api/cms/v4/products/a/all"
	viveportUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
)

// ErrIncompleteVersion is returned when Viveport only exposes a short version
// string that cannot be parsed into a full ClientBuild.
var ErrIncompleteVersion = errors.New("viveport: incomplete version string")

type cmsRequest struct {
	AppIDs             []string `json:"app_ids"`
	Locale             string   `json:"locale"`
	Country            string   `json:"cnty"`
	ShowComingSoon     bool     `json:"show_coming_soon"`
	ContentGenus       string   `json:"content_genus"`
	SubscriptionOnly   int      `json:"subscription_only"`
	IncludeUnpublished bool     `json:"include_unpublished"`
}

type cmsResponse struct {
	Contents []cmsContent `json:"contents"`
}

type cmsContent struct {
	ID   string   `json:"id"`
	Apps []cmsApp `json:"apps"`
}

type cmsApp struct {
	VerName string `json:"ver_name"`
	VerCode int    `json:"ver_code"`
}

func FetchAndroidBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	return fetchApp(httpClient, androidCMSPath, VRChatAndroidAppID, AndroidClientName, false)
}

func FetchAndroidOpenBetaBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	return fetchApp(httpClient, androidCMSPath, VRChatAndroidAppID, AndroidOpenBetaClientName, true)
}

func FetchWindowsBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	return fetchApp(httpClient, windowsCMSPath, VRChatWindowsAppID, WindowsClientName, false)
}

func FetchWindowsOpenBetaBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	return fetchApp(httpClient, windowsCMSPath, VRChatWindowsAppID, WindowsOpenBetaClientName, true)
}

func BuildsDiffer(a, b *steam.ClientBuild) bool {
	if a == nil || b == nil {
		return a != b
	}
	return strings.TrimSpace(a.Version) != strings.TrimSpace(b.Version) ||
		strings.TrimSpace(a.BuildNumber) != strings.TrimSpace(b.BuildNumber) ||
		strings.TrimSpace(a.BuildHash) != strings.TrimSpace(b.BuildHash)
}

func fetchApp(httpClient *http.Client, path, appID, clientName string, includeUnpublished bool) (*steam.ClientBuild, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	reqBody, err := json.Marshal(cmsRequest{
		AppIDs:             []string{appID},
		Locale:             "en_US",
		Country:            "US",
		ShowComingSoon:     true,
		ContentGenus:       "all",
		SubscriptionOnly:   1,
		IncludeUnpublished: includeUnpublished,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, cmsBaseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", viveportUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://www.viveport.com/")

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
		return nil, fmt.Errorf("viveport cms HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var parsed cmsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse viveport cms: %w", err)
	}
	if len(parsed.Contents) == 0 {
		return nil, fmt.Errorf("viveport cms: no contents for app %s", appID)
	}
	content := parsed.Contents[0]
	if len(content.Apps) == 0 {
		return nil, fmt.Errorf("viveport cms: no apps for content %s", content.ID)
	}
	verName := strings.TrimSpace(content.Apps[0].VerName)
	if verName == "" {
		return nil, fmt.Errorf("viveport cms: empty ver_name for app %s", appID)
	}

	cb, err := steam.ExtractBuildFromBytes([]byte(verName), clientName)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrIncompleteVersion, verName)
	}
	cb.SteamBuildID = content.ID
	return cb, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
