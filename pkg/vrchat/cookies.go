package vrchat

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const netscapeHeader = "# Netscape HTTP Cookie File\n"

// LoadNetscapeJar loads cookies from a Netscape cookie file into jar.
func LoadNetscapeJar(jar http.CookieJar, path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	byHost := map[string][]*http.Cookie{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		httpOnly := false
		if strings.HasPrefix(line, "#HttpOnly_") {
			httpOnly = true
			line = strings.TrimPrefix(line, "#HttpOnly_")
		} else if strings.HasPrefix(strings.TrimSpace(line), "#") || strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}
		domain, pathPart, secureStr, expiresStr, name, value := parts[0], parts[2], parts[3], parts[4], parts[5], parts[6]
		secure := strings.EqualFold(secureStr, "TRUE")
		expires, _ := strconv.ParseInt(expiresStr, 10, 64)
		host := strings.TrimPrefix(domain, ".")
		c := &http.Cookie{
			Name:     name,
			Value:    value,
			Path:     pathPart,
			Domain:   host,
			Secure:   secure,
			HttpOnly: httpOnly,
		}
		if expires > 0 {
			c.Expires = time.Unix(expires, 0)
		}
		byHost[host] = append(byHost[host], c)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	for host, cookies := range byHost {
		u, err := url.Parse("https://" + host + "/")
		if err != nil {
			continue
		}
		jar.SetCookies(u, cookies)
	}
	return nil
}

// SaveNetscapeJar writes api.vrchat.cloud cookies (and vrchat.com mirrors) to path.
func SaveNetscapeJar(jar http.CookieJar, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	u, err := url.Parse("https://api.vrchat.cloud/")
	if err != nil {
		return err
	}
	cookies := jar.Cookies(u)
	var b strings.Builder
	b.WriteString(netscapeHeader)
	for _, c := range cookies {
		pathPart := c.Path
		if pathPart == "" {
			pathPart = "/"
		}
		secure := "FALSE"
		if c.Secure {
			secure = "TRUE"
		}
		expires := int64(0)
		if !c.Expires.IsZero() {
			expires = c.Expires.Unix()
		}
		fmt.Fprintf(&b, "#HttpOnly_api.vrchat.cloud\tFALSE\t%s\t%s\t%d\t%s\t%s\n",
			pathPart, secure, expires, c.Name, c.Value)
		fmt.Fprintf(&b, ".vrchat.com\tTRUE\t%s\t%s\t%d\t%s\t%s\n",
			pathPart, secure, expires, c.Name, c.Value)
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

// NewCookieJar returns an empty in-memory cookie jar.
func NewCookieJar() (http.CookieJar, error) {
	return cookiejar.New(nil)
}
