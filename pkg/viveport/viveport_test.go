package viveport

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

const androidFixture = `{
  "products": {"ids": ["MVR-6008335b-2f70-4baf-8880-e097e0435284"], "total": 1},
  "contents": [{
    "id": "MVR-6008335b-2f70-4baf-8880-e097e0435284",
    "apps": [{
      "ver_code": 963880,
      "ver_name": "2026.2.3p1-1865-f71d38272d-Release"
    }]
  }]
}`

const androidBetaFixture = `{
  "products": {"ids": ["MVR-beta"], "total": 1},
  "contents": [{
    "id": "MVR-beta",
    "apps": [{
      "ver_code": 970000,
      "ver_name": "2026.3.1-1879-ecdbfcd219-Release"
    }]
  }]
}`

const windowsShortFixture = `{
  "products": {"ids": ["VR-a0888cce-68e8-4bfd-a9ee-4156261896ee"], "total": 1},
  "contents": [{
    "id": "VR-a0888cce-68e8-4bfd-a9ee-4156261896ee",
    "apps": [{
      "ver_code": 1782422217,
      "ver_name": "2026.2.3p1"
    }]
  }]
}`

func TestFetchAndroidBuildParsesFixture(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method %s", req.Method)
		}
		if req.URL.Path != androidCMSPath {
			t.Fatalf("path %s", req.URL.Path)
		}
		return jsonResponse(androidFixture), nil
	})}
	cb, err := FetchAndroidBuild(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.2.3p1" || cb.BuildNumber != "1865" || cb.BuildHash != "f71d38272d" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != AndroidClientName {
		t.Fatalf("branch %q", cb.Branch)
	}
	if cb.SteamBuildID != "MVR-6008335b-2f70-4baf-8880-e097e0435284" {
		t.Fatalf("id %q", cb.SteamBuildID)
	}
}

func TestFetchWindowsBuildIncompleteVersion(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != windowsCMSPath {
			t.Fatalf("path %s", req.URL.Path)
		}
		return jsonResponse(windowsShortFixture), nil
	})}
	_, err := FetchWindowsBuild(client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrIncompleteVersion) {
		t.Fatalf("got %v", err)
	}
}

func TestFetchAndroidOpenBetaBuild(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(req.Body)
		if !strings.Contains(string(body), `"include_unpublished":true`) {
			t.Fatalf("body %s", body)
		}
		return jsonResponse(androidBetaFixture), nil
	})}
	cb, err := FetchAndroidOpenBetaBuild(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.3.1" || cb.BuildNumber != "1879" || cb.BuildHash != "ecdbfcd219" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != AndroidOpenBetaClientName {
		t.Fatalf("branch %q", cb.Branch)
	}
}

func TestBuildsDiffer(t *testing.T) {
	live := &steam.ClientBuild{Version: "2026.2.3p1", BuildNumber: "1865", BuildHash: "f71d38272d"}
	same := &steam.ClientBuild{Version: "2026.2.3p1", BuildNumber: "1865", BuildHash: "f71d38272d"}
	diff := &steam.ClientBuild{Version: "2026.3.1", BuildNumber: "1879", BuildHash: "ecdbfcd219"}
	if BuildsDiffer(live, same) {
		t.Fatal("expected identical")
	}
	if !BuildsDiffer(live, diff) {
		t.Fatal("expected different")
	}
	if !BuildsDiffer(live, nil) {
		t.Fatal("nil should differ")
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
