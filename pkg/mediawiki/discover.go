package mediawiki

import (
	"fmt"
	"regexp"
	"strings"
)

// worldInfoboxTemplates are the wiki templates whose transclusions carry a
// world id parameter. "Infobox/Official World" is handled identically to
// "Infobox/World".
var worldInfoboxTemplates = []string{
	"Template:Infobox/World",
	"Template:Infobox/Official World",
}

// WorldRef is a world id together with the infobox template it was
// discovered through ("Infobox/World" or "Infobox/Official World").
type WorldRef struct {
	ID      string
	Infobox string
}

var (
	infoboxWorldBlockRe = regexp.MustCompile(`(?is)\{\{(Infobox/(?:Official )?World)(.*?)\}\}`)
	idParamRe           = regexp.MustCompile(`(?i)(?:^|\|)\s*id\s*=\s*([^\n|}]+)`)
	linkParamRe         = regexp.MustCompile(`(?i)\|\s*link\s*=[^\n]*`)
	worldIDRe           = regexp.MustCompile(`^wrld_[0-9a-fA-F-]{36}$`)
	worldIDAnywhereRe   = regexp.MustCompile(`wrld_[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
)

func IsValidWorldID(id string) bool {
	return worldIDRe.MatchString(strings.TrimSpace(id))
}

// parseBlockWorldIDs extracts world ids from one infobox parameter block.
// An explicit id= parameter takes precedence; when absent or empty, the id is
// taken from the link= parameter (matched anywhere in its value, covering
// {{World link|wrld_...}} and {{VRC link|https://.../world/wrld_...}} forms).
func parseBlockWorldIDs(block string) []string {
	var ids []string
	for _, idMatch := range idParamRe.FindAllStringSubmatch(block, -1) {
		id := strings.TrimSpace(idMatch[1])
		if IsValidWorldID(id) {
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		return ids
	}
	for _, linkMatch := range linkParamRe.FindAllString(block, -1) {
		if id := worldIDAnywhereRe.FindString(linkMatch); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

func ParseInfoboxWorlds(wikitext string) []WorldRef {
	var refs []WorldRef
	seen := make(map[string]struct{})
	for _, match := range infoboxWorldBlockRe.FindAllStringSubmatch(wikitext, -1) {
		if len(match) < 3 {
			continue
		}
		infobox := match[1]
		for _, id := range parseBlockWorldIDs(match[2]) {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			refs = append(refs, WorldRef{ID: id, Infobox: infobox})
		}
	}
	return refs
}

// ParseInfoboxWorldIDs returns deduplicated world ids from infobox blocks in
// wikitext (first occurrence order within the page).
func ParseInfoboxWorldIDs(wikitext string) []string {
	refs := ParseInfoboxWorlds(wikitext)
	ids := make([]string, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
	}
	return ids
}

func (c *MediaWikiClient) listTemplateTransclusions(templateTitle string) ([]string, error) {
	var titles []string
	eicontinue := ""

	for {
		// No namespace filter: transclusions are collected from all
		// namespaces (main, Community, etc.).
		params := map[string]string{
			"action":  "query",
			"list":    "embeddedin",
			"eititle": templateTitle,
			"eilimit": "500",
		}
		if eicontinue != "" {
			params["eicontinue"] = eicontinue
		}

		result, err := c.apiRequest(params)
		if err != nil {
			return nil, fmt.Errorf("list embeddedin: %w", err)
		}

		query, ok := result["query"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid response structure: missing query")
		}
		embedded, ok := query["embeddedin"].([]any)
		if !ok {
			return nil, fmt.Errorf("invalid response structure: missing embeddedin")
		}
		for _, item := range embedded {
			m, _ := item.(map[string]any)
			if m == nil {
				continue
			}
			title, _ := m["title"].(string)
			if title != "" {
				titles = append(titles, title)
			}
		}

		if cont, ok := result["continue"].(map[string]any); ok {
			if next, _ := cont["eicontinue"].(string); next != "" {
				eicontinue = next
				continue
			}
		}
		break
	}
	return titles, nil
}

func (c *MediaWikiClient) ListInfoboxWorldPages() ([]string, error) {
	var pages []string
	seenPages := make(map[string]struct{})
	for _, template := range worldInfoboxTemplates {
		titles, err := c.listTemplateTransclusions(template)
		if err != nil {
			return nil, fmt.Errorf("transclusions of %s: %w", template, err)
		}
		if c.logger != nil {
			c.logger.Info("template transclusions found", "template", template, "count", len(titles))
		}
		for _, title := range titles {
			if _, ok := seenPages[title]; ok {
				continue
			}
			seenPages[title] = struct{}{}
			pages = append(pages, title)
		}
	}
	return pages, nil
}

// DiscoverWorldRefs scans infobox transclusions and returns deduplicated world
// ids in discovery order together with the infobox template name(s) each id
// was found under (Infobox/World and/or Infobox/Official World).
func (c *MediaWikiClient) DiscoverWorldRefs() ([]string, map[string][]string, error) {
	pages, err := c.ListInfoboxWorldPages()
	if err != nil {
		return nil, nil, err
	}

	seen := make(map[string]struct{})
	var ids []string
	infoboxes := make(map[string][]string)

	for _, pageTitle := range pages {
		content, err := c.getPageContent(pageTitle)
		if err != nil {
			if c.logger != nil {
				c.logger.Warn("skip page: could not read content", "page", pageTitle, "error", err)
			}
			continue
		}
		for _, ref := range ParseInfoboxWorlds(content) {
			if _, ok := seen[ref.ID]; !ok {
				seen[ref.ID] = struct{}{}
				ids = append(ids, ref.ID)
			}
			if !containsInfobox(infoboxes[ref.ID], ref.Infobox) {
				infoboxes[ref.ID] = append(infoboxes[ref.ID], ref.Infobox)
			}
		}
	}
	return ids, infoboxes, nil
}

func containsInfobox(list []string, infobox string) bool {
	for _, item := range list {
		if item == infobox {
			return true
		}
	}
	return false
}

func (c *MediaWikiClient) DiscoverWorldIDs() ([]string, error) {
	ids, _, err := c.DiscoverWorldRefs()
	return ids, err
}

// EnsureWorldMarkerPage keeps Template:World/<id> in sync with the expected
// id-only infobox call(s); EditPage only writes when the content differs.
func (c *MediaWikiClient) EnsureWorldMarkerPage(worldID string, infoboxes []string) error {
	title := WorldPageTitle(worldID, "")
	return c.EditPage(title, WorldMarkerWikitext(worldID, infoboxes), true)
}
