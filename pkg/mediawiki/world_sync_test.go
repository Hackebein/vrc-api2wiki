package mediawiki

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFormatWorldPublicationDate(t *testing.T) {
	tests := []struct {
		name      string
		world     map[string]any
		want      string
		overwrite bool
	}{
		{
			name:      "publicationDate before Labs",
			world:     map[string]any{"publicationDate": "2018-03-17T08:58:42.296Z"},
			want:      preCommunityLabsPublicationDate,
			overwrite: true,
		},
		{
			name:      "publicationDate day before Labs",
			world:     map[string]any{"publicationDate": "2019-03-12T23:59:59Z"},
			want:      preCommunityLabsPublicationDate,
			overwrite: true,
		},
		{
			name:      "publicationDate at Labs launch",
			world:     map[string]any{"publicationDate": "2019-03-13T00:00:00Z"},
			want:      "2019-03-13T00:00:00Z",
			overwrite: false,
		},
		{
			name:      "later publicationDate",
			world:     map[string]any{"publicationDate": "2020-01-01T00:00:00.000Z"},
			want:      "2020-01-01T00:00:00.000Z",
			overwrite: false,
		},
		{
			name:      "unparseable publicationDate",
			world:     map[string]any{"publicationDate": "not-a-date"},
			want:      "not-a-date",
			overwrite: false,
		},
		{
			name: "created before Labs, never entered Labs",
			world: map[string]any{
				"created_at":          "2017-01-19T01:14:54.000Z",
				"publicationDate":     "2019-05-08T04:34:57.581Z",
				"labsPublicationDate": "none",
			},
			want:      preCommunityLabsPublicationDate,
			overwrite: true,
		},
		{
			name: "created before Labs, later entered Labs",
			world: map[string]any{
				"created_at":          "2017-11-12T05:30:26.368Z",
				"publicationDate":     "2019-05-29T20:42:18.181Z",
				"labsPublicationDate": "2019-05-18T04:09:08.413Z",
			},
			want:      "2019-05-29T20:42:18.181Z",
			overwrite: false,
		},
	}
	for _, tc := range tests {
		got, overwrite := formatWorldPublicationDate(tc.world)
		if got != tc.want || overwrite != tc.overwrite {
			t.Fatalf("%s: formatWorldPublicationDate() = %q, %v; want %q, %v", tc.name, got, overwrite, tc.want, tc.overwrite)
		}
	}
}

func TestSyncWorldDataWritesPreLabsPublicationDate(t *testing.T) {
	edits := map[string]string{}
	client := newWikiTestClient(t, map[string]string{}, edits)

	world := map[string]any{
		"id":              "wrld_test",
		"name":            "Alpha",
		"publicationDate": "2018-03-17T08:58:42.296Z",
	}
	if err := client.SyncWorldData(nil, "wrld_test", world, newImageSyncCache(), openAPICache(t.TempDir())); err != nil {
		t.Fatal(err)
	}

	got := edits["Template:World/wrld_test/publicationDate"]
	if got != preCommunityLabsPublicationDate {
		t.Fatalf("publicationDate write = %q, want pre-labs note; edits=%#v", got, edits)
	}
}

func TestSyncWorldDataOverwritesPreLabsPublicationDate(t *testing.T) {
	edits := map[string]string{}
	client := newWikiTestClient(t, map[string]string{
		"Template:World/wrld_test/publicationDate": "2018-03-17T08:58:42.296Z",
	}, edits)

	world := map[string]any{
		"id":              "wrld_test",
		"name":            "Alpha",
		"publicationDate": "2018-03-17T08:58:42.296Z",
	}
	if err := client.SyncWorldData(nil, "wrld_test", world, newImageSyncCache(), openAPICache(t.TempDir())); err != nil {
		t.Fatal(err)
	}

	got := edits["Template:World/wrld_test/publicationDate"]
	if got != preCommunityLabsPublicationDate {
		t.Fatalf("pre-labs publicationDate should be overwritten, edits=%#v", edits)
	}
}

func TestSyncWorldDataDoesNotOverwriteLaterPublicationDate(t *testing.T) {
	edits := map[string]string{}
	client := newWikiTestClient(t, map[string]string{
		"Template:World/wrld_test/publicationDate": "2018-01-01",
	}, edits)

	world := map[string]any{
		"id":              "wrld_test",
		"name":            "Alpha",
		"publicationDate": "2020-01-01T00:00:00.000Z",
	}
	if err := client.SyncWorldData(nil, "wrld_test", world, newImageSyncCache(), openAPICache(t.TempDir())); err != nil {
		t.Fatal(err)
	}

	if _, ok := edits["Template:World/wrld_test/publicationDate"]; ok {
		t.Fatalf("publicationDate should not be overwritten, edits=%#v", edits)
	}
	if edits["Template:World/wrld_test/name"] != "Alpha" {
		t.Fatalf("name should still be written, edits=%#v", edits)
	}
}

