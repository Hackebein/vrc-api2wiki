package mediawiki

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func StoreListingsPageTitle(year int, shelfTitle string) string {
	return fmt.Sprintf("Template:VRChatStoreListings/%d/%s", year, storeShelfPathSegment(shelfTitle))
}

func StoreCurrentOffersPageTitle() string {
	return "Template:VRChatStoreCurrentOffers"
}

func StoreAvatarStylePageTitle(year int, style string) string {
	return fmt.Sprintf("Template:VRChatStoreListings/%d/Avatars/%s", year, storeShelfPathSegment(style))
}

func StoreAvatarStylesIndexPageTitle(year int) string {
	return fmt.Sprintf("Template:VRChatStoreListings/%d/Avatars", year)
}

func StoreAvatarAuthorsIndexPageTitle(year int) string {
	return fmt.Sprintf("Template:VRChatStoreListings/%d/Avatars/Authors", year)
}

func StoreAvatarAuthorPageTitle(year int, author string) string {
	return fmt.Sprintf("Template:VRChatStoreListings/%d/Avatars/Authors/%s", year, storeShelfPathSegment(author))
}

func storeShelfPathSegment(title string) string {
	title = vrchat.DisplayShelfTitle(title)
	var b strings.Builder
	for _, r := range title {
		switch r {
		case '{', '}', '[', ']', '|', '#', '<', '>', '"', '\'', ':', '\n', '\r', '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func ListingImageFileDescription(listingName, productID string) string {
	return fileDescriptionPage(fileInformationParams{
		description:           fmt.Sprintf("Store listing image for %s.", listingName),
		source:                "VRChat API",
		author:                "VRChat",
		additionalInformation: strings.TrimSpace(productID),
	}, "{{license VRC public section8}}")
}

func ShelfIconFileDescription(shelfTitle string) string {
	return fileDescriptionPage(fileInformationParams{
		description: fmt.Sprintf("Store shelf icon for %s.", vrchat.DisplayShelfTitle(shelfTitle)),
		source:      "VRChat API",
		author:      "VRChat",
	}, "{{license VRC public section8}}")
}

type currentOfferShelf struct {
	Title    string
	IconFile string
}

func RunStoreSync(wiki *MediaWikiClient, api *vrchat.Client, logger *slog.Logger) error {
	auth, ok := vrchat.AuthConfigFromEnv()
	if !ok {
		if logger != nil {
			logger.Info("skipping marketplace sync: VRCHAT_USERNAME/PASSWORD/TOTP_SECRET not set")
		}
		return nil
	}
	if err := api.EnsureAuth(auth, logger); err != nil {
		return fmt.Errorf("vrchat auth: %w", err)
	}

	snapshotDir := filepath.Join("store-data", time.Now().UTC().Format("2006_01_02_15_04_05"))
	if logger != nil {
		logger.Info("fetching marketplace catalog", "snapshot", snapshotDir)
	}
	avatarLimit := ProcessingLimitFromEnv()
	snap, err := api.FetchStoreCatalog(snapshotDir, avatarLimit, logger)
	if err != nil {
		return err
	}
	if logger != nil {
		keys := make([]string, 0, len(snap.TypeCounts))
		for k := range snap.TypeCounts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			logger.Info("marketplace type count", "type", k, "count", snap.TypeCounts[k])
		}
	}

	now := time.Now()
	year := now.UTC().Year()
	onShelf := map[string]bool{}
	var offers []currentOfferShelf

	wikiShelves := loadWikiShelfIndex(wiki, year, logger)

	for _, shelf := range vrchat.StoreShelves(snap.Store) {
		apiTitle := stringField(shelf, "shelfTitle")
		if apiTitle == "" {
			continue
		}
		shelfListings := vrchat.ShelfListings(shelf)
		if len(shelfListings) == 0 {
			if logger != nil {
				logger.Info("skipping empty store shelf", "shelf", vrchat.DisplayShelfTitle(apiTitle))
			}
			continue
		}

		var listingIDs []string
		for _, listing := range shelfListings {
			if id := stringField(listing, "id"); id != "" {
				listingIDs = append(listingIDs, id)
			}
		}
		wikiTitle := resolveWikiShelfTitle(apiTitle, listingIDs, wikiShelves)
		pathTitle := storeShelfPathSegment(wikiTitle)
		if pathTitle == "" {
			continue
		}

		existingPage, _ := wiki.GetPageContent(StoreListingsPageTitle(year, wikiTitle))
		existingCards := parseInventoryWikiIndex(existingPage)
		existingIcon := parseShelfIconFromHeader(existingPage)

		iconFile, err := syncShelfIcon(wiki, api, shelf, wikiTitle, existingIcon, logger)
		if err != nil {
			if logger != nil {
				logger.Warn("shelf icon skipped", "shelf", pathTitle, "err", err)
			}
			iconFile = existingIcon
		}

		var cards []vrchat.InventoryContentDisplay
		for _, listing := range shelfListings {
			id := stringField(listing, "id")
			hydrated := snap.Listings[id]
			if hydrated == nil {
				hydrated = listing
			}
			name := stringField(hydrated, "displayName")
			if name == "" {
				name = stringField(listing, "displayName")
			}
			if id == "" || name == "" {
				if logger != nil {
					logger.Info("skipping empty store listing", "shelf", pathTitle, "id", id, "name", name)
				}
				continue
			}
			onShelf[id] = true
			old := lookupInventoryWikiCard(existingCards, id, name)
			preferredImage := ""
			if old != nil {
				preferredImage = old.Image
			}
			imageName, err := syncListingMedia(wiki, api, hydrated, preferredImage, logger)
			if err != nil {
				return fmt.Errorf("listing media %s: %w", id, err)
			}
			card := vrchat.ListingToDisplay(hydrated, now, imageName)
			applyWikiCardMerge(&card, old, id)
			cards = append(cards, card)
		}
		if len(cards) == 0 {
			if logger != nil {
				logger.Info("skipping empty store shelf", "shelf", pathTitle)
			}
			continue
		}
		page := vrchat.RenderShelfWikitext(wikiTitle, iconFile, cards)
		if err := wiki.EditPage(StoreListingsPageTitle(year, wikiTitle), page, true); err != nil {
			return fmt.Errorf("edit shelf %s: %w", pathTitle, err)
		}
		offers = append(offers, currentOfferShelf{Title: wikiTitle, IconFile: iconFile})
	}

	ids := make([]string, 0, len(snap.Listings))
	for id := range snap.Listings {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	orphanCount := 0
	for _, id := range ids {
		if onShelf[id] {
			continue
		}
		listing := snap.Listings[id]
		if _, err := syncListingMedia(wiki, api, listing, "", logger); err != nil {
			return fmt.Errorf("listing media %s: %w", id, err)
		}
		orphanCount++
	}

	avatarImages, avatarListings, avatarStyles, avatarAuthors, err := syncAvatarMarketplacePages(wiki, api, snap.Avatars, year, now, logger)
	if err != nil {
		return err
	}

	if len(offers) > 0 {
		offers = orderOffersLikeWiki(offers, wikiShelves)
		offerRows := make([]struct{ Title, IconFile string }, len(offers))
		for i, o := range offers {
			offerRows[i] = struct{ Title, IconFile string }{Title: o.Title, IconFile: o.IconFile}
		}
		page := vrchat.RenderCurrentOffersWikitext(year, offerRows)
		if err := wiki.EditPage(StoreCurrentOffersPageTitle(), page, true); err != nil {
			return fmt.Errorf("edit current offers: %w", err)
		}
	}

	if logger != nil {
		logger.Info("marketplace sync complete",
			"listings", len(snap.Listings),
			"shelves", len(offers),
			"orphanListings", orphanCount,
			"avatars", len(snap.Avatars),
			"avatarListings", avatarListings,
			"avatarImages", avatarImages,
			"avatarStyles", avatarStyles,
			"avatarAuthors", avatarAuthors,
			"avatarLimit", avatarLimit,
			"year", year,
			"snapshot", snapshotDir)
	}
	return nil
}

func syncAvatarMarketplacePages(wiki *MediaWikiClient, api *vrchat.Client, avatars []map[string]any, year int, now time.Time, logger *slog.Logger) (images, listings, styles, authors int, err error) {
	built := vrchat.BuildAvatarMarketplaceListings(avatars, now)
	byStyle := map[string][]vrchat.InventoryContentDisplay{}
	byAuthor := map[string][]vrchat.InventoryContentDisplay{}

	for _, listing := range built {
		filename, err := syncAvatarListingImage(wiki, api, listing, logger)
		if err != nil {
			if logger != nil {
				logger.Warn("avatar listing image skipped", "listing", listing.ListingID, "err", err)
			}
			continue
		}
		images++
		card := listing.Display
		card.Image = filename
		byStyle[listing.Style] = append(byStyle[listing.Style], card)
		author := storeShelfPathSegment(listing.Author)
		if author == "" {
			author = "VRChat"
		}
		byAuthor[author] = append(byAuthor[author], card)
	}
	listings = len(built)

	styleNames := make([]string, 0, len(byStyle))
	for style := range byStyle {
		styleNames = append(styleNames, style)
	}
	sort.Strings(styleNames)
	for _, style := range styleNames {
		cards := byStyle[style]
		sort.Slice(cards, func(i, j int) bool {
			return strings.ToLower(cards[i].Name) < strings.ToLower(cards[j].Name)
		})
		page := vrchat.RenderShelfWikitext(style, "", cards)
		if err := wiki.EditPage(StoreAvatarStylePageTitle(year, style), page, true); err != nil {
			return images, listings, 0, 0, fmt.Errorf("edit avatar style %s: %w", style, err)
		}
	}

	authorNames := make([]string, 0, len(byAuthor))
	for author := range byAuthor {
		authorNames = append(authorNames, author)
	}
	sort.Strings(authorNames)
	for _, author := range authorNames {
		cards := byAuthor[author]
		sort.Slice(cards, func(i, j int) bool {
			return strings.ToLower(cards[i].Name) < strings.ToLower(cards[j].Name)
		})
		page := vrchat.RenderShelfWikitext(author, "", cards)
		if err := wiki.EditPage(StoreAvatarAuthorPageTitle(year, author), page, true); err != nil {
			return images, listings, len(styleNames), 0, fmt.Errorf("edit avatar author %s: %w", author, err)
		}
	}

	if len(styleNames) > 0 || len(authorNames) > 0 {
		index := vrchat.RenderAvatarStylesIndexWikitext(year, styleNames)
		if err := wiki.EditPage(StoreAvatarStylesIndexPageTitle(year), index, true); err != nil {
			return images, listings, len(styleNames), len(authorNames), fmt.Errorf("edit avatar styles index: %w", err)
		}
	}
	if len(authorNames) > 0 {
		authorsIndex := vrchat.RenderAvatarAuthorsIndexWikitext(year, authorNames)
		if err := wiki.EditPage(StoreAvatarAuthorsIndexPageTitle(year), authorsIndex, true); err != nil {
			return images, listings, len(styleNames), len(authorNames), fmt.Errorf("edit avatar authors index: %w", err)
		}
	}
	return images, listings, len(styleNames), len(authorNames), nil
}

func syncAvatarListingImage(wiki *MediaWikiClient, api *vrchat.Client, listing vrchat.AvatarMarketplaceListing, logger *slog.Logger) (string, error) {
	name := listing.Display.Name
	if listing.ImageID != "" {
		return uploadNamedFile(wiki, api, listing.ImageID, func(ext string) string {
			return vrchat.ListingImageFilename(name, ext)
		}, name, listing.ListingID, logger)
	}
	return syncAvatarImage(wiki, api, listing.FallbackAvatar, logger)
}

type wikiShelfInfo struct {
	Title      string
	ProductIDs []string
	IconFile   string
}

func loadWikiShelfIndex(wiki *MediaWikiClient, year int, logger *slog.Logger) []wikiShelfInfo {
	offers, err := wiki.GetPageContent(StoreCurrentOffersPageTitle())
	if err != nil {
		if logger != nil {
			logger.Warn("could not load current offers for shelf matching", "err", err)
		}
		return nil
	}
	var out []wikiShelfInfo
	for _, title := range parseCurrentOfferShelves(offers) {
		page, err := wiki.GetPageContent(StoreListingsPageTitle(year, title))
		if err != nil {
			continue
		}
		out = append(out, wikiShelfInfo{
			Title:      title,
			ProductIDs: parseProductIDs(page),
			IconFile:   parseShelfIconFromHeader(page),
		})
	}
	return out
}

func orderOffersLikeWiki(offers []currentOfferShelf, wikiShelves []wikiShelfInfo) []currentOfferShelf {
	if len(wikiShelves) == 0 || len(offers) == 0 {
		return offers
	}
	byTitle := map[string]currentOfferShelf{}
	for _, o := range offers {
		byTitle[o.Title] = o
	}
	var ordered []currentOfferShelf
	seen := map[string]bool{}
	for _, ws := range wikiShelves {
		if o, ok := byTitle[ws.Title]; ok {
			ordered = append(ordered, o)
			seen[ws.Title] = true
		}
	}
	for _, o := range offers {
		if !seen[o.Title] {
			ordered = append(ordered, o)
		}
	}
	return ordered
}

func resolveWikiShelfTitle(apiTitle string, listingIDs []string, wikiShelves []wikiShelfInfo) string {
	display := vrchat.DisplayShelfTitle(apiTitle)
	bestTitle := display
	bestScore := 0.0
	for _, ws := range wikiShelves {
		if ws.Title == display {
			return ws.Title
		}
		score := jaccard(listingIDs, ws.ProductIDs)
		if score > bestScore {
			bestScore = score
			bestTitle = ws.Title
		}
	}
	if bestScore >= 0.5 {
		return bestTitle
	}
	return display
}

func syncShelfIcon(wiki *MediaWikiClient, api *vrchat.Client, shelf map[string]any, wikiTitle, existingIcon string, logger *slog.Logger) (string, error) {
	fileID := stringField(shelf, "shelfIconImageId")
	if fileID == "" {
		return existingIcon, nil
	}
	data, ext, err := api.DownloadFileBytes(fileID)
	if err != nil {
		return existingIcon, err
	}
	if ext == "" {
		ext = "png"
	}
	filename := vrchat.ShelfIconFilename(wikiTitle, ext)
	if existingIcon != "" {
		filename = existingIcon
	}
	desc := ShelfIconFileDescription(wikiTitle)
	uploaded, err := wiki.UploadFile(filename, data, desc)
	if err != nil {
		return "", err
	}
	if logger != nil {
		logger.Info("shelf icon processed", "filename", filename, "uploaded", uploaded)
	}
	return filename, nil
}

func syncListingMedia(wiki *MediaWikiClient, api *vrchat.Client, listing map[string]any, preferredListingImage string, logger *slog.Logger) (string, error) {
	name := strings.TrimSpace(stringField(listing, "displayName"))
	productID := stringField(listing, "id")
	var listingImage string

	if fileID := stringField(listing, "imageId"); fileID != "" {
		filename, err := uploadNamedFile(wiki, api, fileID, func(ext string) string {
			if preferredListingImage != "" {
				return preferredListingImage
			}
			return vrchat.ListingImageFilename(name, ext)
		}, name, productID, logger)
		if err != nil {
			return "", err
		}
		listingImage = filename
	} else if preferredListingImage != "" {
		listingImage = preferredListingImage
	}

	for _, prod := range vrchat.ListingProducts(listing) {
		pid := stringField(prod, "imageId")
		if pid == "" {
			continue
		}
		pname := stringField(prod, "displayName")
		if pname == "" {
			pname = name
		}
		label := stringField(prod, "productTypeLabel")
		if _, err := uploadNamedFile(wiki, api, pid, func(ext string) string {
			return vrchat.ProductImageFilename(label, pname, ext)
		}, pname, stringField(prod, "id"), logger); err != nil {
			if logger != nil {
				logger.Warn("product image skipped", "product", pname, "err", err)
			}
		}
	}

	if productID == "" {
		return listingImage, nil
	}
	gallery, err := api.FetchListingGallery(productID)
	if err != nil {
		if logger != nil {
			logger.Warn("listing gallery unavailable", "listing", productID, "err", err)
		}
		return listingImage, nil
	}
	for _, entry := range gallery {
		gid := stringField(entry, "id")
		if gid == "" {
			continue
		}
		entryName := stringField(entry, "name")
		if _, err := uploadNamedFile(wiki, api, gid, func(ext string) string {
			if entryName != "" {
				if strings.Contains(entryName, ".") {
					return entryName
				}
				return entryName + "." + ext
			}
			return gid + "." + ext
		}, name, productID, logger); err != nil {
			if logger != nil {
				logger.Warn("gallery file skipped", "file", gid, "err", err)
			}
		}
	}
	return listingImage, nil
}

func uploadNamedFile(wiki *MediaWikiClient, api *vrchat.Client, fileID string, nameFn func(ext string) string, displayName, productID string, logger *slog.Logger) (string, error) {
	data, ext, err := api.DownloadFileBytes(fileID)
	if err != nil {
		return "", err
	}
	if ext == "" {
		ext = "png"
	}
	filename := nameFn(ext)
	desc := ListingImageFileDescription(displayName, productID)
	uploaded, err := wiki.UploadFile(filename, data, desc)
	if err != nil {
		return "", err
	}
	if logger != nil {
		logger.Info("store image processed", "filename", filename, "uploaded", uploaded)
	}
	return filename, nil
}

func syncAvatarImage(wiki *MediaWikiClient, api *vrchat.Client, av map[string]any, logger *slog.Logger) (string, error) {
	imageURL := stringField(av, "imageUrl")
	if imageURL == "" {
		imageURL = stringField(av, "thumbnailImageUrl")
	}
	name := stringField(av, "name")
	if name == "" {
		name = stringField(av, "displayName")
	}
	if imageURL == "" {
		return vrchat.ListingImageFilename(name, "png"), nil
	}
	data, ext, err := api.DownloadImage(imageURL)
	if err != nil {
		if idx := strings.Index(imageURL, "file_"); idx >= 0 {
			fid := imageURL[idx:]
			if len(fid) > 41 {
				fid = fid[:41]
			}
			data, ext, err = api.DownloadFileBytes(fid)
		}
		if err != nil {
			return "", err
		}
	}
	filename := vrchat.ListingImageFilename(name, ext)
	desc := ListingImageFileDescription(name, stringField(av, "id"))
	uploaded, err := wiki.UploadFile(filename, data, desc)
	if err != nil {
		return "", err
	}
	if logger != nil {
		logger.Info("avatar image processed", "filename", filename, "uploaded", uploaded)
	}
	return filename, nil
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}
