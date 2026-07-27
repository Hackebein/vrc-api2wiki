package mediawiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func TestDirtyWorldSubpaths(t *testing.T) {
	prev := map[string]any{
		"id":          "wrld_test",
		"name":        "Alpha",
		"visits":      float64(10),
		"favorites":   float64(1),
		"imageUrl":    "https://api.vrchat.cloud/api/1/file/file_a/1/file",
		"description": "same",
	}
	curr := map[string]any{
		"id":          "wrld_test",
		"name":        "Alpha",
		"visits":      float64(11),
		"favorites":   float64(1),
		"imageUrl":    "https://api.vrchat.cloud/api/1/file/file_a/1/file",
		"description": "same",
	}
	dirty := dirtyWorldSubpaths(prev, curr)
	if !dirty["visits"] {
		t.Fatal("visits should be dirty")
	}
	if dirty["name"] || dirty["favorites"] || dirty["imageUrl"] || dirty["description"] {
		t.Fatalf("unexpected dirty keys: %#v", dirty)
	}
}

func TestDirtyWorldSubpathsColdCache(t *testing.T) {
	curr := map[string]any{"id": "wrld_test", "name": "Alpha", "visits": float64(1)}
	dirty := dirtyWorldSubpaths(nil, curr)
	if !dirty["name"] || !dirty["visits"] {
		t.Fatalf("cold cache should mark all keys dirty: %#v", dirty)
	}
}

func TestAPICacheWorldRoundTrip(t *testing.T) {
	c := openAPICache(t.TempDir())
	world := map[string]any{"id": "wrld_x", "name": "X"}
	if err := c.SaveWorld("wrld_x", world); err != nil {
		t.Fatal(err)
	}
	got, ok := c.LoadWorld("wrld_x")
	if !ok || got["name"] != "X" {
		t.Fatalf("load %#v ok=%v", got, ok)
	}
	meta := worldDiscoveryMeta{Infoboxes: []string{"Infobox/World"}, Articles: []string{"Page A"}}
	if err := c.SaveWorldMeta("wrld_x", meta); err != nil {
		t.Fatal(err)
	}
	gotMeta, ok := c.LoadWorldMeta("wrld_x")
	if !ok || !stringListsEqual(gotMeta.Articles, meta.Articles) {
		t.Fatalf("meta %#v", gotMeta)
	}
}

func TestWritePageDoesNotReadContent(t *testing.T) {
	var queries atomic.Int32
	var edits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("action") {
		case "query":
			queries.Add(1)
			if r.Form.Get("meta") == "tokens" {
				_, _ = w.Write([]byte(`{"query":{"tokens":{"csrftoken":"tok"}}}`))
				return
			}
			// page content query — must not happen for WritePage
			_, _ = w.Write([]byte(`{"query":{"pages":{"1":{"revisions":[{"slots":{"main":{"*":"old"}}}]}}}}`))
		case "edit":
			edits.Add(1)
			_, _ = w.Write([]byte(`{"edit":{"result":"Success"}}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	requestDelay = 0
	t.Cleanup(func() { requestDelay = 100 * time.Millisecond })

	client := &MediaWikiClient{
		apiURL:     server.URL,
		httpClient: server.Client(),
		userAgent:  "test",
		tokens:     map[string]string{"csrf": "tok"},
	}
	if err := client.WritePage("Template:World/wrld_x/name", "New", true); err != nil {
		t.Fatal(err)
	}
	if queries.Load() != 0 {
		t.Fatalf("WritePage issued %d query requests", queries.Load())
	}
	if edits.Load() != 1 {
		t.Fatalf("edits=%d", edits.Load())
	}
}

func TestShelfListingsUnchanged(t *testing.T) {
	dir := t.TempDir()
	c := openAPICache(dir)
	listing := map[string]any{"id": "prod_1", "displayName": "Hat", "imageId": "file_1"}
	if err := c.SaveListing("prod_1", listing); err != nil {
		t.Fatal(err)
	}
	store := map[string]any{
		"shelves": []any{
			map[string]any{
				"shelfTitle":        "Hats",
				"shelfIconImageId":  "file_icon",
				"listings":          []any{listing},
			},
		},
	}
	if err := c.SaveStore(store); err != nil {
		t.Fatal(err)
	}
	prevStore, _ := c.LoadStore()
	shelf := vrchatStoreShelf(store)
	listings := map[string]map[string]any{"prod_1": listing}
	if !shelfListingsUnchanged(c, prevStore, shelf, listings) {
		t.Fatal("expected unchanged")
	}
	changed := map[string]any{"id": "prod_1", "displayName": "Hat", "imageId": "file_2"}
	listings["prod_1"] = changed
	if shelfListingsUnchanged(c, prevStore, shelf, listings) {
		t.Fatal("expected changed when imageId differs")
	}
}

func vrchatStoreShelf(store map[string]any) map[string]any {
	shelves, _ := store["shelves"].([]any)
	m, _ := shelves[0].(map[string]any)
	return m
}

func TestEnsureWorldMarkerSkipsWhenCached(t *testing.T) {
	dir := t.TempDir()
	apiSnap := openAPICache(dir)
	if err := apiSnap.SaveWorldMeta("wrld_m", worldDiscoveryMeta{Infoboxes: []string{"Infobox/World"}}); err != nil {
		t.Fatal(err)
	}
	client := &MediaWikiClient{offline: true, outputDir: t.TempDir(), logger: nil}
	if err := client.EnsureWorldMarkerPage("wrld_m", nil, apiSnap); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(client.outputDir)
	if len(entries) != 0 {
		t.Fatalf("expected no offline writes, got %v", entries)
	}
}

func TestPersistStoreAPICache(t *testing.T) {
	c := openAPICache(t.TempDir())
	snap := &vrchat.StoreSnapshot{
		Store:    map[string]any{"id": "esto"},
		Listings: map[string]map[string]any{"prod_1": {"id": "prod_1"}},
	}
	if err := persistStoreAPICache(c, snap, map[string]string{"Hats": "ShelfIcon Hats.png"}); err != nil {
		t.Fatal(err)
	}
	icons := c.LoadShelfIcons()
	if icons["Hats"] != "ShelfIcon Hats.png" {
		t.Fatalf("icons %#v", icons)
	}
	raw, _ := os.ReadFile(c.listingPath("prod_1"))
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil || got["id"] != "prod_1" {
		t.Fatalf("listing file %s err %v", raw, err)
	}
	if !strings.Contains(filepath.ToSlash(c.listingPath("prod_1")), "/store/listings/") {
		t.Fatalf("path %s", c.listingPath("prod_1"))
	}
}
