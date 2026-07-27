package vrchat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const UnitySDKClientName = "unity-sdk"

func ClientConfigPlatform(clientName string) string {
	switch strings.TrimSpace(clientName) {
	case "windows", "open-beta-windows":
		return "PC"
	case "android-quest", "open-beta-android-quest":
		return "QuestStore"
	case "android-pico":
		return "PicoStore"
	case "android-google-play":
		return "GooglePlay"
	case "android-viveport", "open-beta-android-viveport":
		return "XRElite"
	case "windows-viveport", "open-beta-windows-viveport":
		return "PC"
	case "android-steamos":
		return "Default"
	default:
		return "Default"
	}
}

type apiConfig struct {
	MinSupportedClientBuildNumber map[string]struct {
		MinBuildNumber int `json:"minBuildNumber"`
	} `json:"minSupportedClientBuildNumber"`
	SdkUnityVersion string `json:"sdkUnityVersion"`
}

func (c *Client) fetchAPIConfig() (*apiConfig, error) {
	req, err := http.NewRequest(http.MethodGet, apiBase+"/config", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	var parsed apiConfig
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &parsed, nil
}

func minBuildMap(parsed *apiConfig) (map[string]int, error) {
	if len(parsed.MinSupportedClientBuildNumber) == 0 {
		return nil, fmt.Errorf("config: missing minSupportedClientBuildNumber")
	}
	out := make(map[string]int, len(parsed.MinSupportedClientBuildNumber))
	for platform, info := range parsed.MinSupportedClientBuildNumber {
		out[platform] = info.MinBuildNumber
	}
	return out, nil
}

func sdkUnityVersion(parsed *apiConfig) (string, error) {
	v := strings.TrimSpace(parsed.SdkUnityVersion)
	if v == "" {
		return "", fmt.Errorf("config: missing sdkUnityVersion")
	}
	return v, nil
}

func (c *Client) GetMinSupportedClientBuildNumbers() (map[string]int, error) {
	parsed, err := c.fetchAPIConfig()
	if err != nil {
		return nil, err
	}
	return minBuildMap(parsed)
}

func (c *Client) GetSdkUnityVersion() (string, error) {
	parsed, err := c.fetchAPIConfig()
	if err != nil {
		return "", err
	}
	return sdkUnityVersion(parsed)
}

func (c *Client) GetClientBuildConfig() (mins map[string]int, sdkUnity string, err error) {
	parsed, err := c.fetchAPIConfig()
	if err != nil {
		return nil, "", err
	}
	mins, err = minBuildMap(parsed)
	if err != nil {
		return nil, "", err
	}
	sdkUnity, err = sdkUnityVersion(parsed)
	if err != nil {
		return nil, "", err
	}
	return mins, sdkUnity, nil
}

func MinBuildForClient(mins map[string]int, clientName string) (int, error) {
	platform := ClientConfigPlatform(clientName)
	if n, ok := mins[platform]; ok {
		return n, nil
	}
	if n, ok := mins["Default"]; ok {
		return n, nil
	}
	return 0, fmt.Errorf("config: no min build for platform %q (client %q)", platform, clientName)
}

func ClearClientBuildIfBelowMin(b *steam.ClientBuild, min int) bool {
	if b == nil {
		return false
	}
	n, err := strconv.Atoi(strings.TrimSpace(b.BuildNumber))
	if err == nil && n >= min {
		return false
	}
	b.Version = ""
	b.BuildNumber = ""
	b.BuildHash = ""
	return true
}
