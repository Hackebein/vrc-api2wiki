package mediawiki

import (
	"testing"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func TestParseInventoryWikiFields(t *testing.T) {
	wt := `<noinclude>x</noinclude>
{{InventoryContentDisplay|name=Vega’s Tiara
|image=ListingImage Vega’s Tiara.jpg
|seller=VRChat
|type=Accessory
|price=1000
|price_vrcplus=900
|availabilityDate=July 3, 2026 - September 4, 2026
|description=A tiara
|productID=prod_d7a0343f-d965-4d03-9dba-a7fb5603b67f
}}
{{InventoryContentDisplay
|name=VRC+ Credits Drop
|image=ListingImage VRC+ Credit Drop.jpg
|seller=VRChat
|price=Free (VRC+ Exclusive {{VRC+}})
|availabilityDate=June 26, 2026 - July 30, 2026
|description=Free
|productID=prod_71ff701a-2ae2-45d2-8300-2e71601eaa35
}}
`
	got := parseInventoryWikiFields(wt)
	vega := got["prod_d7a0343f-d965-4d03-9dba-a7fb5603b67f"]
	if vega.AvailabilityDate != "July 3, 2026 - September 4, 2026" {
		t.Fatalf("vega date %q", vega.AvailabilityDate)
	}
	if vega.Image != "ListingImage Vega’s Tiara.jpg" {
		t.Fatalf("vega image %q", vega.Image)
	}
	if vega.PriceText != "" {
		t.Fatalf("vega price text %q", vega.PriceText)
	}
	cred := got["prod_71ff701a-2ae2-45d2-8300-2e71601eaa35"]
	if cred.PriceText != "Free (VRC+ Exclusive {{VRC+}})" {
		t.Fatalf("credit price %q", cred.PriceText)
	}
}

func TestParseMultiProductIDAndNameLookup(t *testing.T) {
	wt := `{{InventoryContentDisplay|name=Reference Cube Teal
|image=ListingImage Reference Cube Teal.png
|seller=VRChat
|price=Free
|availabilityDate=July 22, 2026 - ???
|description=d
|productID=prod_old,prod_mid
}}
`
	idx := parseInventoryWikiIndex(wt)
	if c := lookupInventoryWikiCard(idx, "prod_mid", ""); c == nil || c.Name != "Reference Cube Teal" {
		t.Fatalf("lookup by mid id: %+v", c)
	}
	if c := lookupInventoryWikiCard(idx, "prod_new", "Reference Cube Teal"); c == nil || len(c.ProductIDs) != 2 {
		t.Fatalf("lookup by name: %+v", c)
	}
	ids := parseProductIDs(wt)
	if len(ids) != 2 {
		t.Fatalf("parseProductIDs %v", ids)
	}
}

func TestJoinProductIDs(t *testing.T) {
	got := joinProductIDs([]string{"prod_a"}, "prod_b")
	if formatProductIDs(got) != "prod_a,prod_b" {
		t.Fatal(got)
	}
	got = joinProductIDs([]string{"prod_a", "prod_b"}, "prod_b")
	if formatProductIDs(got) != "prod_a,prod_b" {
		t.Fatal(got)
	}
}

func TestApplyWikiCardMergeIDChangeFreezesDate(t *testing.T) {
	old := &inventoryWikiCard{
		Name:             "Reference Cube Teal",
		ProductIDs:       []string{"prod_old"},
		AvailabilityDate: "July 22, 2026 - ???",
		PriceText:        "Free (special)",
	}
	card := vrchat.InventoryContentDisplay{
		Name:             "Reference Cube Teal",
		ProductID:        "prod_new",
		AvailabilityDate: "July 26, 2026",
		PriceText:        "",
	}
	applyWikiCardMerge(&card, old, "prod_new")
	if card.ProductID != "prod_old,prod_new" {
		t.Fatalf("productID %q", card.ProductID)
	}
	if card.AvailabilityDate != "July 22, 2026 - ???" {
		t.Fatalf("date should freeze, got %q", card.AvailabilityDate)
	}
	if card.PriceText != "Free (special)" {
		t.Fatalf("price text %q", card.PriceText)
	}
}

func TestApplyWikiCardMergeMultiIDKeepsDate(t *testing.T) {
	old := &inventoryWikiCard{
		ProductIDs:       []string{"prod_a", "prod_b"},
		AvailabilityDate: "July 22, 2026 - ???",
	}
	card := vrchat.InventoryContentDisplay{
		ProductID:        "prod_b",
		AvailabilityDate: "August 1, 2026",
	}
	applyWikiCardMerge(&card, old, "prod_b")
	if card.ProductID != "prod_a,prod_b" {
		t.Fatalf("%q", card.ProductID)
	}
	if card.AvailabilityDate != "July 22, 2026 - ???" {
		t.Fatalf("%q", card.AvailabilityDate)
	}
}

func TestApplyWikiCardMergeSingleIDPreservesDate(t *testing.T) {
	old := &inventoryWikiCard{
		ProductIDs:       []string{"prod_a"},
		AvailabilityDate: "June 19, 2026",
	}
	card := vrchat.InventoryContentDisplay{
		ProductID:        "prod_a",
		AvailabilityDate: "July 26, 2026",
	}
	applyWikiCardMerge(&card, old, "prod_a")
	if card.ProductID != "prod_a" {
		t.Fatalf("%q", card.ProductID)
	}
	if card.AvailabilityDate != "June 19, 2026" {
		t.Fatalf("%q", card.AvailabilityDate)
	}
}

func TestApplyWikiCardMergeNilOld(t *testing.T) {
	card := vrchat.InventoryContentDisplay{
		ProductID:        "prod_x",
		AvailabilityDate: "July 26, 2026",
	}
	applyWikiCardMerge(&card, nil, "prod_x")
	if card.ProductID != "prod_x" || card.AvailabilityDate != "July 26, 2026" {
		t.Fatalf("%+v", card)
	}
}

func TestResolveWikiShelfTitle(t *testing.T) {
	wiki := []wikiShelfInfo{{
		Title:      "Vket Summer 2026",
		ProductIDs: []string{"prod_a", "prod_b", "prod_c"},
	}}
	got := resolveWikiShelfTitle("Vket 2026 Summer", []string{"prod_a", "prod_b", "prod_c", "prod_d"}, wiki)
	if got != "Vket Summer 2026" {
		t.Fatalf("got %q", got)
	}
	got = resolveWikiShelfTitle("Brand New Shelf", []string{"prod_z"}, wiki)
	if got != "Brand New Shelf" {
		t.Fatalf("got %q", got)
	}
}

func TestParseCurrentOfferShelves(t *testing.T) {
	wt := `{{VRChatStoreListings/2026/Open Beta Profile Decorations}}
{{VRChatStoreListings/2026/Vket Summer 2026}}
`
	got := parseCurrentOfferShelves(wt)
	if len(got) != 2 || got[0] != "Open Beta Profile Decorations" || got[1] != "Vket Summer 2026" {
		t.Fatalf("%v", got)
	}
}
