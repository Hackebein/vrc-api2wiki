package vrchat

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type InventoryContentDisplay struct {
	Name             string
	Image            string
	Seller           string
	Publisher        string
	Author           string
	Type             string
	Price            int
	PriceVRCPlus     int
	PriceText        string
	AvailabilityDate string
	Description      string
	ProductID        string
	AvatarID         string
}

var minVersionShelfPrefix = regexp.MustCompile(`^\s*\{[^}]*\}\s*`)

func DisplayShelfTitle(raw string) string {
	return strings.TrimSpace(minVersionShelfPrefix.ReplaceAllString(raw, ""))
}

func ListingImageFilename(displayName, ext string) string {
	if ext == "" {
		ext = "png"
	}
	return fmt.Sprintf("ListingImage %s.%s", strings.TrimSpace(displayName), ext)
}

func ShelfIconFilename(shelfTitle, ext string) string {
	if ext == "" {
		ext = "png"
	}
	return fmt.Sprintf("ShelfIcon %s.%s", DisplayShelfTitle(shelfTitle), ext)
}

func ProductImageFilename(productTypeLabel, displayName, ext string) string {
	if ext == "" {
		ext = "png"
	}
	label := strings.ReplaceAll(strings.TrimSpace(productTypeLabel), " ", "")
	if label == "" {
		label = "Product"
	}
	return fmt.Sprintf("%s %s.%s", label, strings.TrimSpace(displayName), ext)
}

func RenderInventoryContentDisplay(d InventoryContentDisplay) string {
	var b strings.Builder
	b.WriteString("{{InventoryContentDisplay|name=" + d.Name + "\n")
	if d.Image != "" {
		b.WriteString("|image=" + d.Image + "\n")
	}
	if d.Publisher != "" && d.Author != "" && d.Author != "VRChat" {
		b.WriteString("|publisher=" + d.Publisher + "\n")
		b.WriteString("|author=" + d.Author + "\n")
	} else {
		seller := d.Seller
		if seller == "" {
			seller = "VRChat"
		}
		b.WriteString("|seller=" + seller + "\n")
	}
	if d.Type != "" && d.Type != "Bundle" && d.Type != "CreditBundle" {
		b.WriteString("|type=" + d.Type + "\n")
	}
	if d.PriceText != "" {
		b.WriteString("|price=" + d.PriceText + "\n")
	} else if d.Price <= 0 {
		b.WriteString("|price=Free\n")
	} else {
		b.WriteString("|price=" + strconv.Itoa(d.Price) + "\n")
		if d.PriceVRCPlus > 0 {
			b.WriteString("|price_vrcplus=" + strconv.Itoa(d.PriceVRCPlus) + "\n")
		}
	}
	if d.AvailabilityDate != "" {
		b.WriteString("|availabilityDate=" + d.AvailabilityDate + "\n")
	}
	b.WriteString("|description=" + d.Description + "\n")
	b.WriteString("|productID=" + d.ProductID + "\n")
	if d.AvatarID != "" {
		b.WriteString("|avatarID=" + d.AvatarID + "\n")
	}
	b.WriteString("}}\n")
	return b.String()
}

func ListingToDisplay(listing map[string]any, now time.Time, imageFilename string) InventoryContentDisplay {
	name := strings.TrimSpace(stringField(listing, "displayName"))
	if imageFilename == "" {
		imageFilename = ListingImageFilename(name, "png")
	}
	d := InventoryContentDisplay{
		Name:         name,
		Image:        imageFilename,
		Type:         normalizeType(ListingTypeLabel(listing)),
		Price:        intField(listing, "priceTokens"),
		PriceVRCPlus: intField(listing, "vrcPlusDiscountPrice"),
		Description:  stringField(listing, "description"),
		ProductID:    stringField(listing, "id"),
		Seller:       "VRChat",
	}
	if d.Type == "Bundle" || d.Type == "CreditBundle" {
		d.Type = ""
	}
	author, publisher := listingAttribution(listing)
	d.Author = author
	d.Publisher = publisher

	start := listingStartDate(listing, now)
	if exp := stringField(listing, "whenToExpire"); exp != "" {
		end := "???"
		if t, err := time.Parse(time.RFC3339, exp); err == nil {
			end = formatWikiDate(t)
		} else if len(exp) >= 10 {
			if t, err := time.Parse("2006-01-02", exp[:10]); err == nil {
				end = formatWikiDate(t)
			}
		}
		d.AvailabilityDate = formatWikiDate(start) + " - " + end
		if d.Price <= 0 && strings.Contains(strings.ToUpper(name), "VRC+") {
			d.PriceText = "Free (VRC+ Exclusive {{VRC+}})"
		}
	} else {
		d.AvailabilityDate = formatWikiDate(start)
	}

	if id := stringField(listing, "avatarId"); id != "" {
		d.AvatarID = id
	}
	return d
}

