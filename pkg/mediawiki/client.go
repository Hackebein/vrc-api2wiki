package mediawiki

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// requestDelay is applied before every wiki API request to stay gentle on the
// server.
const requestDelay = 100 * time.Millisecond

// rateLimitBackoffs is the escalating wait schedule when the wiki reports a
// rate limit (HTTP 429 or the "ratelimited" API error). After the last entry
// is exhausted the request fails.
var rateLimitBackoffs = []time.Duration{
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
	300 * time.Second,
}

type WikiConfig struct {
	URL       string
	Username  string
	Password  string
	Header    string
	HeaderVal string
}

type MediaWikiClient struct {
	apiURL     string
	httpClient *http.Client
	userAgent  string
	tokens     map[string]string
	mu         sync.RWMutex

	username string
	password string

	headerName  string
	headerValue string

	// offline: no credentials were provided. Reads still go live against the
	// wiki API; writes are diverted to local files in outputDir.
	offline   bool
	outputDir string

	logger *slog.Logger
}

var buildVersion = "dev"

func getUserAgent() string {
	v := strings.TrimSpace(buildVersion)
	if v == "" {
		v = "dev"
	}
	return fmt.Sprintf("VRC-API2Wiki/%s hackebein@gmail.com", v)
}

func NewMediaWikiClient(config WikiConfig, httpClient *http.Client) (*MediaWikiClient, error) {
	if httpClient == nil {
		jar, _ := cookiejar.New(nil)
		httpClient = &http.Client{Jar: jar}
	} else if httpClient.Jar == nil {
		jar, _ := cookiejar.New(nil)
		httpClient.Jar = jar
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c := &MediaWikiClient{
		apiURL:      config.URL,
		httpClient:  httpClient,
		userAgent:   getUserAgent(),
		tokens:      make(map[string]string),
		username:    strings.TrimSpace(config.Username),
		password:    strings.TrimSpace(config.Password),
		headerName:  strings.TrimSpace(config.Header),
		headerValue: strings.TrimSpace(config.HeaderVal),
		logger:      logger,
	}

	if c.username == "" && c.password == "" {
		c.offline = true
		c.outputDir = "./wiki-output"
		if c.logger != nil {
			c.logger.Info("offline mode enabled: live wiki reads without login, writes go to files", "dir", c.outputDir)
		}
	}

	if c.username != "" && c.password != "" {
		if err := c.Login(); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func SanitizeForWiki(text string) string {
	text = strings.ReplaceAll(text, "|", "{{!}}")
	text = strings.ReplaceAll(text, "=", "{{=}}")
	return text
}

func (c *MediaWikiClient) apiRequest(params map[string]string) (map[string]any, error) {
	params["format"] = "json"

	if c.headerName == "" && c.headerValue == "" {
		if hn, hv := os.Getenv("VRCWIKI_AUTHORIZATION_HEADER"), os.Getenv("VRCWIKI_AUTHORIZATION_VALUE"); hn != "" && hv != "" {
			c.headerName, c.headerValue = hn, hv
		}
	}

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	encoded := form.Encode()

	return c.doRequest(func() (*http.Request, error) {
		req, err := http.NewRequest(http.MethodPost, c.apiURL, strings.NewReader(encoded))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", c.userAgent)
		if c.headerName != "" && c.headerValue != "" {
			req.Header.Set(c.headerName, c.headerValue)
		}
		return req, nil
	})
}

// doRequest executes an HTTP request built by build, applying a per-request
// delay and retrying on rate-limit responses (HTTP 429 or the "ratelimited"
// API error) using the escalating rateLimitBackoffs schedule. The request is
// rebuilt for each attempt so the body can be re-sent.
func (c *MediaWikiClient) doRequest(build func() (*http.Request, error)) (map[string]any, error) {
	for attempt := 0; ; attempt++ {
		time.Sleep(requestDelay)

		req, err := build()
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("execute request: %w", err)
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if err := c.waitForRateLimit(attempt, "HTTP 429"); err != nil {
				return nil, err
			}
			continue
		}

		var result map[string]any
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse json: %w", err)
		}
		if e, ok := result["error"].(map[string]any); ok {
			code, _ := e["code"].(string)
			info, _ := e["info"].(string)
			if code == "ratelimited" {
				if err := c.waitForRateLimit(attempt, info); err != nil {
					return nil, err
				}
				continue
			}
			return nil, fmt.Errorf("API error: %s - %s", code, info)
		}
		return result, nil
	}
}

// waitForRateLimit sleeps for the backoff duration of the given attempt, or
// returns an error when the retry schedule is exhausted.
func (c *MediaWikiClient) waitForRateLimit(attempt int, detail string) error {
	if attempt >= len(rateLimitBackoffs) {
		return fmt.Errorf("rate limited (%s): exhausted retries after %d attempts", detail, len(rateLimitBackoffs))
	}
	wait := rateLimitBackoffs[attempt]
	if c.logger != nil {
		c.logger.Warn("rate limited, backing off",
			"detail", detail,
			"wait", wait.String(),
			"attempt", attempt+1,
			"max_attempts", len(rateLimitBackoffs),
		)
	}
	time.Sleep(wait)
	return nil
}

func (c *MediaWikiClient) getToken(tokenType string) (string, error) {
	c.mu.RLock()
	if t, ok := c.tokens[tokenType]; ok {
		c.mu.RUnlock()
		return t, nil
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.tokens[tokenType]; ok {
		return t, nil
	}
	params := map[string]string{"action": "query", "meta": "tokens", "type": tokenType}
	result, err := c.apiRequest(params)
	if err != nil {
		return "", fmt.Errorf("get %s token: %w", tokenType, err)
	}
	query, ok := result["query"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response: missing query")
	}
	tokens, ok := query["tokens"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response: missing tokens")
	}
	tokenKey := tokenType + "token"
	token, ok := tokens[tokenKey].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}
	c.tokens[tokenType] = token
	return token, nil
}

func (c *MediaWikiClient) invalidateToken(tokenType string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.tokens, tokenType)
}

