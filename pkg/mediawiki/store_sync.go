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

func ListingImageFileDescription(listingName, productID, sourceRef string) string {
	additional := strings.TrimSpace(productID)
	if ref := strings.TrimSpace(sourceRef); ref != "" {
		if additional != "" {
			additional = additional + " " + ref
		} else {
			additional = ref
		}
	}
	return fileDescriptionPage(fileInformationParams{
		description:           fmt.Sprintf("Store listing image for %s.", listingName),
		source:                "VRChat API",
		author:                "VRChat",
		additionalInformation: additional,
	}, "{{license VRC public section8}}")
}

func ShelfIconFileDescription(shelfTitle, sourceRef string) string {
	return fileDescriptionPage(fileInformationParams{
		description:           fmt.Sprintf("Store shelf icon for %s.", vrchat.DisplayShelfTitle(shelfTitle)),
		source:                "VRChat API",
		author:                "VRChat",
		additionalInformation: strings.TrimSpace(sourceRef),
	}, "{{license VRC public section8}}")
}

type imageSyncCache struct {
	byFileID map[string]string
	disk     *diskImageCache
}

func newImageSyncCache() *imageSyncCache {
	return &imageSyncCache{
		byFileID: make(map[string]string),
		disk:     openDiskImageCache(imageCacheDir),
	}
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
	defer func() {
		if err := api.PersistSession(); err != nil && logger != nil {
			logger.Warn("persist vrchat session failed", "err", err)
		}
	}()

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

	apiSnap := openAPICache(apiCacheDir)
	prevStore, _ := apiSnap.LoadStore()
	shelfIcons := apiSnap.LoadShelfIcons()

	now := time.Now()
	year := now.UTC().Year()
	onShelf := map[string]bool{}
	var offers []currentOfferShelf

	wikiShelves := loadWikiShelfIndex(wiki, year, logger)
	images := newImageSyncCache()

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

		shelfUnchanged := shelfListingsUnchanged(apiSnap, prevStore, shelf, snap.Listings)
		if shelfUnchanged {
			iconFile := shelfIcons[wikiTitle]
			offers = append(offers, currentOfferShelf{Title: wikiTitle, IconFile: iconFile})
			for _, id := range listingIDs {
				onShelf[id] = true
			}
			if logger != nil {
				logger.Info("store shelf unchanged; skipped", "shelf", pathTitle)
			}
			continue
		}

		existingPage, _ := wiki.GetPageContent(StoreListingsPageTitle(year, wikiTitle))
		existingCards := parseInventoryWikiIndex(existingPage)
		existingIcon := parseShelfIconFromHeader(existingPage)
		if existingIcon == "" {
			existingIcon = shelfIcons[wikiTitle]
		}

		iconFile, err := syncShelfIcon(wiki, api, shelf, wikiTitle, existingIcon, images, logger)
		if err != nil {
			if logger != nil {
				logger.Warn("shelf icon skipped", "shelf", pathTitle, "err", err)
			}
			iconFile = existingIcon
		}
		if iconFile != "" {
			shelfIcons[wikiTitle] = iconFile
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
			imageName := preferredImage
			prevListing, hasPrev := apiSnap.LoadListing(id)
			if !hasPrev || !apiMapsEqual(prevListing, hydrated) {
				imageName, err = syncListingMedia(wiki, api, hydrated, preferredImage, images, logger)
				if err != nil {
					return fmt.Errorf("listing media %s: %w", id, err)
				}
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
		if err := wiki.WritePage(StoreListingsPageTitle(year, wikiTitle), page, true); err != nil {
			return fmt.Errorf("write shelf %s: %w", pathTitle, err)
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
		prevListing, hasPrev := apiSnap.LoadListing(id)
		if hasPrev && apiMapsEqual(prevListing, listing) {
			orphanCount++
			continue
		}
		if _, err := syncListingMedia(wiki, api, listing, "", images, logger); err != nil {
			return fmt.Errorf("listing media %s: %w", id, err)
		}
		orphanCount++
	}

	avatarImages, avatarListings, avatarStyles, avatarAuthors, err := syncAvatarMarketplacePages(wiki, api, snap.Avatars, year, now, images, logger)
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
		if err := wiki.WritePage(StoreCurrentOffersPageTitle(), page, true); err != nil {
			return fmt.Errorf("write current offers: %w", err)
		}
	}

	if err := persistStoreAPICache(apiSnap, snap, shelfIcons); err != nil {
		return err
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

func shelfListingsUnchanged(apiSnap *apiCache, prevStore map[string]any, shelf map[string]any, listings map[string]map[string]any) bool {
	if prevStore == nil || apiSnap == nil {
		return false
	}
	apiTitle := stringField(shelf, "shelfTitle")
	prevShelf := findShelfByTitle(prevStore, apiTitle)
	if prevShelf == nil {
		return false
	}
	if shelfIconImageID(shelf) != shelfIconImageID(prevShelf) {
		return false
	}
	for _, listing := range vrchat.ShelfListings(shelf) {
		id := stringField(listing, "id")
		if id == "" {
			return false
		}
		curr := listings[id]
		if curr == nil {
			curr = listing
		}
		prev, ok := apiSnap.LoadListing(id)
		if !ok || !apiMapsEqual(prev, curr) {
			return false
		}
	}
	return true
}

func persistStoreAPICache(apiSnap *apiCache, snap *vrchat.StoreSnapshot, shelfIcons map[string]string) error {
	if err := apiSnap.SaveStore(snap.Store); err != nil {
		return fmt.Errorf("save store api cache: %w", err)
	}
	ids := make([]string, 0, len(snap.Listings))
	for id := range snap.Listings {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := apiSnap.SaveListing(id, snap.Listings[id]); err != nil {
			return fmt.Errorf("save listing api cache %s: %w", id, err)
		}
	}
	if err := apiSnap.SaveShelfIcons(shelfIcons); err != nil {
		return fmt.Errorf("save shelf icons api cache: %w", err)
	}
	return nil
}

func syncAvatarMarketplacePages(wiki *MediaWikiClient, api *vrchat.Client, avatars []map[string]any, year int, now time.Time, cache *imageSyncCache, logger *slog.Logger) (imageCount, listings, styles, authors int, err error) {
	built := vrchat.BuildAvatarMarketplaceListings(avatars, now)
	byStyle := map[string][]vrchat.InventoryContentDisplay{}
	byAuthor := map[string][]vrchat.InventoryContentDisplay{}

	for _, listing := range built {
		filename, err := syncAvatarListingImage(wiki, api, listing, cache, logger)
		if err != nil {
			if logger != nil {
				logger.Warn("avatar listing image skipped", "listing", listing.ListingID, "err", err)
			}
			continue
		}
		imageCount++
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
			return imageCount, listings, 0, 0, fmt.Errorf("edit avatar style %s: %w", style, err)
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
			return imageCount, listings, len(styleNames), 0, fmt.Errorf("edit avatar author %s: %w", author, err)
		}
	}

	if len(styleNames) > 0 || len(authorNames) > 0 {
		index := vrchat.RenderAvatarStylesIndexWikitext(year, styleNames)
		if err := wiki.EditPage(StoreAvatarStylesIndexPageTitle(year), index, true); err != nil {
			return imageCount, listings, len(styleNames), len(authorNames), fmt.Errorf("edit avatar styles index: %w", err)
		}
	}
	if len(authorNames) > 0 {
		authorsIndex := vrchat.RenderAvatarAuthorsIndexWikitext(year, authorNames)
		if err := wiki.EditPage(StoreAvatarAuthorsIndexPageTitle(year), authorsIndex, true); err != nil {
			return imageCount, listings, len(styleNames), len(authorNames), fmt.Errorf("edit avatar authors index: %w", err)
		}
	}
	return imageCount, listings, len(styleNames), len(authorNames), nil
}

func syncAvatarListingImage(wiki *MediaWikiClient, api *vrchat.Client, listing vrchat.AvatarMarketplaceListing, cache *imageSyncCache, logger *slog.Logger) (string, error) {
	name := listing.Display.Name
	if listing.ImageID != "" {
		return uploadNamedFile(wiki, api, listing.ImageID, func(ext string) string {
			return vrchat.ListingImageFilename(name, ext)
		}, func(sourceRef string) string {
			return ListingImageFileDescription(name, listing.ListingID, sourceRef)
		}, cache, logger)
	}
	return syncAvatarImage(wiki, api, listing.FallbackAvatar, cache, logger)
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

func syncShelfIcon(wiki *MediaWikiClient, api *vrchat.Client, shelf map[string]any, wikiTitle, existingIcon string, cache *imageSyncCache, logger *slog.Logger) (string, error) {
	fileID := stringField(shelf, "shelfIconImageId")
	if fileID == "" {
		return existingIcon, nil
	}
	filename, err := uploadNamedFile(wiki, api, fileID, func(ext string) string {
		if existingIcon != "" {
			return existingIcon
		}
		return vrchat.ShelfIconFilename(wikiTitle, ext)
	}, func(sourceRef string) string {
		return ShelfIconFileDescription(wikiTitle, sourceRef)
	}, cache, logger)
	if err != nil {
		return existingIcon, err
	}
	return filename, nil
}

func syncListingMedia(wiki *MediaWikiClient, api *vrchat.Client, listing map[string]any, preferredListingImage string, cache *imageSyncCache, logger *slog.Logger) (string, error) {
	name := strings.TrimSpace(stringField(listing, "displayName"))
	productID := stringField(listing, "id")
	var listingImage string

	if fileID := stringField(listing, "imageId"); fileID != "" {
		filename, err := uploadNamedFile(wiki, api, fileID, func(ext string) string {
			if preferredListingImage != "" {
				return preferredListingImage
			}
			return vrchat.ListingImageFilename(name, ext)
		}, func(sourceRef string) string {
			return ListingImageFileDescription(name, productID, sourceRef)
		}, cache, logger)
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
		prodID := stringField(prod, "id")
		if _, err := uploadNamedFile(wiki, api, pid, func(ext string) string {
			return vrchat.ProductImageFilename(label, pname, ext)
		}, func(sourceRef string) string {
			return ListingImageFileDescription(pname, prodID, sourceRef)
		}, cache, logger); err != nil {
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
		}, func(sourceRef string) string {
			return ListingImageFileDescription(name, productID, sourceRef)
		}, cache, logger); err != nil {
			if logger != nil {
				logger.Warn("gallery file skipped", "file", gid, "err", err)
			}
		}
	}
	return listingImage, nil
}

func uploadNamedFile(wiki *MediaWikiClient, api *vrchat.Client, fileID string, nameFn func(ext string) string, descFn func(sourceRef string) string, cache *imageSyncCache, logger *slog.Logger) (string, error) {
	if cache != nil {
		if name, ok := cache.byFileID[fileID]; ok {
			return name, nil
		}
	}

	info, err := api.GetFileDownload(fileID)
	if err != nil {
		return "", err
	}
	ext := info.Ext
	if ext == "" {
		ext = "png"
	}
	filename := nameFn(ext)
	sourceRef := FileSourceRef(fileID, info.Version)
	desc := descFn(sourceRef)

	uploaded, skippedDownload, err := syncFileBytes(wiki, filename, sourceRef, desc, cache, func() ([]byte, error) {
		data, _, err := api.DownloadImage(info.URL)
		if err != nil {
			data, _, err = api.DownloadFileBytes(fileID)
		}
		return data, err
	}, logger)
	if err != nil {
		return "", err
	}
	if cache != nil {
		cache.byFileID[fileID] = filename
	}
	if logger != nil {
		logger.Info("store image processed", "filename", filename, "uploaded", uploaded, "skipped_download", skippedDownload)
	}
	return filename, nil
}

// syncFileBytes uploads filename from disk cache or downloadFn. Replacements
// are detected because sourceRef includes fileID@version (or yt:<id>); a new
// ref misses the disk cache and fails the wiki sourceRef check.
func syncFileBytes(wiki *MediaWikiClient, filename, sourceRef, desc string, cache *imageSyncCache, downloadFn func() ([]byte, error), logger *slog.Logger) (uploaded, skippedDownload bool, err error) {
	var disk *diskImageCache
	if cache != nil {
		disk = cache.disk
	}

	st, err := wiki.getFileState(filename)
	if err != nil {
		return false, false, err
	}

	if data, sha, ok := disk.Get(sourceRef); ok {
		if st.exists && st.sha1 != "" && strings.EqualFold(st.sha1, sha) {
			if sourceRef != "" && !strings.Contains(st.content, sourceRef) {
				if err := wiki.ensureFileDescription(filename, desc); err != nil {
					return false, false, err
				}
			}
			return false, true, nil
		}
		uploaded, err = wiki.UploadFile(filename, data, desc)
		return uploaded, true, err
	}

	if st.exists && st.sha1 != "" {
		if sourceRef == "" || strings.Contains(st.content, sourceRef) {
			return false, true, nil
		}
	}

	data, err := downloadFn()
	if err != nil {
		return false, false, err
	}
	if err := disk.Put(sourceRef, data); err != nil && logger != nil {
		logger.Warn("image cache write failed", "ref", sourceRef, "err", err)
	}
	uploaded, err = wiki.UploadFile(filename, data, desc)
	return uploaded, false, err
}

func syncAvatarImage(wiki *MediaWikiClient, api *vrchat.Client, av map[string]any, cache *imageSyncCache, logger *slog.Logger) (string, error) {
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
	if fid := vrchat.FileIDFromURL(imageURL); fid != "" {
		return uploadNamedFile(wiki, api, fid, func(ext string) string {
			return vrchat.ListingImageFilename(name, ext)
		}, func(sourceRef string) string {
			return ListingImageFileDescription(name, stringField(av, "id"), sourceRef)
		}, cache, logger)
	}
	data, ext, err := api.DownloadImage(imageURL)
	if err != nil {
		return "", err
	}
	filename := vrchat.ListingImageFilename(name, ext)
	desc := ListingImageFileDescription(name, stringField(av, "id"), "")
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
