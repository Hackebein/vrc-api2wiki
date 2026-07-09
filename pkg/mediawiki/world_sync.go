package mediawiki

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func WorldPageTitle(worldID, subpath string) string {
	if subpath == "" {
		return fmt.Sprintf("Template:World/%s", worldID)
	}
	return fmt.Sprintf("Template:World/%s/%s", worldID, subpath)
}

// imageProperties are world properties whose value is an image URL; the image
// is mirrored to the wiki and the subpage stores the wiki file reference.
var imageProperties = map[string]struct{}{
	"imageUrl":          {},
	"thumbnailImageUrl": {},
}

// WorldImageFilename returns the wiki file name (without "File:" prefix) for
// an image property of a world, e.g. "wrld_..._imageUrl.png".
func WorldImageFilename(worldID, property, ext string) string {
	return fmt.Sprintf("%s_%s.%s", worldID, property, ext)
}

func (c *MediaWikiClient) syncWorldImage(api *vrchat.Client, worldID, property, imageURL string) (string, error) {
	data, ext, err := api.DownloadImage(imageURL)
	if err != nil {
		return "", fmt.Errorf("download %s image: %w", property, err)
	}
	filename := WorldImageFilename(worldID, property, ext)
	uploaded, err := c.UploadFile(filename, data)
	if err != nil {
		return "", fmt.Errorf("upload %s: %w", filename, err)
	}
	if c.logger != nil {
		c.logger.Info("world image processed", "world_id", worldID, "property", property, "filename", filename, "uploaded", uploaded)
	}
	return "File:" + filename, nil
}

// syncYouTubeThumbnail mirrors the thumbnail of a world's YouTube preview
// video to the wiki. The previewYoutubeId subpage keeps the raw video id; the
// infobox template derives the file name from the world id.
func (c *MediaWikiClient) syncYouTubeThumbnail(api *vrchat.Client, worldID, videoID string) error {
	data, ext, err := api.DownloadYouTubeThumbnail(videoID)
	if err != nil {
		return err
	}
	filename := WorldImageFilename(worldID, "previewYoutubeId", ext)
	uploaded, err := c.UploadFile(filename, data)
	if err != nil {
		return fmt.Errorf("upload %s: %w", filename, err)
	}
	if c.logger != nil {
		c.logger.Info("youtube thumbnail processed", "world_id", worldID, "video_id", videoID, "filename", filename, "uploaded", uploaded)
	}
	return nil
}

func (c *MediaWikiClient) SyncWorldData(api *vrchat.Client, worldID string, world map[string]any) error {
	pages := vrchat.FlattenWorld(world)

	for subpath, value := range pages {
		if _, isImage := imageProperties[subpath]; isImage {
			fileRef, err := c.syncWorldImage(api, worldID, subpath, value)
			if err != nil {
				return fmt.Errorf("sync image %s for %s: %w", subpath, worldID, err)
			}
			pages[subpath] = fileRef
		}
	}

	if videoID, ok := pages["previewYoutubeId"]; ok {
		if err := c.syncYouTubeThumbnail(api, worldID, videoID); err != nil {
			return fmt.Errorf("sync youtube thumbnail for %s: %w", worldID, err)
		}
	}

	for subpath, value := range pages {
		title := WorldPageTitle(worldID, subpath)
		text := SanitizeForWiki(value)
		if err := c.EditPage(title, text, true); err != nil {
			return fmt.Errorf("edit %s: %w", title, err)
		}
	}

	if c.logger != nil {
		c.logger.Info("world synced", "world_id", worldID, "pages", len(pages))
	}
	return nil
}

func WorldIDsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("VRC_API2WIKI_WORLD_IDS"))
	if raw == "" {
		return nil
	}
	var ids []string
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if IsValidWorldID(id) {
			ids = append(ids, id)
		}
	}
	return ids
}

func ProcessingLimitFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("VRC_API2WIKI_LIMIT"))
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// WorldMarkerWikitext builds the content of the Template:World/<id> marker
// page: the id-only infobox call(s) for the template(s) the world was
// discovered through, so visiting the marker page previews the infobox
// rendered entirely from the data subpages. Worlds without discovery info
// (e.g. from VRC_API2WIKI_WORLD_IDS) get the generic Infobox/World call.
func WorldMarkerWikitext(worldID string, infoboxes []string) string {
	if len(infoboxes) == 0 {
		infoboxes = []string{"Infobox/World"}
	}
	order := []string{"Infobox/World", "Infobox/Official World"}
	var b strings.Builder
	for _, name := range order {
		if !containsInfobox(infoboxes, name) {
			continue
		}
		b.WriteString("{{")
		b.WriteString(name)
		b.WriteString("\n|id=")
		b.WriteString(worldID)
		b.WriteString("\n}}\n")
	}
	return b.String()
}

