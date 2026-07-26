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

func ClientConfigPlatform(clientName string) string {
	switch strings.TrimSpace(clientName) {
	case "public", "open-beta":
		return "PC"
	case "quest", "open-beta-quest":
		return "QuestStore"
	case "android-steamos":
		return "Default"
	default:
		return "Default"
	}
}

func (c *Client) GetMinSupportedClientBuildNumbers() (map[string]int, error) {
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
	var parsed struct {
		MinSupportedClientBuildNumber map[string]struct {
			MinBuildNumber int `json:"minBuildNumber"`
		} `json:"minSupportedClientBuildNumber"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if len(parsed.MinSupportedClientBuildNumber) == 0 {
		return nil, fmt.Errorf("config: missing minSupportedClientBuildNumber")
	}
	out := make(map[string]int, len(parsed.MinSupportedClientBuildNumber))
	for platform, info := range parsed.MinSupportedClientBuildNumber {
		out[platform] = info.MinBuildNumber
	}
	return out, nil
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
