package vrchat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	OfficialStoreID = "esto_00000000-0000-0000-0000-000000000000"
)

var unsafeFilenameChars = regexp.MustCompile(`[<>:"/\\|?*]`)

type StoreSnapshot struct {
	FetchedAt   time.Time                 `json:"fetchedAt"`
	Store       map[string]any            `json:"store"`
	Listings    map[string]map[string]any `json:"listings"`
	Avatars     []map[string]any          `json:"avatars"`
	TypeCounts  map[string]int            `json:"typeCounts"`
	SnapshotDir string                    `json:"-"`
}

func SafeFilename(name string) string {
	s := unsafeFilenameChars.ReplaceAllString(name, "")
	return strings.ReplaceAll(s, " ", "_")
}

func (c *Client) FetchStoreCatalog(snapshotDir string, avatarLimit int, logger *slog.Logger) (*StoreSnapshot, error) {
	snap := &StoreSnapshot{
		FetchedAt:   time.Now().UTC(),
		Listings:    map[string]map[string]any{},
		TypeCounts:  map[string]int{},
		SnapshotDir: snapshotDir,
	}
	if snapshotDir != "" {
		if err := os.MkdirAll(filepath.Join(snapshotDir, "Listings"), 0o755); err != nil {
			return nil, err
		}
	}

	storeURL := fmt.Sprintf("%s/economy/store?storeId=%s&hydrateListings=true&hydrateProducts=true", apiBase, OfficialStoreID)
	var store map[string]any
	if err := c.AuthedGetJSON(storeURL, &store); err != nil {
		return nil, fmt.Errorf("get store: %w", err)
	}
	snap.Store = store
	if snapshotDir != "" {
		if err := writeJSON(filepath.Join(snapshotDir, "vrchat_store_response.json"), store); err != nil {
			return nil, err
		}
	}

	listingIDs := collectListingIDs(store)

	for id := range listingIDs {
		listing, err := c.fetchHydratedListing(id, snapshotDir)
		if err != nil {
			return nil, fmt.Errorf("listing %s: %w", id, err)
		}
		snap.Listings[id] = listing
		snap.TypeCounts[ListingTypeLabel(listing)]++
	}

	// avatars, err := c.FetchPaidMarketplaceAvatars(snapshotDir, avatarLimit)
	// if err != nil {
	// 	return nil, fmt.Errorf("marketplace avatars: %w", err)
	// }
	// snap.Avatars = avatars
	// snap.TypeCounts["Avatar"] += len(avatars)
	// if logger != nil && avatarLimit > 0 {
	// 	logger.Info("avatar limit applied", "limit", avatarLimit, "fetched", len(avatars))

	if snapshotDir != "" {
		if err := writeJSON(filepath.Join(snapshotDir, "snapshot_meta.json"), snap); err != nil {
			return nil, err
		}
	}
	return snap, nil
}

func collectListingIDs(store map[string]any) map[string]bool {
	ids := map[string]bool{}
	shelves, _ := store["shelves"].([]any)
	for _, s := range shelves {
		shelf, _ := s.(map[string]any)
		if shelf == nil {
			continue
		}
		if hl, ok := shelf["highlightListing"].(map[string]any); ok {
			if id, _ := hl["id"].(string); id != "" {
				ids[id] = true
			}
		}
		listings, _ := shelf["listings"].([]any)
		for _, l := range listings {
			lm, _ := l.(map[string]any)
			if lm == nil {
				continue
			}
			if id, _ := lm["id"].(string); id != "" {
				ids[id] = true
			}
		}
		for _, id := range stringSlice(shelf["listingIds"]) {
			ids[id] = true
		}
	}
	return ids
}

func (c *Client) fetchHydratedListing(id, snapshotDir string) (map[string]any, error) {
	u := fmt.Sprintf("%s/listing/%s?hydrate=true", apiBase, id)
	var listing map[string]any
	if err := c.AuthedGetJSON(u, &listing); err != nil {
		u2 := fmt.Sprintf("https://vrchat.com/api/1/listing/%s?hydrate=true", id)
		if err2 := c.AuthedGetJSON(u2, &listing); err2 != nil {
			return nil, err
		}
	}
	if snapshotDir != "" {
		name, _ := listing["displayName"].(string)
		dir := filepath.Join(snapshotDir, "Listings", SafeFilename(name))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		if err := writeJSON(filepath.Join(dir, id+".json"), listing); err != nil {
			return nil, err
		}
		gallery, gerr := c.FetchListingGallery(id)
		if gerr == nil {
			_ = writeJSON(filepath.Join(dir, "product_image_gallery_response.json"), gallery)
		}
	}
	return listing, nil
}

