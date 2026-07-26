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
	if cb.Branch != QuestClientName {
		t.Fatalf("branch %q", cb.Branch)
	}
	if cb.SteamBuildID != "27044595435241978" {
		t.Fatalf("id %q", cb.SteamBuildID)
	}
}

func TestFetchVRChatQuestOpenBetaPrefersNamedChannel(t *testing.T) {
	const channelsBody = `{
	  "data": {
	    "node": {
	      "release_channels": {
	        "nodes": [
	          {
	            "id": "1",
	            "channel_name": "LIVE",
	            "latest_supported_binary": {
	              "id": "livebin",
	              "version": "2026.2.3p3-1867-42912f4b5c-Release",
	              "version_code": 1005160
	            }
	          },
	          {
	            "id": "2",
	            "channel_name": "Open Beta (Opt-in)",
	            "latest_supported_binary": {
	              "id": "betabin",
	              "version": "2026.3.1-1879-ecdbfcd219-Release",
	              "version_code": 1006150
	            }
	          }
	        ]
	      }
	    }
	  }
	}`
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(channelsBody)),
			Header:     make(http.Header),
		}, nil
	})}
	cb, err := FetchVRChatQuestOpenBetaBuild(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.3.1" || cb.BuildNumber != "1879" || cb.BuildHash != "ecdbfcd219" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != QuestOpenBetaClientName || cb.SteamBuildID != "betabin" {
		t.Fatalf("%+v", cb)
	}
}

func TestFetchVRChatQuestOpenBetaUsesNewestPrimaryWhenChannelHidden(t *testing.T) {
	const channelsBody = `{
	  "data": {
	    "node": {
	      "release_channels": {
	        "nodes": [
	          {
	            "id": "360466918137694",
	            "channel_name": "LIVE",
	            "latest_supported_binary": {
	              "id": "livebin",
	              "version": "2026.2.3p3-1867-42912f4b5c-Release",
	              "version_code": 1005160
	            }
	          }
	        ]
	      }
	    }
	  }
	}`
	const channelBody = `{
	  "data": {
	    "node": {
	      "application": {
	        "primary_binaries": {
	          "edges": [
	            {
	              "node": {
	                "id": "obbin",
	                "version": "2026.3.1-1879-6488d4bbd4-Release",
	                "version_code": 1006160
	              }
	            },
	            {
	              "node": {
	                "id": "livebin",
	                "version": "2026.2.3p3-1867-42912f4b5c-Release",
	                "version_code": 1005160
	              }
	            }
	          ]
	        }
	      }
	    }
	  }
	}`
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		_ = req.ParseForm()
		body := channelsBody
		if req.Form.Get("doc_id") == oculusReleaseChannelDocID {
			body = channelBody
		}
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	cb, err := FetchVRChatQuestOpenBetaBuild(client)
	if err != nil {
		t.Fatal(err)
	}
	if cb.Version != "2026.3.1" || cb.BuildNumber != "1879" || cb.BuildHash != "6488d4bbd4" {
		t.Fatalf("%+v", cb)
	}
	if cb.Branch != QuestOpenBetaClientName {
		t.Fatalf("branch %q", cb.Branch)
	}
}

func TestIsOpenBetaChannelName(t *testing.T) {
	for _, name := range []string{"Open Beta (Opt-in)", "open beta", "BETA", "open-beta"} {
		if !isOpenBetaChannelName(name) {
			t.Fatalf("%q", name)
		}
	}
	if isOpenBetaChannelName("LIVE") {
		t.Fatal("LIVE")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
