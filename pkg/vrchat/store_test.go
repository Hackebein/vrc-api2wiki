package vrchat

import "testing"

func TestStoreShelvesFollowsShelfIDs(t *testing.T) {
	a := map[string]any{"id": "ess_a", "shelfTitle": "A"}
	b := map[string]any{"id": "ess_b", "shelfTitle": "B"}
	c := map[string]any{"id": "ess_c", "shelfTitle": "C"}
	store := map[string]any{
		"shelfIds": []any{"ess_c", "ess_a", "ess_b"},
		"shelves":  []any{a, b, c},
	}
	got := StoreShelves(store)
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	if stringField(got[0], "id") != "ess_c" || stringField(got[1], "id") != "ess_a" || stringField(got[2], "id") != "ess_b" {
		t.Fatalf("order %q %q %q", stringField(got[0], "id"), stringField(got[1], "id"), stringField(got[2], "id"))
	}
}

func TestStoreShelvesAppendsLeftovers(t *testing.T) {
	a := map[string]any{"id": "ess_a"}
	b := map[string]any{"id": "ess_b"}
	c := map[string]any{"id": "ess_c"}
	store := map[string]any{
		"shelfIds": []any{"ess_b"},
		"shelves":  []any{a, b, c},
	}
	got := StoreShelves(store)
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	if stringField(got[0], "id") != "ess_b" || stringField(got[1], "id") != "ess_a" || stringField(got[2], "id") != "ess_c" {
		t.Fatalf("order %q %q %q", stringField(got[0], "id"), stringField(got[1], "id"), stringField(got[2], "id"))
	}
}

func TestStoreShelvesWithoutShelfIDsKeepsArrayOrder(t *testing.T) {
	a := map[string]any{"id": "ess_a"}
	b := map[string]any{"id": "ess_b"}
	store := map[string]any{
		"shelves": []any{a, b},
	}
	got := StoreShelves(store)
	if len(got) != 2 || stringField(got[0], "id") != "ess_a" || stringField(got[1], "id") != "ess_b" {
		t.Fatalf("%v", idsOf(got))
	}
}

func TestShelfListingsFollowsListingIDs(t *testing.T) {
	l1 := map[string]any{"id": "prod_1", "displayName": "One"}
	l2 := map[string]any{"id": "prod_2", "displayName": "Two"}
	l3 := map[string]any{"id": "prod_3", "displayName": "Three"}
	shelf := map[string]any{
		"listingIds": []any{"prod_3", "prod_1", "prod_2"},
		"listings":   []any{l1, l2, l3},
	}
	got := ShelfListings(shelf)
	if idsOf(got) != "prod_3,prod_1,prod_2" {
		t.Fatalf("%s", idsOf(got))
	}
}

func TestShelfListingsDedupesHighlight(t *testing.T) {
	hl := map[string]any{"id": "prod_1", "displayName": "Highlight"}
	l1 := map[string]any{"id": "prod_1", "displayName": "One"}
	l2 := map[string]any{"id": "prod_2", "displayName": "Two"}
	shelf := map[string]any{
		"listingIds":       []any{"prod_2", "prod_1"},
		"highlightListing": hl,
		"listings":         []any{l1, l2},
	}
	got := ShelfListings(shelf)
	if idsOf(got) != "prod_2,prod_1" {
		t.Fatalf("%s", idsOf(got))
	}
}

func TestShelfListingsWithoutListingIDsKeepsArrayOrderNoDup(t *testing.T) {
	hl := map[string]any{"id": "prod_1"}
	l1 := map[string]any{"id": "prod_1"}
	l2 := map[string]any{"id": "prod_2"}
	shelf := map[string]any{
		"highlightListing": hl,
		"listings":         []any{l1, l2},
	}
	got := ShelfListings(shelf)
	if idsOf(got) != "prod_1,prod_2" {
		t.Fatalf("%s", idsOf(got))
	}
}

func TestShelfListingsAppendsLeftovers(t *testing.T) {
	l1 := map[string]any{"id": "prod_1"}
	l2 := map[string]any{"id": "prod_2"}
	l3 := map[string]any{"id": "prod_3"}
	shelf := map[string]any{
		"listingIds": []any{"prod_2"},
		"listings":   []any{l1, l2, l3},
	}
	got := ShelfListings(shelf)
	if idsOf(got) != "prod_2,prod_1,prod_3" {
		t.Fatalf("%s", idsOf(got))
	}
}

func idsOf(items []map[string]any) string {
	out := make([]byte, 0, 32)
	for i, m := range items {
		if i > 0 {
			out = append(out, ',')
		}
		out = append(out, stringField(m, "id")...)
	}
	return string(out)
}
