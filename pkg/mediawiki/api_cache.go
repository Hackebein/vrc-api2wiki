package mediawiki

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

const apiCacheDir = ".api-cache"

type apiCache struct {
	dir string
}

func openAPICache(dir string) *apiCache {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = apiCacheDir
	}
	return &apiCache{dir: dir}
}

func (c *apiCache) worldPath(worldID string) string {
	return filepath.Join(c.dir, "worlds", worldID+".json")
}

func (c *apiCache) worldMetaPath(worldID string) string {
	return filepath.Join(c.dir, "worlds", worldID+".meta.json")
}

func (c *apiCache) storePath() string {
	return filepath.Join(c.dir, "store", "store.json")
}

func (c *apiCache) listingPath(id string) string {
	return filepath.Join(c.dir, "store", "listings", id+".json")
}

func (c *apiCache) shelfIconsPath() string {
	return filepath.Join(c.dir, "store", "shelf_icons.json")
}

type worldDiscoveryMeta struct {
	Infoboxes []string `json:"infoboxes,omitempty"`
	Articles  []string `json:"articles,omitempty"`
}

func (c *apiCache) LoadWorld(worldID string) (map[string]any, bool) {
	data, err := os.ReadFile(c.worldPath(worldID))
	if err != nil || len(data) == 0 {
		return nil, false
	}
	var world map[string]any
	if err := json.Unmarshal(data, &world); err != nil || world == nil {
		return nil, false
	}
	return world, true
}

func (c *apiCache) SaveWorld(worldID string, world map[string]any) error {
	return writeAPICacheJSON(c.worldPath(worldID), world)
}

func (c *apiCache) LoadWorldMeta(worldID string) (worldDiscoveryMeta, bool) {
	data, err := os.ReadFile(c.worldMetaPath(worldID))
	if err != nil || len(data) == 0 {
		return worldDiscoveryMeta{}, false
	}
	var meta worldDiscoveryMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return worldDiscoveryMeta{}, false
	}
	return meta, true
}

func (c *apiCache) SaveWorldMeta(worldID string, meta worldDiscoveryMeta) error {
	meta.Infoboxes = normalizedStringList(meta.Infoboxes)
	meta.Articles = normalizedStringList(meta.Articles)
	return writeAPICacheJSON(c.worldMetaPath(worldID), meta)
}

func (c *apiCache) LoadStore() (map[string]any, bool) {
	data, err := os.ReadFile(c.storePath())
	if err != nil || len(data) == 0 {
		return nil, false
	}
	var store map[string]any
	if err := json.Unmarshal(data, &store); err != nil || store == nil {
		return nil, false
	}
	return store, true
}

func (c *apiCache) SaveStore(store map[string]any) error {
	return writeAPICacheJSON(c.storePath(), store)
}

func (c *apiCache) LoadListing(id string) (map[string]any, bool) {
	data, err := os.ReadFile(c.listingPath(id))
	if err != nil || len(data) == 0 {
		return nil, false
	}
	var listing map[string]any
	if err := json.Unmarshal(data, &listing); err != nil || listing == nil {
		return nil, false
	}
	return listing, true
}

func (c *apiCache) SaveListing(id string, listing map[string]any) error {
	return writeAPICacheJSON(c.listingPath(id), listing)
}

func (c *apiCache) LoadShelfIcons() map[string]string {
	data, err := os.ReadFile(c.shelfIconsPath())
	if err != nil || len(data) == 0 {
		return map[string]string{}
	}
	var icons map[string]string
	if err := json.Unmarshal(data, &icons); err != nil || icons == nil {
		return map[string]string{}
	}
	return icons
}

func (c *apiCache) SaveShelfIcons(icons map[string]string) error {
	if icons == nil {
		icons = map[string]string{}
	}
	return writeAPICacheJSON(c.shelfIconsPath(), icons)
}

func writeAPICacheJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("api cache mkdir: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("api cache write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("api cache rename: %w", err)
	}
	return nil
}

func normalizedStringList(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func stringListsEqual(a, b []string) bool {
	a = normalizedStringList(a)
	b = normalizedStringList(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func apiMapsEqual(a, b map[string]any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

// dirtyWorldSubpaths returns FlattenWorld keys whose sanitized wiki text differs
// between prev and curr API world objects. When prev is nil, every curr key is dirty.
func dirtyWorldSubpaths(prev, curr map[string]any) map[string]bool {
	currPages := vrchat.FlattenWorld(curr)
	dirty := make(map[string]bool, len(currPages))
	if prev == nil {
		for k := range currPages {
			dirty[k] = true
		}
		return dirty
	}
	prevPages := vrchat.FlattenWorld(prev)
	for k, v := range currPages {
		pv, ok := prevPages[k]
		if !ok || SanitizeForWiki(pv) != SanitizeForWiki(v) {
			dirty[k] = true
		}
	}
	return dirty
}

func shelfIconImageID(shelf map[string]any) string {
	return stringField(shelf, "shelfIconImageId")
}

func findShelfByTitle(store map[string]any, apiTitle string) map[string]any {
	display := vrchat.DisplayShelfTitle(apiTitle)
	for _, shelf := range vrchat.StoreShelves(store) {
		if vrchat.DisplayShelfTitle(stringField(shelf, "shelfTitle")) == display {
			return shelf
		}
	}
	return nil
}
