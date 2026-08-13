package pico

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const (
	VRChatPicoAppID         = "7288745304105664518"
	PicoClientName          = "android-pico"
	picoItemInfoURL         = "https://appstore-us.picoxr.com/api/app/v1/item/info"
	picoStoreUserAgent      = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	picoDeviceName          = "A8110"
	picoAppLanguage         = "en"
	picoManifestVersionCode = "401200000"
	picoClientType          = "3"
)

type itemInfoRequest struct {
	ItemID int64 `json:"item_id"`
}

type itemInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Detail struct {
			AppVersion string `json:"app_version"`
		} `json:"detail"`
	} `json:"data"`
}

func FetchVRChatPicoBuild(httpClient *http.Client) (*steam.ClientBuild, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	itemID, err := strconv.ParseInt(VRChatPicoAppID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("pico item id: %w", err)
	}
	reqBody, err := json.Marshal(itemInfoRequest{ItemID: itemID})
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(picoItemInfoURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("device_name", picoDeviceName)
	q.Set("app_language", picoAppLanguage)
	q.Set("manifest_version_code", picoManifestVersionCode)
	q.Set("client_type", picoClientType)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", picoStoreUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://store-global.picoxr.com/")

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
	var parsed itemInfoResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse pico store item info: %w", err)
	}
	if parsed.Code != 0 {
		msg := strings.TrimSpace(parsed.Msg)
		if msg == "" {
			msg = fmt.Sprintf("code %d", parsed.Code)
		}
		return "", fmt.Errorf("pico store item info: %s", msg)
	}
	version := strings.TrimSpace(parsed.Data.Detail.AppVersion)
	if version == "" {
		return "", fmt.Errorf("pico store item info: app_version not found")
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
