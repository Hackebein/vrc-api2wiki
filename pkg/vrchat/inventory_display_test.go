package vrchat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListingToDisplayAccessory(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "StoreLogger", "Response_Data", "2026_07_10 21_04_10", "Listings", "Vega’s_Tiara", "prod_d7a0343f-d965-4d03-9dba-a7fb5603b67f.json"))
	if err != nil {
		t.Skip(err)
	}
	var listing map[string]any
	if err := json.Unmarshal(raw, &listing); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	d := ListingToDisplay(listing, now, "ListingImage Vega’s Tiara.jpg")
	if d.Name != "Vega’s Tiara" {
		t.Fatalf("name %q", d.Name)
	}
	if d.Type != "Accessory" {
		t.Fatalf("type %q", d.Type)
	}
	if d.Price != 1000 || d.PriceVRCPlus != 900 {
		t.Fatalf("prices %d/%d", d.Price, d.PriceVRCPlus)
	}
	if d.Image != "ListingImage Vega’s Tiara.jpg" {
		t.Fatalf("image %q", d.Image)
	}
	wikitext := RenderInventoryContentDisplay(d)
	if !strings.Contains(wikitext, "|type=Accessory") {
		t.Fatalf("missing type: %s", wikitext)
	}
	if !strings.Contains(wikitext, "|seller=VRChat") {
		t.Fatalf("expected seller: %s", wikitext)
	}
	if !strings.Contains(wikitext, "|image=ListingImage Vega’s Tiara.jpg") {
		t.Fatalf("expected jpg image: %s", wikitext)
	}
}

func TestRenderShelfMatchesShape(t *testing.T) {
	cards := []InventoryContentDisplay{{
		Name: "Test", Image: "ListingImage Test.png", Seller: "VRChat",
		Type: "Item", Price: 100, PriceVRCPlus: 90,
		AvailabilityDate: "July 10, 2026 - ???", Description: "desc", ProductID: "prod_x",
	}}
	out := RenderShelfWikitext("New Accessories", "ShelfIcon New Accessories.png", cards)
	if !strings.Contains(out, `[[File:ShelfIcon New Accessories.png|frameless|middle|link=|30px]] New Accessories`) {
		t.Fatal(out)
	}
	if !strings.Contains(out, "[[Category:Shop templates]]") {
		t.Fatal(out)
	}
}

func TestOpenBetaShelfTitleClean(t *testing.T) {
	out := RenderShelfWikitext(`{"MinVersion":1878} Open Beta Profile Decorations`, "", nil)
	if !strings.Contains(out, `>Open Beta Profile Decorations</span>`) {
		t.Fatal(out)
	}
	if strings.Contains(out, "MinVersion") {
		t.Fatal(out)
	}
}

func TestFreePriceOmitsVRCPlus(t *testing.T) {
	wt := RenderInventoryContentDisplay(InventoryContentDisplay{
		Name: "Free Thing", Image: "ListingImage Free Thing.png", Seller: "VRChat",
		Type: "Sticker", Price: 0, PriceVRCPlus: 0,
		AvailabilityDate: "July 10, 2026 - ???", Description: "d", ProductID: "prod_f",
	})
	if !strings.Contains(wt, "|price=Free\n") {
		t.Fatal(wt)
	}
	if strings.Contains(wt, "price_vrcplus") {
		t.Fatal(wt)
	}
}

func TestBundleOmitsType(t *testing.T) {
	d := ListingToDisplay(map[string]any{
		"displayName": "Pack", "subtitle": "Bundle", "id": "prod_b",
		"priceTokens": 1.0, "vrcPlusDiscountPrice": 1.0, "description": "x",
	}, time.Now(), "")
	if d.Type != "" {
		t.Fatalf("type should be empty, got %q", d.Type)
	}
	wt := RenderInventoryContentDisplay(d)
	if strings.Contains(wt, "|type=") {
		t.Fatal(wt)
	}
}

