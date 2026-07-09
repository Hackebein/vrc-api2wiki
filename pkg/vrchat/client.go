package vrchat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const worldsAPIBase = "https://api.vrchat.cloud/api/1/worlds"

var buildVersion = "dev"

func getUserAgent() string {
	v := strings.TrimSpace(buildVersion)
	if v == "" {
		v = "dev"
	}
	return fmt.Sprintf("VRC-API2Wiki/%s hackebein@gmail.com", v)
}

type Client struct {
	httpClient *http.Client
	userAgent  string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		httpClient: httpClient,
		userAgent:  getUserAgent(),
	}
}

func (c *Client) GetWorld(worldID string) (map[string]any, error) {
	url := worldsAPIBase + "/" + worldID
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("world %s: HTTP %d: %s", worldID, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var world map[string]any
	if err := json.Unmarshal(body, &world); err != nil {
		return nil, fmt.Errorf("parse world json: %w", err)
	}
	return world, nil
}
