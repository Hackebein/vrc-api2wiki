package vrchat

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"testing"
)

func TestNetscapeJarRoundTrip(t *testing.T) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse("https://api.vrchat.cloud/")
	jar.SetCookies(u, []*http.Cookie{{Name: "auth", Value: "abc", Path: "/", HttpOnly: true}})
	path := filepath.Join(t.TempDir(), "cookies.jar")
	if err := SaveNetscapeJar(jar, path); err != nil {
		t.Fatal(err)
	}
	jar2, _ := cookiejar.New(nil)
	if err := LoadNetscapeJar(jar2, path); err != nil {
		t.Fatal(err)
	}
	got := jar2.Cookies(u)
	if len(got) == 0 || got[0].Name != "auth" || got[0].Value != "abc" {
		t.Fatalf("got %#v", got)
	}
}
