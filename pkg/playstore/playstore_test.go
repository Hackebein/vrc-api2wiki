package playstore

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFetchVRChatPlayStoreBuildPicksHighest(t *testing.T) {
	const body = `
		<html><body>
		[[["2026.2.1p4-1839-94254b59aa-Release"]],[[[35]]]]
		[[["2026.2.3p3-1867-62fb4319cb-Release"]],[[[35]]]]
		[[["2026.2.3p1-1865-f71d38272d-Release"]],[[[35]]]]
		</body></html>
	`
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "play.google.com" {
			t.Fatalf("host %s", req.URL.Host)
		}
		if !strings.Contains(req.URL.RawQuery, "id="+VRChatPlayStorePackageID) {
			t.Fatalf("query %s", req.URL.RawQuery)
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
	cb, err := FetchVRChatPlayStoreBuild(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.2.3p3" || cb.BuildNumber != "1867" || cb.BuildHash != "62fb4319cb" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != PlayStoreClientName {
		t.Fatalf("branch %q", cb.Branch)
	}
}

func TestFetchVRChatPlayStoreBuildHTTPError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
		}, nil
	})}
	_, err := FetchVRChatPlayStoreBuild(client)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchVRChatPlayStoreBuildNoVersion(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader("<html>no builds here</html>")),
			Header:     make(http.Header),
		}, nil
	})}
	_, err := FetchVRChatPlayStoreBuild(client)
	if err == nil {
		t.Fatal("expected error")
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