func AvatarPrimaryStyle(av map[string]any) string {
	styles, _ := av["styles"].(map[string]any)
	if styles != nil {
		if p := strings.TrimSpace(stringField(styles, "primary")); p != "" {
			return p
		}
	}
	return "Uncategorized"
}

func AvatarToDisplay(av map[string]any, now time.Time, imageFilename string) InventoryContentDisplay {
	name := strings.TrimSpace(stringField(av, "name"))
	if name == "" {
		name = strings.TrimSpace(stringField(av, "displayName"))
	}
	avatarID := stringField(av, "id")
	productID := stringField(av, "productId")
	if productID == "" {
		productID = avatarID
	}
	if imageFilename == "" {
		imageFilename = ListingImageFilename(name, "png")
	}
	d := InventoryContentDisplay{
		Name:        name,
		Image:       imageFilename,
		Seller:      stringField(av, "authorName"),
		Type:        "Avatar",
		Description: stringField(av, "description"),
		ProductID:   productID,
		AvatarID:    avatarID,
		Price:       intField(av, "lowestPrice"),
	}
	if d.Seller == "" {
		d.Seller = "VRChat"
	}
	if t, err := time.Parse(time.RFC3339, stringField(av, "listingDate")); err == nil {
		d.AvailabilityDate = formatWikiDate(t) + " - ???"
	} else {
		d.AvailabilityDate = formatWikiDate(now) + " - ???"
	}
	return d
}

type AvatarMarketplaceListing struct {
	ListingID      string
	Style          string
	Author         string
	ImageID        string
	FallbackAvatar map[string]any
	Display        InventoryContentDisplay
}

type avatarListingAccum struct {
	listingID   string
	displayName string
	description string
	priceTokens int
	imageID     string
	members     []map[string]any
	memberIDs   map[string]struct{}
	styleCounts map[string]int
}

func BuildAvatarMarketplaceListings(avatars []map[string]any, now time.Time) []AvatarMarketplaceListing {
	byID := map[string]*avatarListingAccum{}
	order := make([]string, 0)
	for _, av := range avatars {
		if av == nil {
			continue
		}
		style := AvatarPrimaryStyle(av)
		listings, _ := av["publishedListings"].([]any)
		for _, raw := range listings {
			pl, _ := raw.(map[string]any)
			if pl == nil {
				continue
			}
			listingID := strings.TrimSpace(stringField(pl, "listingId"))
			if listingID == "" {
				continue
			}
			acc := byID[listingID]
			if acc == nil {
				acc = &avatarListingAccum{
					listingID:   listingID,
					memberIDs:   map[string]struct{}{},
					styleCounts: map[string]int{},
				}
				byID[listingID] = acc
				order = append(order, listingID)
			}
			if acc.displayName == "" {
				acc.displayName = strings.TrimSpace(stringField(pl, "displayName"))
			}
			if acc.description == "" {
				acc.description = strings.TrimSpace(stringField(pl, "description"))
			}
			if acc.imageID == "" {
				acc.imageID = strings.TrimSpace(stringField(pl, "imageId"))
			}
			if price := intField(pl, "priceTokens"); price > 0 {
				acc.priceTokens = price
			}
			avID := stringField(av, "id")
			if avID != "" {
				if _, seen := acc.memberIDs[avID]; seen {
					continue
				}
				acc.memberIDs[avID] = struct{}{}
			}
			acc.members = append(acc.members, av)
			acc.styleCounts[style]++
		}
	}

	out := make([]AvatarMarketplaceListing, 0, len(order))
	for _, listingID := range order {
		acc := byID[listingID]
		if len(acc.members) == 0 {
			continue
		}
		first := acc.members[0]
		name := acc.displayName
		if name == "" {
			name = strings.TrimSpace(stringField(first, "name"))
			if name == "" {
				name = strings.TrimSpace(stringField(first, "displayName"))
			}
		}
		desc := acc.description
		if desc == "" {
			desc = stringField(first, "description")
		}
		author := strings.TrimSpace(stringField(first, "authorName"))
		if author == "" {
			author = "VRChat"
		}
		price := acc.priceTokens
		if price <= 0 {
			price = intField(first, "lowestPrice")
		}
		cardType := "Avatar"
		avatarID := ""
		if len(acc.members) == 1 {
			avatarID = stringField(first, "id")
		} else {
			cardType = "Bundle"
		}
		start := earliestAvatarListingDate(acc.members, now)
		d := InventoryContentDisplay{
			Name:             name,
			Image:            ListingImageFilename(name, "png"),
			Seller:           author,
			Type:             cardType,
			Price:            price,
			Description:      desc,
			ProductID:        listingID,
			AvatarID:         avatarID,
			AvailabilityDate: formatWikiDate(start) + " - ???",
		}
		out = append(out, AvatarMarketplaceListing{
			ListingID:      listingID,
			Style:          majorityStyle(acc.styleCounts),
			Author:         author,
			ImageID:        acc.imageID,
			FallbackAvatar: first,
			Display:        d,
		})
	}
	return out
}

