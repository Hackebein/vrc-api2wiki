package meta

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFetchVRChatQuestBuildParsesFixture(t *testing.T) {
	const body = `{
	  "data": {
	    "node": {
	      "supportedBinaries": {
	        "edges": [
	          {
	            "node": {
	              "id": "27044595435241978",
	              "version": "2026.2.3p3-1867-42912f4b5c-Release",
	              "versionCode": 1005160,
	              "changeLog": null
	            }
	          }
	        ]
	      }
	    }
	  }
	}`
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "graph.oculus.com" {
			t.Fatalf("host %s", req.URL.Host)
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	cb, err := FetchVRChatQuestBuild(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.2.3p3" || cb.BuildNumber != "1867" || cb.BuildHash != "42912f4b5c" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != QuestAndroidClientName {
		t.Fatalf("branch %q", cb.Branch)
	}
	if cb.SteamBuildID != "27044595435241978" {
		t.Fatalf("id %q", cb.SteamBuildID)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
