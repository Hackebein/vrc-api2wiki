package mediawiki

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Hackebein/vrc-api2wiki/pkg/steam"
	"github.com/Hackebein/vrc-api2wiki/pkg/vcc"
)

func TestWindowsBuildPublishesOneRevision(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		editResult string
		wantError  bool
	}{
		{"success", "Success", false},
		{"failed edit", "Failure", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			const title = "Template:ClientBuild/open-beta-windows"
			edits := make(chan string, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.FormValue("action") {
				case "query":
					if got := r.FormValue("titles"); got != title {
						t.Errorf("queried %q, want %q", got, title)
					}
					fmt.Fprint(w, `{"query":{"pages":{"-1":{"missing":""}}}}`)
				case "edit":
					if got := r.FormValue("title"); got != title {
						t.Errorf("edited %q, want %q", got, title)
					}
					select {
					case edits <- r.FormValue("text"):
					default:
						t.Error("unexpected extra edit")
					}
					fmt.Fprintf(w, `{"edit":{"result":%q}}`, scenario.editResult)
				default:
					t.Errorf("unexpected wiki action %q", r.FormValue("action"))
					fmt.Fprint(w, `{"error":{"code":"unexpected-action"}}`)
				}
			}))
			defer server.Close()
			wiki := &MediaWikiClient{apiURL: server.URL, httpClient: server.Client(), tokens: map[string]string{"csrf": "test"}}
			build := &steam.ClientBuild{
				Version: "2026.4.2", BuildNumber: "1950", BuildHash: "4a2b57ca98",
				UnityVersion: "6000.0.70f1-DWR",
			}
			err := writeClientBuild(wiki, t.TempDir(), steam.WindowsOpenBetaClientName, build, map[string]int{"PC": 1936}, nil)
			if (err != nil) != scenario.wantError {
				t.Fatalf("sync error = %v, want error %t", err, scenario.wantError)
			}
			if len(edits) != 1 {
				t.Fatalf("got %d edits, want exactly one attempted edit", len(edits))
			}
			text := <-edits
			for _, field := range []string{
				"|version = 2026.4.2\n", "|buildNumber = 1950\n", "|buildHash = 4a2b57ca98\n",
				"|unityVersion = 6000.0.70f1-DWR\n", "|display = 2026.4.2 (1950)\n",
			} {
				if !strings.Contains(text, field) {
					t.Errorf("same revision must contain %q; got %q", field, text)
				}
			}
		})
	}
}

func TestCreatorCompanionPublishesVersionWithoutBuild(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"query":{"pages":{"-1":{"missing":""}}}}`)
	}))
	defer server.Close()
	output := t.TempDir()
	wiki := &MediaWikiClient{apiURL: server.URL, httpClient: server.Client(), offline: true, outputDir: output}
	if err := writeToolVersion(wiki, t.TempDir(), vcc.CreatorCompanionClientName, "2.4.5", nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(wiki.pageFilePath("Template:ClientBuild/creator-companion"))
	if err != nil {
		t.Fatal(err)
	}
	if text := string(data); !strings.Contains(text, "|display = 2.4.5\n") || !strings.Contains(text, "|buildNumber = \n") {
		t.Fatalf("version-only display or empty build missing from %q", text)
	}
	if files, err := os.ReadDir(output); err != nil || len(files) != 1 {
		t.Fatalf("offline sync must produce one wiki page, got %v, %v", files, err)
	}
}
