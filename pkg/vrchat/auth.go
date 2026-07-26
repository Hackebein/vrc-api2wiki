package vrchat

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultCookiePath = ".vrchat-session/cookies.jar"
	authRequestDelay  = 5 * time.Second
)

var apiBase = "https://api.vrchat.cloud/api/1"

type AuthConfig struct {
	Username   string
	Password   string
	TOTPSecret string
	CookiePath string
}

func AuthConfigFromEnv() (AuthConfig, bool) {
	cfg := AuthConfig{
		Username:   strings.TrimSpace(os.Getenv("VRCHAT_USERNAME")),
		Password:   strings.TrimSpace(os.Getenv("VRCHAT_PASSWORD")),
		TOTPSecret: strings.TrimSpace(os.Getenv("VRCHAT_TOTP_SECRET")),
		CookiePath: defaultCookiePath,
	}
	ok := cfg.Username != "" && cfg.Password != "" && cfg.TOTPSecret != ""
	return cfg, ok
}

func (c *Client) EnsureAuth(cfg AuthConfig, logger *slog.Logger) error {
	if cfg.CookiePath == "" {
		cfg.CookiePath = defaultCookiePath
	}
	if c.httpClient.Jar == nil {
		jar, err := NewCookieJar()
		if err != nil {
			return err
		}
		c.httpClient.Jar = jar
	}
	if err := LoadNetscapeJar(c.httpClient.Jar, cfg.CookiePath); err != nil {
		return fmt.Errorf("load cookie jar: %w", err)
	}

	user, err := c.fetchAuthUser("")
	if err == nil && userOK(user) {
		if logger != nil {
			logger.Info("vrchat session reused", "displayName", displayName(user))
		}
		return SaveNetscapeJar(c.httpClient.Jar, cfg.CookiePath)
	}

	user, err = c.login(cfg, logger)
	if err != nil {
		return err
	}
	if logger != nil {
		logger.Info("vrchat login ok", "displayName", displayName(user))
	}
	return SaveNetscapeJar(c.httpClient.Jar, cfg.CookiePath)
}

func displayName(user map[string]any) string {
	if v, ok := user["displayName"].(string); ok && v != "" {
		return v
	}
	if v, ok := user["username"].(string); ok {
		return v
	}
	return ""
}

func userOK(user map[string]any) bool {
	if user == nil {
		return false
	}
	if _, ok := user["error"]; ok {
		return false
	}
	if requires2FA(user) {
		return false
	}
	_, hasID := user["id"].(string)
	return hasID
}

func requires2FA(data map[string]any) bool {
	v, ok := data["requiresTwoFactorAuth"]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case []any:
		return len(t) > 0
	case []string:
		return len(t) > 0
	default:
		return true
	}
}

func (c *Client) fetchAuthUser(basicCreds string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, apiBase+"/auth/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	if basicCreds != "" {
		token := base64.StdEncoding.EncodeToString([]byte(basicCreds))
		req.Header.Set("Authorization", "Basic "+token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse auth/user HTTP %d: %w (%s)", resp.StatusCode, err, truncate(string(body), 200))
	}
	if resp.StatusCode != http.StatusOK && !requires2FA(data) {
		return data, fmt.Errorf("auth/user HTTP %d: %s", resp.StatusCode, truncate(string(body), 300))
	}
	return data, nil
}

func (c *Client) login(cfg AuthConfig, logger *slog.Logger) (map[string]any, error) {
	data, err := c.fetchAuthUser(cfg.Username + ":" + cfg.Password)
	if err != nil && !requires2FA(data) {
		return nil, fmt.Errorf("login: %w", err)
	}
	if userOK(data) {
		return data, nil
	}
	if !requires2FA(data) {
		return nil, fmt.Errorf("login failed: missing user and no 2FA challenge")
	}
	if logger != nil {
		logger.Info("vrchat 2FA required; verifying TOTP")
	}
	if err := c.verifyTOTP(cfg.TOTPSecret); err != nil {
		return nil, err
	}
	user, err := c.fetchAuthUser("")
	if err != nil {
		return nil, fmt.Errorf("auth/user after TOTP: %w", err)
	}
	if !userOK(user) {
		return nil, fmt.Errorf("auth/user after TOTP incomplete")
	}
	return user, nil
}

func (c *Client) verifyTOTP(secret string) error {
	var lastBody string
	for attempt := 0; attempt < 4; attempt++ {
		code, err := GenerateTOTP(secret)
		if err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]string{"code": code})
		req, err := http.NewRequest(http.MethodPost, apiBase+"/auth/twofactorauth/totp/verify", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastBody = string(body)
		if resp.StatusCode == http.StatusTooManyRequests {
			time.Sleep(time.Duration(30*(attempt+1)) * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("TOTP verify HTTP %d: %s", resp.StatusCode, truncate(lastBody, 300))
		}
		var verified map[string]any
		_ = json.Unmarshal(body, &verified)
		if v, ok := verified["verified"].(bool); ok && !v {
			return fmt.Errorf("TOTP not verified: %s", truncate(lastBody, 300))
		}
		return nil
	}
	return fmt.Errorf("TOTP verify rate-limited: %s", truncate(lastBody, 300))
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (c *Client) AuthedGet(rawURL string) ([]byte, error) {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if !c.lastAuthReq.IsZero() {
		if wait := authRequestDelay - time.Since(c.lastAuthReq); wait > 0 {
			time.Sleep(wait)
		}
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	c.lastAuthReq = time.Now()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d: %s", rawURL, resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}

func (c *Client) AuthedGetJSON(rawURL string, dest any) error {
	body, err := c.AuthedGet(rawURL)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("parse %s: %w", rawURL, err)
	}
	return nil
}

func (c *Client) getBytes(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
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
		return nil, fmt.Errorf("GET %s: HTTP %d: %s", rawURL, resp.StatusCode, truncate(string(body), 300))
	}
	return body, nil
}