func TestPropMapsToItem(t *testing.T) {
	d := ListingToDisplay(map[string]any{
		"displayName": "Taiyaki", "subtitle": "Prop", "id": "prod_p",
		"priceTokens": 100.0, "description": "x",
	}, time.Now(), "")
	if d.Type != "Item" {
		t.Fatalf("type %q", d.Type)
	}
}

func TestEmojiBundleKeepsType(t *testing.T) {
	d := ListingToDisplay(map[string]any{
		"displayName": "Tanabata Emoji Pack",
		"subtitle":    "Bundle",
		"id":          "prod_e",
		"priceTokens": 500.0,
		"description": "x",
		"hydratedProducts": []any{
			map[string]any{"productTypeLabel": "Emoji", "inventoryItemType": "emoji"},
			map[string]any{"productTypeLabel": "Emoji", "inventoryItemType": "emoji"},
		},
	}, time.Now(), "")
	if d.Type != "Emoji" {
		t.Fatalf("type %q", d.Type)
	}
	wt := RenderInventoryContentDisplay(d)
	if !strings.Contains(wt, "|type=Emoji\n") {
		t.Fatal(wt)
	}
}

func TestListingImageFilenameTrimsDisplayName(t *testing.T) {
	got := ListingImageFilename("Tanabata Emoji Pack ", "jpg")
	if got != "ListingImage Tanabata Emoji Pack.jpg" {
		t.Fatalf("%q", got)
	}
}

func TestAvatarPrimaryStyleAndDisplay(t *testing.T) {
	av := map[string]any{
		"id":          "avtr_1",
		"productId":   "prod_1",
		"name":        "Test Avatar",
		"authorName":  "Creator",
		"description": "desc",
		"lowestPrice": 500.0,
		"listingDate": "2026-05-11T23:57:22.402Z",
		"styles":      map[string]any{"primary": "Anime", "secondary": "Human"},
	}
	if got := AvatarPrimaryStyle(av); got != "Anime" {
		t.Fatalf("style %q", got)
	}
	if AvatarPrimaryStyle(map[string]any{}) != "Uncategorized" {
		t.Fatal("expected Uncategorized")
	}
	d := AvatarToDisplay(av, time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC), "ListingImage Test Avatar.png")
	if d.Type != "Avatar" || d.Price != 500 || d.ProductID != "prod_1" || d.AvatarID != "avtr_1" {
		t.Fatalf("%+v", d)
	}
	if d.AvailabilityDate != "May 11, 2026 - ???" {
		t.Fatalf("date %q", d.AvailabilityDate)
	}
	wt := RenderInventoryContentDisplay(d)
	if !strings.Contains(wt, "|type=Avatar\n") || !strings.Contains(wt, "|price=500\n") {
		t.Fatal(wt)
	}
}

func TestRenderAvatarStylesIndex(t *testing.T) {
	out := RenderAvatarStylesIndexWikitext(2026, []string{"Anime", "Furry"})
	if !strings.Contains(out, "{{VRChatStoreListings/2026/Avatars/Authors}}") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "{{VRChatStoreListings/2026/Avatars/Anime}}") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "{{VRChatStoreListings/2026/Avatars/Furry}}") {
		t.Fatal(out)
	}
}

func TestRenderAvatarAuthorsIndex(t *testing.T) {
	out := RenderAvatarAuthorsIndexWikitext(2026, []string{"Alice", "Bob"})
	if !strings.Contains(out, "{{VRChatStoreListings/2026/Avatars/Authors/Alice}}") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "{{VRChatStoreListings/2026/Avatars/Authors/Bob}}") {
		t.Fatal(out)
	}
}

