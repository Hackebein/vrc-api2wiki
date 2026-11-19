package vrchat

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type releaseTestTransport func(*http.Request) (*http.Response, error)

func (f releaseTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func releaseResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		Request: req, StatusCode: status, Header: http.Header{"Content-Type": {"text/plain"}},
		Body: io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchAppStoreRelease(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := &http.Client{Transport: releaseTestTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host != "itunes.apple.com" || req.URL.Path != "/lookup" || req.URL.Query().Get("id") != "6448578770" {
					t.Fatalf("unexpected lookup %s", req.URL)
				}
				version := "2026.4.1p9"
				if req.URL.Query().Get("country") == "de" {
					version = "2026.4.1p10"
					if status != http.StatusOK {
						return releaseResponse(req, status, ""), nil
					}
				}
				body := fmt.Sprintf(`{"results":[{"trackId":1,"version":"2030.1.1"},{"trackId":6448578770,"bundleId":"com.vrchat.mobile","version":%q}]}`, version)
				return releaseResponse(req, http.StatusOK, body), nil
			})}
			listing, err := FetchAppStoreRelease(client)
			if status != http.StatusOK {
				if err == nil {
					t.Fatal("one available region must not replace the published result")
				}
			} else if err != nil || listing.Version != "2026.4.1p10" || !strings.HasSuffix(listing.LookupURL, "country=de") {
				t.Fatalf("got %+v, %v", listing, err)
			}
		})
	}
}

const testReleaseIndex = `# VRChat
- [VRChat 2026.4.1](https://docs.vrchat.com/docs/vrchat-202641.md): October 28 - Build 1936 - Live
  - [VRChat 2026.4.1p2](https://docs.vrchat.com/docs/vrchat-202641p2.md): Patch!
  - [VRChat 2026.4.1p3 (Steam Only)](https://docs.vrchat.com/docs/steam-only.md): Steam update
  - [VRChat 2026.4.1p3](https://docs.vrchat.com/docs/vrchat-202641p3.md): November 16 - Build 1939 - Live
  - [VRChat 2026.4.1 (OPEN BETA)](https://docs.vrchat.com/docs/vrchat-202641-open-beta.md): October 9 - Build 1931 - Open Beta
  - [VRChat 2026.4.1p4 (OPEN BETA)](https://docs.vrchat.com/docs/vrchat-202641p4-open-beta.md): Open Beta
`

func testReleaseNotesClient(t *testing.T) *http.Client {
	t.Helper()
	return &http.Client{Transport: releaseTestTransport(func(req *http.Request) (*http.Response, error) {
		var body string
		switch req.URL.String() {
		case LatestLiveReleaseURL:
			resp := releaseResponse(req, http.StatusFound, "")
			resp.Header.Set("Location", "/docs/vrchat-202641p3")
			return resp, nil
		case "https://docs.vrchat.com/docs/vrchat-202641p3":
			body = "<html>Release notes</html>"
		case releaseIndexURL:
			body = testReleaseIndex
		case "https://docs.vrchat.com/docs/vrchat-202641.md":
			body = "---\nupdatedAt: 2026-11-02T00:00:00Z\n---\n\n# VRChat 2026.4.1\n\nOctober 28 - Build 1936 - Live\n"
		case "https://docs.vrchat.com/docs/vrchat-202641p2.md":
			body = "# VRChat 2026.4.1p2\n\nPatch! The client is `2026.4.1p2` build `1938`.\n"
		case "https://docs.vrchat.com/docs/vrchat-202641p3.md":
			body = "# VRChat 2026.4.1p3\n\nNovember 16 - Build 1939 - Live\n"
		default:
			t.Fatalf("unexpected release notes request %s", req.URL)
		}
		return releaseResponse(req, http.StatusOK, body), nil
	})}
}

func TestFetchLiveMainReleaseKeepsOriginalDateAndBuild(t *testing.T) {
	notes, err := FetchLiveMainRelease(testReleaseNotesClient(t))
	want := ClientReleaseNotes{
		Version: "2026.4.1", BuildNumber: "1936", ReleaseDate: "2026-10-28",
		URL: "https://docs.vrchat.com/docs/vrchat-202641",
	}
	if err != nil || notes != want {
		t.Fatalf("got %+v, %v, want %+v", notes, err, want)
	}
}

func TestFetchIOSReleaseNotesMatchesExactLiveBuild(t *testing.T) {
	client := testReleaseNotesClient(t)
	for version, expected := range map[string]string{"2026.4.1p2": "1938", "2026.4.1p3": "1939"} {
		notes, err := FetchIOSReleaseNotes(client, version)
		if err != nil || notes.Version != version || notes.BuildNumber != expected {
			t.Fatalf("got %+v, %v, want %s build %s", notes, err, version, expected)
		}
	}
	if _, err := FetchIOSReleaseNotes(client, "2026.4.1p4"); !errors.Is(err, ErrReleaseNotesNotFound) {
		t.Fatalf("expected a beta-only version to have no live notes, got %v", err)
	}
}