func (c *Client) FetchListingGallery(listingID string) ([]map[string]any, error) {
	u := fmt.Sprintf("%s/files?n=64&offset=0&sort=order&order=ascending&tag=listinggallery&galleryId=%s", apiBase, listingID)
	body, err := c.AuthedGet(u)
	if err != nil {
		u2 := fmt.Sprintf("https://vrchat.com/api/1/files?n=64&offset=0&sort=order&order=ascending&tag=listinggallery&galleryId=%s", listingID)
		body, err = c.AuthedGet(u2)
		if err != nil {
			return nil, err
		}
	}
	var page []map[string]any
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parse listing gallery %s: %w", listingID, err)
	}
	return page, nil
}

func ListingProducts(listing map[string]any) []map[string]any {
	var out []map[string]any
	for _, key := range []string{"products", "hydratedProducts"} {
		raw, _ := listing[key].([]any)
		for _, p := range raw {
			if m, ok := p.(map[string]any); ok {
				out = append(out, m)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return out
}

type FileDownloadInfo struct {
	FileID   string
	Version  int
	URL      string
	FileName string
	Ext      string
}

func (c *Client) GetFileDownload(fileID string) (*FileDownloadInfo, error) {
	u := fmt.Sprintf("%s/file/%s", apiBase, fileID)
	var meta map[string]any
	if err := c.AuthedGetJSON(u, &meta); err != nil {
		u2 := fmt.Sprintf("https://vrchat.com/api/1/file/%s", fileID)
		if err2 := c.AuthedGetJSON(u2, &meta); err2 != nil {
			return nil, err
		}
	}
	versions, _ := meta["versions"].([]any)
	if len(versions) == 0 {
		return nil, fmt.Errorf("file %s: no versions", fileID)
	}
	last, _ := versions[len(versions)-1].(map[string]any)
	fileObj, _ := last["file"].(map[string]any)
	if fileObj == nil {
		return nil, fmt.Errorf("file %s: missing file object", fileID)
	}
	ver, _ := last["version"].(float64)
	ext, _ := meta["extension"].(string)
	info := &FileDownloadInfo{
		FileID:   fileID,
		Version:  int(ver),
		URL:      stringField(fileObj, "url"),
		FileName: stringField(fileObj, "fileName"),
		Ext:      strings.TrimPrefix(ext, "."),
	}
	if info.URL == "" {
		return nil, fmt.Errorf("file %s: empty download url", fileID)
	}
	if info.Ext == "" && info.FileName != "" {
		if i := strings.LastIndex(info.FileName, "."); i >= 0 {
			info.Ext = info.FileName[i+1:]
		}
	}
	if info.Ext == "" {
		info.Ext = "png"
	}
	return info, nil
}

func (c *Client) DownloadFileBytes(fileID string) ([]byte, string, error) {
	info, err := c.GetFileDownload(fileID)
	if err != nil {
		return nil, "", err
	}
	data, ext, err := c.DownloadImage(info.URL)
	if err != nil {
		body, err2 := c.getBytes(info.URL)
		if err2 != nil {
			return nil, "", err
		}
		if ext == "" {
			ext = info.Ext
		}
		return body, ext, nil
	}
	if ext == "" {
		ext = info.Ext
	}
	return data, ext, nil
}

func ListingTypeLabel(listing map[string]any) string {
	if b, _ := listing["hasCompanion"].(bool); b {
		sub := stringField(listing, "subtitle")
		if sub == "" || strings.EqualFold(sub, "Bundle") {
			return "Companion"
		}
	}
	sub := stringField(listing, "subtitle")
	if sub == "" {
		return "Unknown"
	}
	if strings.EqualFold(sub, "Bundle") {
		if listingIsEmojiBundle(listing) {
			return "Emoji"
		}
		return "Bundle"
	}
	return strings.ReplaceAll(sub, " ", "")
}

func listingIsEmojiBundle(listing map[string]any) bool {
	products, _ := listing["hydratedProducts"].([]any)
	if len(products) == 0 {
		products, _ = listing["products"].([]any)
	}
	if len(products) == 0 {
		return false
	}
	emoji := 0
	for _, p := range products {
		prod, _ := p.(map[string]any)
		if prod == nil {
			continue
		}
		label := strings.ToLower(stringField(prod, "productTypeLabel"))
		itemType := strings.ToLower(stringField(prod, "inventoryItemType"))
		if label == "emoji" || itemType == "emoji" {
			emoji++
		}
	}
	return emoji > 0 && emoji == len(products)
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func StoreShelves(store map[string]any) []map[string]any {
	raw, _ := store["shelves"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, s := range raw {
		if m, ok := s.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func ShelfListings(shelf map[string]any) []map[string]any {
	var out []map[string]any
	if hl, ok := shelf["highlightListing"].(map[string]any); ok {
		out = append(out, hl)
	}
	raw, _ := shelf["listings"].([]any)
	for _, l := range raw {
		if m, ok := l.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
