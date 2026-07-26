package vrchat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
)

func TestClientConfigPlatform(t *testing.T) {
	cases := map[string]string{
		"public":           "PC",
		"open-beta":        "PC",
		"quest":            "QuestStore",
		"open-beta-quest":  "QuestStore",
		"android-steamos":  "Default",
		"something-else":   "Default",
	}
	for client, want := range cases {
		if got := ClientConfigPlatform(client); got != want {
			t.Fatalf("%s: got %q want %q", client, got, want)
		}
	}
}

func TestClearClientBuildIfBelowMin(t *testing.T) {
	b := &steam.ClientBuild{Version: "2026.2.3", BuildNumber: "1862", BuildHash: "abc"}
	if !ClearClientBuildIfBelowMin(b, 1865) {
		t.Fatal("expected clear")
	}
	if b.Version != "" || b.BuildNumber != "" || b.BuildHash != "" {
		t.Fatalf("not cleared: %+v", b)
	}

	ok := &steam.ClientBuild{Version: "2026.2.3p3", BuildNumber: "1867", BuildHash: "def"}
	if ClearClientBuildIfBelowMin(ok, 1865) {
		t.Fatal("should keep")
	}
	if ok.Version != "2026.2.3p3" || ok.BuildNumber != "1867" {
		t.Fatalf("%+v", ok)
	}

	bad := &steam.ClientBuild{Version: "x", BuildNumber: "nope", BuildHash: "h"}
	if !ClearClientBuildIfBelowMin(bad, 1) {
		t.Fatal("unparseable should clear")
	}
}

func TestGetMinSupportedClientBuildNumbers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/config" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"minSupportedClientBuildNumber": map[string]any{
				"PC":         map[string]any{"minBuildNumber": 1865},
				"QuestStore": map[string]any{"minBuildNumber": 1865},
				"Default":    map[string]any{"minBuildNumber": 1860},
			},
		})
	}))
	defer srv.Close()

	prev := apiBase
	apiBase = srv.URL
	defer func() { apiBase = prev }()

	c := NewClient(srv.Client())
	mins, err := c.GetMinSupportedClientBuildNumbers()
	if err != nil {
		t.Fatal(err)
	}
	if mins["PC"] != 1865 || mins["QuestStore"] != 1865 {
		t.Fatalf("%v", mins)
	}
	n, err := MinBuildForClient(mins, "public")
	if err != nil || n != 1865 {
		t.Fatalf("public min %d %v", n, err)
	}
	n, err = MinBuildForClient(mins, "android-steamos")
	if err != nil || n != 1860 {
		t.Fatalf("android-steamos min %d %v", n, err)
	}
}