// WorldAliasPageTitle returns the Community-namespace alias page title for a
// world id, e.g. "Community:wrld_...". Visiting it redirects to (or lists) the
// article(s) that use the world's infobox.
func WorldAliasPageTitle(worldID string) string {
	return fmt.Sprintf("Community:%s", worldID)
}

// langLink wraps an article title in Special:MyLanguage so the reader is sent
// to the translation matching their interface language, falling back to the
// source page.
func langLink(target string) string {
	return "Special:MyLanguage/" + target
}

// WorldAliasWikitext builds the content of a world-id alias page. A single
// target becomes a redirect; multiple targets render a {{Disambiguation}} page
// with a bullet list whose display title is taken from the world's name
// subpage (Template:World/<id>/name). Links go through Special:MyLanguage so
// translated variants auto-select the reader's language. Targets are sorted so
// repeated runs produce identical content (keeping EditPage's change detection
// idempotent).
func WorldAliasWikitext(worldID string, targets []string) string {
	sorted := append([]string(nil), targets...)
	sort.Strings(sorted)
	if len(sorted) == 1 {
		return fmt.Sprintf("#REDIRECT [[%s]]\n", langLink(sorted[0]))
	}
	nameSubpage := WorldPageTitle(worldID, "name")
	var b strings.Builder
	fmt.Fprintf(&b, "{{#ifexist:%s|{{DISPLAYTITLE:{{World/%s/name}}}}}}\n", nameSubpage, worldID)
	b.WriteString("{{Disambiguation}}\n")
	for _, target := range sorted {
		b.WriteString("* [[")
		b.WriteString(langLink(target))
		b.WriteString("]]\n")
	}
	return b.String()
}

// EnsureWorldAliasPage keeps Community:<id> in sync with the article(s) that
// use the world's infobox; EditPage only writes when the content differs. It
// is a no-op when no article targets are known.
func (c *MediaWikiClient) EnsureWorldAliasPage(worldID string, targets []string) error {
	if len(targets) == 0 {
		return nil
	}
	return c.EditPage(WorldAliasPageTitle(worldID), WorldAliasWikitext(worldID, targets), true)
}

func RunSync(c *MediaWikiClient, api *vrchat.Client, logger *slog.Logger) error {
	worldInfoboxes := make(map[string][]string)
	worldArticlePages := make(map[string][]string)

	worldIDs := WorldIDsFromEnv()
	if len(worldIDs) == 0 {
		d, err := c.DiscoverWorldRefs()
		if err != nil {
			return fmt.Errorf("discover world ids: %w", err)
		}
		worldIDs = d.IDs
		worldInfoboxes = d.Infoboxes
		worldArticlePages = d.ArticlePages
	} else if logger != nil {
		logger.Info("using world ids from VRC_API2WIKI_WORLD_IDS", "count", len(worldIDs))
	}

	discoveredCount := len(worldIDs)
	if logger != nil {
		logger.Info("discovered world ids", "count", discoveredCount)
	}

	limit := ProcessingLimitFromEnv()
	if limit > 0 && limit < len(worldIDs) {
		if logger != nil {
			logger.Info("processing limit applied", "limit", limit, "discovered", discoveredCount)
		}
		worldIDs = worldIDs[:limit]
	}

	for _, worldID := range worldIDs {
		if err := c.EnsureWorldMarkerPage(worldID, worldInfoboxes[worldID]); err != nil {
			return fmt.Errorf("ensure marker for %s: %w", worldID, err)
		}
	}

	for _, worldID := range worldIDs {
		if err := c.EnsureWorldAliasPage(worldID, worldArticlePages[worldID]); err != nil {
			return fmt.Errorf("ensure alias for %s: %w", worldID, err)
		}
	}

	for _, worldID := range worldIDs {
		world, err := api.GetWorld(worldID)
		if err != nil {
			if logger != nil {
				logger.Error("fetch world failed", "world_id", worldID, "error", err)
			}
			continue
		}
		if err := c.SyncWorldData(api, worldID, world); err != nil {
			return fmt.Errorf("sync world %s: %w", worldID, err)
		}
	}
	return nil
}
