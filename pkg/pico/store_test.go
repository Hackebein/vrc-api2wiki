package pico

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const itemInfoFixture = `{
  "code": 0,
  "msg": "",
  "data": {
    "item_id": 7288745304105664518,
    "version_code": 968210,
    "detail": {
      "app_version": "2026.2.3p1-1865-f71d38272d-Release"
    }
  }
}`

func TestFetchVRChatPicoBuildParsesAppVersion(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Fatalf("method %s", req.Method)
		}
		if req.URL.Host != "appstore-us.picoxr.com" {
			t.Fatalf("host %s", req.URL.Host)
		}
		if req.URL.Path != "/api/app/v1/item/info" {
			t.Fatalf("path %s", req.URL.Path)
		}
		q := req.URL.Query()
		if q.Get("device_name") == "" || q.Get("app_language") == "" || q.Get("manifest_version_code") == "" {
			t.Fatalf("query %s", req.URL.RawQuery)
		}
		if req.Header.Get("User-Agent") == "" {
			t.Fatal("missing User-Agent")
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type %s", req.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(req.Body)
		var sent itemInfoRequest
		if err := json.Unmarshal(body, &sent); err != nil {
			t.Fatal(err)
		}
		if sent.ItemID != 7288745304105664518 {
			t.Fatalf("item_id %d", sent.ItemID)
		}
		return jsonResponse(itemInfoFixture), nil
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
		return jsonResponse(`{"code":0,"msg":"","data":{}}`), nil
	})}
	_, err := FetchVRChatPicoBuild(client)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchVRChatPicoBuildAPIError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(`{"code":10001,"msg":"invalid parameter","data":{}}`), nil
	})}
	_, err := FetchVRChatPicoBuild(client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid parameter") {
		t.Fatalf("%v", err)
	}
}

func TestExtractAppVersion(t *testing.T) {
	v, err := extractAppVersion([]byte(itemInfoFixture))
	if err != nil {
		t.Fatal(err)
	}
	if v != "2026.2.3p1-1865-f71d38272d-Release" {
		t.Fatalf("%q", v)
	}
	if _, err := extractAppVersion([]byte(`{"code":0,"data":{}}`)); err == nil {
		t.Fatal("expected error")
	}
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