func TestBuildAvatarMarketplaceListingsCollapseAndStyle(t *testing.T) {
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	avatars := []map[string]any{
		{
			"id": "avtr_a", "name": "Wolf A", "authorName": "PackAuthor", "description": "a",
			"lowestPrice": 100.0, "listingDate": "2026-01-02T00:00:00Z",
			"styles": map[string]any{"primary": "Furry"},
			"publishedListings": []any{
				map[string]any{
					"listingId": "prod_bundle", "displayName": "Wolf Pack", "description": "bundle desc",
					"priceTokens": 5000.0, "imageId": "file_abc", "listingType": "permanent",
				},
			},
		},
		{
			"id": "avtr_b", "name": "Wolf B", "authorName": "PackAuthor", "description": "b",
			"lowestPrice": 100.0, "listingDate": "2026-01-01T00:00:00Z",
			"styles": map[string]any{"primary": "Furry"},
			"publishedListings": []any{
				map[string]any{
					"listingId": "prod_bundle", "displayName": "Wolf Pack",
					"priceTokens": 5000.0, "listingType": "permanent",
				},
			},
		},
		{
			"id": "avtr_c", "name": "Solo Anime", "authorName": "SoloAuthor", "description": "solo",
			"lowestPrice": 800.0, "listingDate": "2026-03-01T00:00:00Z",
			"styles": map[string]any{"primary": "Anime"},
			"publishedListings": []any{
				map[string]any{
					"listingId": "prod_solo", "displayName": "Solo Listing",
					"priceTokens": 900.0, "listingType": "permanent",
				},
			},
		},
		{
			"id": "avtr_d", "name": "Cross Style", "authorName": "PackAuthor", "description": "d",
			"lowestPrice": 100.0, "listingDate": "2026-01-03T00:00:00Z",
			"styles": map[string]any{"primary": "Anime"},
			"publishedListings": []any{
				map[string]any{
					"listingId": "prod_bundle", "displayName": "Wolf Pack",
					"priceTokens": 5000.0, "listingType": "permanent",
				},
			},
		},
	}
	got := BuildAvatarMarketplaceListings(avatars, now)
	if len(got) != 2 {
		t.Fatalf("listings %d: %+v", len(got), got)
	}
	byID := map[string]AvatarMarketplaceListing{}
	for _, L := range got {
		byID[L.ListingID] = L
	}
	bundle := byID["prod_bundle"]
	if bundle.Display.Name != "Wolf Pack" || bundle.Display.Price != 5000 || bundle.Display.Type != "Bundle" {
		t.Fatalf("bundle %+v", bundle.Display)
	}
	if bundle.Display.AvatarID != "" {
		t.Fatalf("bundle should omit avatarID: %+v", bundle.Display)
	}
	if bundle.Display.ProductID != "prod_bundle" || bundle.ImageID != "file_abc" {
		t.Fatalf("bundle ids %+v image %q", bundle.Display, bundle.ImageID)
	}
	if bundle.Style != "Furry" { // 2 Furry vs 1 Anime
		t.Fatalf("majority style %q", bundle.Style)
	}
	if bundle.Author != "PackAuthor" {
		t.Fatalf("author %q", bundle.Author)
	}
	if bundle.Display.AvailabilityDate != "January 1, 2026 - ???" {
		t.Fatalf("earliest date %q", bundle.Display.AvailabilityDate)
	}
	solo := byID["prod_solo"]
	if solo.Display.Type != "Avatar" || solo.Display.AvatarID != "avtr_c" || solo.Style != "Anime" {
		t.Fatalf("solo %+v style %q", solo.Display, solo.Style)
	}
	wt := RenderInventoryContentDisplay(bundle.Display)
	if strings.Contains(wt, "|type=") {
		t.Fatalf("bundle should omit type: %s", wt)
	}
}

func TestCurrentOffersWikitext(t *testing.T) {
	out := RenderCurrentOffersWikitext(2026, []struct{ Title, IconFile string }{
		{Title: "New Accessories", IconFile: "ShelfIcon New Accessories.png"},
		{Title: `{"MinVersion":1878} Open Beta Profile Decorations`, IconFile: ""},
	})
	if !strings.Contains(out, "{{VRChatStoreListings/2026/New Accessories}}") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "{{VRChatStoreListings/2026/Open Beta Profile Decorations}}") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "[[File:ShelfIcon New Accessories.png|frameless|middle|link=|30px]]") {
		t.Fatal(out)
	}
}
