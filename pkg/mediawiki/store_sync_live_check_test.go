//go:build livecheck

package mediawiki

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func TestLiveShelfParity(t *testing.T) {
	if os.Getenv("VRCHAT_USERNAME") == "" {
		t.Skip("no auth")
	}
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Timeout: 120 * time.Second, Jar: jar}
	readWiki, err := NewMediaWikiClient(WikiConfig{
		URL: "https://wiki.vrchat.com/api.php",
		Username: os.Getenv("VRCWIKI_USERNAME"), Password: os.Getenv("VRCWIKI_PASSWORD"),
		Header: os.Getenv("VRCWIKI_AUTHORIZATION_HEADER"), HeaderVal: os.Getenv("VRCWIKI_AUTHORIZATION_VALUE"),
	}, httpClient)
	if err != nil { t.Fatal(err) }

	_ = os.Unsetenv("VRCWIKI_USERNAME")
	_ = os.Unsetenv("VRCWIKI_PASSWORD")
	offlineWiki, err := NewMediaWikiClient(WikiConfig{
		URL: "https://wiki.vrchat.com/api.php",
		Header: os.Getenv("VRCWIKI_AUTHORIZATION_HEADER"), HeaderVal: os.Getenv("VRCWIKI_AUTHORIZATION_VALUE"),
	}, httpClient)
	if err != nil { t.Fatal(err) }

	api := vrchat.NewClient(httpClient)
	cfg, _ := vrchat.AuthConfigFromEnv()
	cfg.CookiePath = ".vrchat-session/cookies.jar"
	if err := api.EnsureAuth(cfg, nil); err != nil { t.Fatal(err) }

	snapDir := filepath.Join("..", "..", "store-data", "2026_07_25_22_25_08")
	store := map[string]any{}
	b, _ := os.ReadFile(filepath.Join(snapDir, "vrchat_store_response.json"))
	json.Unmarshal(b, &store)
	listings := map[string]map[string]any{}
	filepath.WalkDir(filepath.Join(snapDir, "Listings"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasPrefix(d.Name(), "prod_") { return nil }
		var m map[string]any
		bb, _ := os.ReadFile(path)
		json.Unmarshal(bb, &m)
		if id, _ := m["id"].(string); id != "" { listings[id] = m }
		return nil
	})

	now := time.Now()
	year := 2026
	wikiShelves := loadWikiShelfIndex(readWiki, year, nil)
	for _, shelf := range vrchat.StoreShelves(store) {
		apiTitle, _ := shelf["shelfTitle"].(string)
		var listingIDs []string
		for _, listing := range vrchat.ShelfListings(shelf) {
			if id, _ := listing["id"].(string); id != "" { listingIDs = append(listingIDs, id) }
		}
		wikiTitle := resolveWikiShelfTitle(apiTitle, listingIDs, wikiShelves)
		existingPage, _ := readWiki.GetPageContent(StoreListingsPageTitle(year, wikiTitle))
		existingFields := parseInventoryWikiFields(existingPage)
		existingIcon := parseShelfIconFromHeader(existingPage)
		iconFile, _ := syncShelfIcon(offlineWiki, api, shelf, wikiTitle, existingIcon, nil)
		var cards []vrchat.InventoryContentDisplay
		for _, listing := range vrchat.ShelfListings(shelf) {
			id, _ := listing["id"].(string)
			hydrated := listings[id]
			if hydrated == nil { hydrated = listing }
			preferred := ""
			if old, ok := existingFields[id]; ok {
				preferred = old.Image
			}
			imageName, err := syncListingMedia(offlineWiki, api, hydrated, preferred, nil)
			if err != nil { t.Fatal(err) }
			card := vrchat.ListingToDisplay(hydrated, now, imageName)
			if old, ok := existingFields[id]; ok {
				if old.AvailabilityDate != "" { card.AvailabilityDate = old.AvailabilityDate }
				if old.PriceText != "" { card.PriceText = old.PriceText }
			}
			cards = append(cards, card)
		}
		page := vrchat.RenderShelfWikitext(wikiTitle, iconFile, cards)
		if err := offlineWiki.EditPage(StoreListingsPageTitle(year, wikiTitle), page, true); err != nil {
			t.Fatal(err)
		}
		live := existingPage
		// compare image lines and productIDs
		liveImgs := map[string]string{}
		for id, f := range parseInventoryWikiFields(live) { liveImgs[id] = f.Image }
		genImgs := map[string]string{}
		for id, f := range parseInventoryWikiFields(page) { genImgs[id] = f.Image }
		t.Logf("SHELF %q icon=%q cards=%d", wikiTitle, iconFile, len(cards))
		for id, img := range liveImgs {
			if g, ok := genImgs[id]; !ok {
				t.Logf("  missing on gen: %s (%s)", id, img)
			} else if g != img {
				t.Logf("  image mismatch %s: live=%q gen=%q", id, img, g)
			}
		}
		for id, img := range genImgs {
			if _, ok := liveImgs[id]; !ok {
				t.Logf("  new on gen: %s (%s)", id, img)
			}
		}
	}
}
