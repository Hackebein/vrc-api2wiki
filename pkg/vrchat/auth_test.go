package vrchat

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnsureAuthLoginWithTOTP(t *testing.T) {
	var step int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/1/auth/user" && r.Header.Get("Authorization") == "" && step == 0:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"no cookie"}}`))
		case r.URL.Path == "/api/1/auth/user" && strings.HasPrefix(r.Header.Get("Authorization"), "Basic ") && step == 0:
			step = 1
			http.SetCookie(w, &http.Cookie{Name: "auth", Value: "authcookie", Path: "/"})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"requiresTwoFactorAuth": []string{"totp"},
			})
		case r.URL.Path == "/api/1/auth/twofactorauth/totp/verify":
			step = 2
			http.SetCookie(w, &http.Cookie{Name: "twoFactorAuth", Value: "tfacookie", Path: "/"})
			_ = json.NewEncoder(w).Encode(map[string]any{"verified": true})
		case r.URL.Path == "/api/1/auth/user" && step >= 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":          "usr_test",
				"displayName": "Test User",
			})
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unexpected"}`))
		}
	}))
	defer srv.Close()

	oldBase := apiBase
	apiBase = srv.URL + "/api/1"
	defer func() { apiBase = oldBase }()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(&http.Client{Jar: jar, Timeout: 10 * time.Second})
	dir := t.TempDir()
	cfg := AuthConfig{
		Username:   "user@example.com",
		Password:   "secret",
		TOTPSecret: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ",
		CookiePath: filepath.Join(dir, "cookies.jar"),
	}
	if err := client.EnsureAuth(cfg, nil); err != nil {
		t.Fatal(err)
	}
}

func TestUserOK(t *testing.T) {
	if userOK(nil) {
		t.Fatal("nil should be false")
	}
	if userOK(map[string]any{"requiresTwoFactorAuth": []any{"totp"}}) {
		t.Fatal("2FA challenge should be false")
	}
	if !userOK(map[string]any{"id": "usr_x", "displayName": "X"}) {
		t.Fatal("valid user")
	}
}