func majorityStyle(counts map[string]int) string {
	best := "Uncategorized"
	bestN := -1
	for style, n := range counts {
		if n > bestN || (n == bestN && style < best) {
			best = style
			bestN = n
		}
	}
	if bestN < 0 {
		return "Uncategorized"
	}
	return best
}

func earliestAvatarListingDate(members []map[string]any, now time.Time) time.Time {
	earliest := time.Time{}
	for _, av := range members {
		if t, err := time.Parse(time.RFC3339, stringField(av, "listingDate")); err == nil {
			if earliest.IsZero() || t.Before(earliest) {
				earliest = t
			}
		}
	}
	if earliest.IsZero() {
		return now
	}
	return earliest
}

func listingAttribution(listing map[string]any) (author, publisher string) {
	products, _ := listing["products"].([]any)
	if len(products) == 0 {
		products, _ = listing["hydratedProducts"].([]any)
	}
	if len(products) == 0 {
		return "", ""
	}
	prod, _ := products[0].(map[string]any)
	if prod == nil {
		return "", ""
	}
	attr, _ := prod["attribution"].(map[string]any)
	if attr == nil {
		attr, _ = listing["attribution"].(map[string]any)
	}
	if attr == nil {
		return "", ""
	}
	if c, ok := attr["creator"].(map[string]any); ok {
		author = stringField(c, "customName")
	}
	if p, ok := attr["publisher"].(map[string]any); ok {
		publisher = stringField(p, "customName")
	}
	return author, publisher
}

func listingStartDate(listing map[string]any, now time.Time) time.Time {
	for _, key := range []string{"created", "updated"} {
		if s := stringField(listing, key); s != "" {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t
			}
			if len(s) >= 10 {
				if t, err := time.Parse("2006-01-02", s[:10]); err == nil {
					return t
				}
			}
		}
	}
	return now
}

func formatWikiDate(t time.Time) string {
	return t.Format("January ") + strconv.Itoa(t.Day()) + t.Format(", 2006")
}

func normalizeType(t string) string {
	switch strings.ToLower(t) {
	case "loadingscreen", "loading screen":
		return "LoadingScreen"
	case "warpeffect", "warp effect":
		return "WarpEffect"
	case "creditbundle", "credit bundle":
		return "CreditBundle"
	case "prop":
		return "Item"
	default:
		return t
	}
}

func intField(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}

func RenderShelfWikitext(shelfTitle string, shelfIconFile string, cards []InventoryContentDisplay) string {
	title := DisplayShelfTitle(shelfTitle)
	var b strings.Builder
	b.WriteString("<noinclude><span style=\"font-size:180%\">")
	if shelfIconFile != "" {
		b.WriteString("[[File:" + shelfIconFile + "|frameless|middle|link=|30px]] ")
	}
	b.WriteString(title + "</span></noinclude>\n")
	b.WriteString("<div style=\"display: inline-block; flex-wrap: wrap;\">\n")
	for _, card := range cards {
		b.WriteString(RenderInventoryContentDisplay(card))
	}
	b.WriteString("</div><noinclude>[[Category:Shop templates]]</noinclude>\n")
	return b.String()
}

func RenderCurrentOffersWikitext(year int, shelves []struct{ Title, IconFile string }) string {
	var b strings.Builder
	for _, s := range shelves {
		title := DisplayShelfTitle(s.Title)
		b.WriteString("<span style=\"font-size:180%\">")
		if s.IconFile != "" {
			b.WriteString("[[File:" + s.IconFile + "|frameless|middle|link=|30px]] ")
		}
		b.WriteString(title + "</span>\n")
		b.WriteString(fmt.Sprintf("{{VRChatStoreListings/%d/%s}}\n\n", year, title))
	}
	b.WriteString("<noinclude>[[Category:Shop templates]]</noinclude>\n")
	return b.String()
}

func RenderAvatarStylesIndexWikitext(year int, styles []string) string {
	var b strings.Builder
	b.WriteString("<span style=\"font-size:180%\">Authors</span>\n")
	b.WriteString(fmt.Sprintf("{{VRChatStoreListings/%d/Avatars/Authors}}\n\n", year))
	for _, style := range styles {
		b.WriteString(fmt.Sprintf("<span style=\"font-size:180%%\">%s</span>\n", style))
		b.WriteString(fmt.Sprintf("{{VRChatStoreListings/%d/Avatars/%s}}\n\n", year, style))
	}
	b.WriteString("<noinclude>[[Category:Shop templates]]</noinclude>\n")
	return b.String()
}

func RenderAvatarAuthorsIndexWikitext(year int, authors []string) string {
	var b strings.Builder
	for _, author := range authors {
		b.WriteString(fmt.Sprintf("<span style=\"font-size:180%%\">%s</span>\n", author))
		b.WriteString(fmt.Sprintf("{{VRChatStoreListings/%d/Avatars/Authors/%s}}\n\n", year, author))
	}
	b.WriteString("<noinclude>[[Category:Shop templates]]</noinclude>\n")
	return b.String()
}