func isBadTokenError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "badtoken")
}

func (c *MediaWikiClient) reloginIfPossible() error {
	if c.offline || c.username == "" || c.password == "" {
		return nil
	}
	c.invalidateToken("login")
	if err := c.Login(); err != nil {
		return fmt.Errorf("re-login after badtoken: %w", err)
	}
	return nil
}

func (c *MediaWikiClient) withCSRFWriteRetry(op func(csrf string) error) error {
	const maxAttempts = 2
	var lastErr error
	for range maxAttempts {
		csrf, err := c.getToken("csrf")
		if err != nil {
			return fmt.Errorf("get csrf: %w", err)
		}
		lastErr = op(csrf)
		if lastErr == nil {
			return nil
		}
		if !isBadTokenError(lastErr) {
			return lastErr
		}
		c.invalidateToken("csrf")
		if err := c.reloginIfPossible(); err != nil {
			return err
		}
	}
	return lastErr
}

func (c *MediaWikiClient) Login() error {
	loginToken, err := c.getToken("login")
	if err != nil {
		return fmt.Errorf("get login token: %w", err)
	}
	params := map[string]string{
		"action":     "login",
		"lgname":     c.username,
		"lgpassword": c.password,
		"lgtoken":    loginToken,
	}
	result, err := c.apiRequest(params)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	login, ok := result["login"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid login response structure")
	}
	if r, _ := login["result"].(string); r != "Success" {
		reason, _ := login["reason"].(string)
		if reason == "" {
			reason = "unknown"
		}
		return fmt.Errorf("login failed: %s", reason)
	}
	c.mu.Lock()
	c.tokens = make(map[string]string)
	c.mu.Unlock()
	if c.logger != nil {
		c.logger.Info("wiki login success")
	}
	return nil
}

