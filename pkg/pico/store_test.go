package pico

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFetchVRChatPicoBuildParsesAppVersion(t *testing.T) {
	const body = `
		<html><body>
		<script>
		window.__DETAIL__ = {"detail":{"app_version":"2026.2.3p1-1865-f71d38272d-Release","version_code":968210}};
		</script>
		<div>Version<span>2026.2.3p1-1865-f71d38272d-Release (06/25/2026)</span></div>
		</body></html>
	`
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "store-global.picoxr.com" {
			t.Fatalf("host %s", req.URL.Host)
		}
		if !strings.Contains(req.URL.Path, VRChatPicoAppID) {
			t.Fatalf("path %s", req.URL.Path)
		}
		if req.Header.Get("User-Agent") == "" {
			t.Fatal("missing User-Agent")
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	cb, err := FetchVRChatPicoBuild(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.2.3p1" || cb.BuildNumber != "1865" || cb.BuildHash != "f71d38272d" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != PicoClientName {
		t.Fatalf("branch %q", cb.Branch)
	}
	if cb.SteamBuildID != VRChatPicoAppID {
		t.Fatalf("id %q", cb.SteamBuildID)
	}
}

func TestFetchVRChatPicoBuildHTTPError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
		}, nil
	})}
	_, err := FetchVRChatPicoBuild(client)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchVRChatPicoBuildNoAppVersion(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("<html>no app_version here</html>")),
			Header:     make(http.Header),
		}, nil
	})}
	_, err := FetchVRChatPicoBuild(client)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestExtractAppVersion(t *testing.T) {
	v, err := extractAppVersion([]byte(`{"app_version":"2026.2.3p1-1865-f71d38272d-Release"}`))
	if err != nil {
		t.Fatal(err)
	}
	if v != "2026.2.3p1-1865-f71d38272d-Release" {
		t.Fatalf("%q", v)
	}
	if _, err := extractAppVersion([]byte(`{}`)); err == nil {
		t.Fatal("expected error")
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
