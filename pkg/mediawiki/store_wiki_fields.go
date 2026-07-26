package mediawiki

import (
	"regexp"
	"strings"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

var (
	inventoryFieldRe = regexp.MustCompile(`(?m)^\s*\|([^=]+)=(.*)$`)
	currentOfferRe   = regexp.MustCompile(`\{\{VRChatStoreListings/(\d+)/([^}]+)\}\}`)
	shelfIconFileRe  = regexp.MustCompile(`\[\[File:(ShelfIcon [^|\]]+)`)
)

type inventoryWikiCard struct {
	Name             string
	ProductIDs       []string
	AvailabilityDate string
	PriceText        string
	Image            string
}

type inventoryWikiIndex struct {
	byID   map[string]*inventoryWikiCard
	byName map[string]*inventoryWikiCard
}

func parseInventoryWikiIndex(wikitext string) inventoryWikiIndex {
	idx := inventoryWikiIndex{
		byID:   map[string]*inventoryWikiCard{},
		byName: map[string]*inventoryWikiCard{},
	}
	nameCount := map[string]int{}
	var cards []*inventoryWikiCard

	rest := wikitext
	for {
		body, next, ok := nextInventoryContentDisplay(rest)
		if !ok {
			break
		}
		rest = next
		fields := map[string]string{}
		if strings.HasPrefix(body, "name=") {
			if i := strings.IndexByte(body, '\n'); i >= 0 {
				fields["name"] = strings.TrimSpace(body[len("name="):i])
				body = body[i+1:]
			} else {
				fields["name"] = strings.TrimSpace(body[len("name="):])
				body = ""
			}
		}
		for _, fm := range inventoryFieldRe.FindAllStringSubmatch(body, -1) {
			fields[strings.TrimSpace(fm[1])] = strings.TrimSpace(fm[2])
		}
		ids := splitProductIDs(fields["productID"])
		if len(ids) == 0 {
			continue
		}
		card := &inventoryWikiCard{
			Name:             strings.TrimSpace(fields["name"]),
			ProductIDs:       ids,
			AvailabilityDate: fields["availabilityDate"],
			Image:            fields["image"],
		}
		if p := fields["price"]; p != "" && p != "Free" && !isAllDigits(p) {
			card.PriceText = p
		}
		cards = append(cards, card)
		for _, id := range ids {
			idx.byID[id] = card
		}
		if card.Name != "" {
			nameCount[normalizeCardName(card.Name)]++
		}
	}
	for _, card := range cards {
		if card.Name == "" {
			continue
		}
		key := normalizeCardName(card.Name)
		if nameCount[key] == 1 {
			idx.byName[key] = card
		}
	}
	return idx
}

func parseInventoryWikiFields(wikitext string) map[string]inventoryWikiFields {
	idx := parseInventoryWikiIndex(wikitext)
	out := map[string]inventoryWikiFields{}
	seen := map[*inventoryWikiCard]bool{}
	for _, card := range idx.byID {
		if seen[card] {
			continue
		}
		seen[card] = true
		f := inventoryWikiFields{
			AvailabilityDate: card.AvailabilityDate,
			PriceText:        card.PriceText,
			Image:            card.Image,
		}
		for _, id := range card.ProductIDs {
			out[id] = f
		}
	}
	return out
}

type inventoryWikiFields struct {
	AvailabilityDate string
	PriceText        string
	Image            string
}

func lookupInventoryWikiCard(idx inventoryWikiIndex, apiID, name string) *inventoryWikiCard {
	if apiID != "" {
		if c := idx.byID[apiID]; c != nil {
			return c
		}
	}
	if name != "" {
		return idx.byName[normalizeCardName(name)]
	}
	return nil
}

func splitProductIDs(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func joinProductIDs(existing []string, apiID string) []string {
	apiID = strings.TrimSpace(apiID)
	if apiID == "" {
		return append([]string(nil), existing...)
	}
	out := make([]string, 0, len(existing)+1)
	seen := map[string]bool{}
	for _, id := range existing {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if !seen[apiID] {
		out = append(out, apiID)
	}
	return out
}

func formatProductIDs(ids []string) string {
	return strings.Join(ids, ",")
}

func normalizeCardName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func applyWikiCardMerge(card *vrchat.InventoryContentDisplay, old *inventoryWikiCard, apiID string) {
	if card == nil {
		return
	}
	var ids []string
	if old != nil {
		ids = joinProductIDs(old.ProductIDs, apiID)
	} else if strings.TrimSpace(apiID) != "" {
		ids = []string{strings.TrimSpace(apiID)}
	}
	if len(ids) > 0 {
		card.ProductID = formatProductIDs(ids)
	}
	if old == nil {
		return
	}
	if old.PriceText != "" {
		card.PriceText = old.PriceText
	}
	if old.AvailabilityDate == "" {
		return
	}
	if len(ids) > 1 || len(old.ProductIDs) == 1 {
		card.AvailabilityDate = old.AvailabilityDate
	}
}

func nextInventoryContentDisplay(s string) (body, rest string, ok bool) {
	const marker = "{{InventoryContentDisplay"
	i := strings.Index(s, marker)
	if i < 0 {
		return "", s, false
	}
	j := i + len(marker)
	for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
		j++
	}
	if j >= len(s) || s[j] != '|' {
		return nextInventoryContentDisplay(s[j:])
	}
	start := j + 1
	depth := 1
	for k := j; k < len(s)-1; k++ {
		if s[k] == '{' && s[k+1] == '{' {
			depth++
			k++
			continue
		}
		if s[k] == '}' && s[k+1] == '}' {
			depth--
			k++
			if depth == 0 {
				return s[start : k-1], s[k+1:], true
			}
		}
	}
	return "", s, false
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func parseCurrentOfferShelves(wikitext string) []string {
	var titles []string
	seen := map[string]bool{}
	for _, m := range currentOfferRe.FindAllStringSubmatch(wikitext, -1) {
		title := strings.TrimSpace(m[2])
		if title == "" || seen[title] {
			continue
		}
		seen[title] = true
		titles = append(titles, title)
	}
	return titles
}

func parseShelfIconFromHeader(wikitext string) string {
	if m := shelfIconFileRe.FindStringSubmatch(wikitext); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func parseProductIDs(wikitext string) []string {
	idx := parseInventoryWikiIndex(wikitext)
	ids := make([]string, 0, len(idx.byID))
	for id := range idx.byID {
		ids = append(ids, id)
	}
	return ids
}

func jaccard(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	setA := map[string]bool{}
	for _, x := range a {
		setA[x] = true
	}
	inter := 0
	setB := map[string]bool{}
	for _, x := range b {
		setB[x] = true
		if setA[x] {
			inter++
		}
	}
	union := len(setA)
	for x := range setB {
		if !setA[x] {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