func TestSyncWorldDataSeedsLaterPublicationDateWhenMissing(t *testing.T) {
	world := map[string]any{
		"id":              "wrld_test",
		"name":            "Alpha",
		"publicationDate": "2020-01-01T00:00:00.000Z",
	}
	apiSnap := openAPICache(t.TempDir())
	if err := apiSnap.SaveWorld("wrld_test", world); err != nil {
		t.Fatal(err)
	}

	edits := map[string]string{}
	client := newWikiTestClient(t, map[string]string{}, edits)
	if err := client.SyncWorldData(nil, "wrld_test", world, newImageSyncCache(), apiSnap); err != nil {
		t.Fatal(err)
	}

	got := edits["Template:World/wrld_test/publicationDate"]
	if got != "2020-01-01T00:00:00.000Z" {
		t.Fatalf("missing later publicationDate should be seeded, edits=%#v", edits)
	}
	if _, ok := edits["Template:World/wrld_test/name"]; ok {
		t.Fatalf("unchanged name should not be rewritten, edits=%#v", edits)
	}
}

func TestSyncWorldDataOverwritesCreatedBeforeLabs(t *testing.T) {
	edits := map[string]string{}
	client := newWikiTestClient(t, map[string]string{
		"Template:World/wrld_test/publicationDate": "2019-05-08T04:34:57.581Z",
	}, edits)

	world := map[string]any{
		"id":                  "wrld_test",
		"name":                "The Great Pug",
		"created_at":          "2017-01-19T01:14:54.000Z",
		"publicationDate":     "2019-05-08T04:34:57.581Z",
		"labsPublicationDate": "none",
	}
	if err := client.SyncWorldData(nil, "wrld_test", world, newImageSyncCache(), openAPICache(t.TempDir())); err != nil {
		t.Fatal(err)
	}

	got := edits["Template:World/wrld_test/publicationDate"]
	if got != preCommunityLabsPublicationDate {
		t.Fatalf("created-before-Labs world should be overwritten, edits=%#v", edits)
	}
}

func TestSyncWorldDataSeedsPreLabsPublicationDateWhenNotDirty(t *testing.T) {
	world := map[string]any{
		"id":              "wrld_test",
		"name":            "Alpha",
		"publicationDate": "2018-03-17T08:58:42.296Z",
	}
	apiSnap := openAPICache(t.TempDir())
	if err := apiSnap.SaveWorld("wrld_test", world); err != nil {
		t.Fatal(err)
	}

	edits := map[string]string{}
	client := newWikiTestClient(t, map[string]string{}, edits)
	if err := client.SyncWorldData(nil, "wrld_test", world, newImageSyncCache(), apiSnap); err != nil {
		t.Fatal(err)
	}

	got := edits["Template:World/wrld_test/publicationDate"]
	if got != preCommunityLabsPublicationDate {
		t.Fatalf("missing pre-labs publicationDate should still be written, edits=%#v", edits)
	}
}

func newWikiTestClient(t *testing.T, pages, edits map[string]string) *MediaWikiClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("action") {
		case "query":
			if r.Form.Get("meta") == "tokens" {
				_, _ = w.Write([]byte(`{"query":{"tokens":{"csrftoken":"tok"}}}`))
				return
			}
			title := r.Form.Get("titles")
			content, ok := pages[title]
			if !ok {
				_, _ = w.Write([]byte(`{"query":{"pages":{"-1":{"missing":""}}}}`))
				return
			}
			fmt.Fprintf(w, `{"query":{"pages":{"1":{"revisions":[{"slots":{"main":{"*":%q}}}]}}}}`, content)
		case "edit":
			title := r.Form.Get("title")
			text := r.Form.Get("text")
			edits[title] = text
			pages[title] = text
			_, _ = w.Write([]byte(`{"edit":{"result":"Success"}}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	requestDelay = 0
	t.Cleanup(func() { requestDelay = 100 * time.Millisecond })

	return &MediaWikiClient{
		apiURL:     server.URL,
		httpClient: server.Client(),
		userAgent:  "test",
		tokens:     map[string]string{"csrf": "tok"},
	}
}