func (c *MediaWikiClient) GetPageContent(title string) (string, error) {
	return c.getPageContent(title)
}

func (c *MediaWikiClient) getPageContent(title string) (string, error) {
	params := map[string]string{
		"action":  "query",
		"titles":  title,
		"prop":    "revisions",
		"rvprop":  "content",
		"rvslots": "main",
	}
	result, err := c.apiRequest(params)
	if err != nil {
		return "", fmt.Errorf("get page content for %s: %w", title, err)
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
			return "", fmt.Errorf("page does not exist: %s", title)
		}
		revisions, _ := pageMap["revisions"].([]any)
		if len(revisions) == 0 {
			return "", fmt.Errorf("no revisions found for page: %s", title)
		}
		rev, _ := revisions[0].(map[string]any)
		slots, _ := rev["slots"].(map[string]any)
		main, _ := slots["main"].(map[string]any)
		content, _ := main["*"].(string)
		return content, nil
	}
	return "", fmt.Errorf("could not extract content from page: %s", title)
}

func (c *MediaWikiClient) PageExists(title string) (bool, error) {
	_, err := c.getPageContent(title)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "page does not exist") {
		return false, nil
	}
	return false, err
}

func (c *MediaWikiClient) EditPage(title, text string, bot bool) error {
	trimmedNew := strings.TrimSpace(text)
	pageMissing := false
	currentContent, err := c.getPageContent(title)
	if err != nil {
		if !strings.Contains(err.Error(), "page does not exist") {
			return fmt.Errorf("get current content for page %s: %w", title, err)
		}
		pageMissing = true
	} else {
		if strings.TrimSpace(currentContent) == trimmedNew {
			if c.offline && c.logger != nil {
				c.logger.Info("offline: skip page (unchanged on wiki)", "title", title)
			}
			return nil
		}
	}

	summary := BuildEditSummary(title, trimmedNew)

	if c.offline {
		if err := os.MkdirAll(c.outputDir, 0o755); err != nil {
			return fmt.Errorf("ensure output dir: %w", err)
		}
		path := c.pageFilePath(title)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		if c.logger != nil {
			if pageMissing {
				c.logger.Info("offline: would create page", "title", title, "file", path, "bytes", len(text))
			} else {
				c.logger.Info("offline: would edit page (content changed)", "title", title, "file", path, "bytes", len(text))
			}
		}
		return nil
	}

	return c.withCSRFWriteRetry(func(csrf string) error {
		params := map[string]string{
			"action":  "edit",
			"title":   title,
			"text":    text,
			"summary": summary,
			"token":   csrf,
		}
		if bot {
			params["bot"] = "true"
		}
		result, err := c.apiRequest(params)
		if err != nil {
			return fmt.Errorf("edit request failed: %w", err)
		}
		edit, ok := result["edit"].(map[string]any)
		if !ok {
			return fmt.Errorf("invalid edit response structure")
		}
		if r, _ := edit["result"].(string); r != "Success" {
			return fmt.Errorf("edit failed: %s", r)
		}
		if c.logger != nil {
			c.logger.Info("wiki edit success", "title", title, "bot", bot)
		}
		return nil
	})
}

func (c *MediaWikiClient) pageFilePath(title string) string {
	dir := c.outputDir
	if strings.TrimSpace(dir) == "" {
		dir = "./wiki-output"
	}
	return dir + "/" + sanitizeFilename(title)
}

func sanitizeFilename(title string) string {
	title = strings.TrimSpace(title)
	var b strings.Builder
	for _, r := range title {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	s := b.String()
	var out strings.Builder
	prevUnderscore := false
	for _, r := range s {
		if r == '_' {
			if !prevUnderscore {
				out.WriteRune(r)
				prevUnderscore = true
			}
			continue
		}
		out.WriteRune(r)
		prevUnderscore = false
	}
	s = strings.Trim(out.String(), " _")
	if s == "" {
		s = "page"
	}
	return s + ".md"
}
