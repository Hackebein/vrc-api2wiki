package vrchat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var excludedKeys = map[string]struct{}{
	"instances":             {},
	"slimInstances":         {},
	"unityPackageUrl":       {},
	"unityPackageUrlObject": {},
	"assetUrlObject":        {},
	"pluginUrlObject":       {},
	"thumbnailImageUrl":     {},
	"occupants":             {},
	"privateOccupants":      {},
	"publicOccupants":       {},
	"tags":                  {},
}

func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val) == "" || val == "none"
	case bool:
		return false
	case float64:
		return false
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	default:
		return false
	}
}

// IsExcludedWorldKey reports whether a top-level world JSON key is omitted from sync.
func IsExcludedWorldKey(key string) bool {
	_, ok := excludedKeys[key]
	return ok
}

func scalarToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case json.Number:
		return val.String()
	default:
		return fmt.Sprint(v)
	}
}

func joinScalarArray(items []any) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if isEmptyValue(item) {
			continue
		}
		switch item.(type) {
		case map[string]any, []any:
			continue
		default:
			parts = append(parts, scalarToString(item))
		}
	}
	return strings.Join(parts, ", ")
}

func arrayObjectKey(obj map[string]any, index int) string {
	if platform, ok := obj["platform"].(string); ok && strings.TrimSpace(platform) != "" {
		return strings.TrimSpace(platform)
	}
	return strconv.Itoa(index)
}

func flattenValue(path []string, value any, pages map[string]string) {
	if len(path) == 0 {
		return
	}
	key := path[len(path)-1]
	if _, skip := excludedKeys[key]; skip {
		return
	}
	if isEmptyValue(value) {
		return
	}

	switch val := value.(type) {
	case map[string]any:
		for k, child := range val {
			if _, skip := excludedKeys[k]; skip {
				continue
			}
			flattenValue(append(path, k), child, pages)
		}
	case []any:
		if len(val) == 0 {
			return
		}
		allScalars := true
		for _, item := range val {
			switch item.(type) {
			case map[string]any, []any:
				allScalars = false
			}
		}
		if allScalars {
			joined := joinScalarArray(val)
			if joined != "" {
				pages[strings.Join(path, "/")] = joined
			}
			return
		}
		for i, item := range val {
			switch child := item.(type) {
			case map[string]any:
				segment := arrayObjectKey(child, i)
				for k, grandchild := range child {
					if _, skip := excludedKeys[k]; skip {
						continue
					}
					flattenValue(append(path, segment, k), grandchild, pages)
				}
			case []any:
				flattenValue(append(path, strconv.Itoa(i)), child, pages)
			default:
				if !isEmptyValue(child) {
					flattenValue(append(path, strconv.Itoa(i)), child, pages)
				}
			}
		}
	default:
		pages[strings.Join(path, "/")] = scalarToString(val)
	}
}

func FlattenWorld(world map[string]any) map[string]string {
	pages := make(map[string]string)
	keys := make([]string, 0, len(world))
	for k := range world {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, skip := excludedKeys[k]; skip {
			continue
		}
		flattenValue([]string{k}, world[k], pages)
	}
	addDerivedPages(world, pages)
	return pages
}

var platformLabels = map[string]string{
	"standalonewindows": "PC",
	"android":           "Android",
	"ios":               "iOS",
}

const authorTagPrefix = "author_tag_"

var listingTagLabels = map[string]string{
	"author_tag_avatar": "Avatar Worlds",
	"author_tag_game":   "Games",
	"content_featured":  "Featured",
}

var contentTagLabels = map[string]string{
	"content_adult":    "Adult",
	"content_combat":   "Combat",
	"content_gore":     "Gore",
	"content_horror":   "Horror",
	"content_other":    "Other",
	"content_sex":      "Sex",
	"content_violence": "Violence",
}

var excludedAuthorListingTags = map[string]struct{}{
	"author_tag_avatar": {},
	"author_tag_game":   {},
}

func deriveDisplayTags(tags []any) string {
	parts := make([]string, 0, len(tags))
	for _, item := range tags {
		tag := strings.TrimSpace(scalarToString(item))
		if tag == "" || !strings.HasPrefix(tag, authorTagPrefix) {
			continue
		}
		if _, skip := excludedAuthorListingTags[tag]; skip {
			continue
		}
		parts = append(parts, strings.TrimPrefix(tag, authorTagPrefix))
	}
	return strings.Join(parts, ", ")
}

func deriveListingTags(tags []any) string {
	parts := make([]string, 0, len(tags))
	for _, item := range tags {
		tag := strings.TrimSpace(scalarToString(item))
		if tag == "" {
			continue
		}
		if label, ok := listingTagLabels[tag]; ok {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, ", ")
}

func deriveContentTags(tags []any) string {
	parts := make([]string, 0, len(tags))
	for _, item := range tags {
		tag := strings.TrimSpace(scalarToString(item))
		if tag == "" {
			continue
		}
		if label, ok := contentTagLabels[tag]; ok {
			parts = append(parts, label)
		}
	}
	return strings.Join(parts, ", ")
}

func worldTags(world map[string]any) []any {
	raw, ok := world["tags"]
	if !ok || isEmptyValue(raw) {
		return nil
	}
	tags, ok := raw.([]any)
	if !ok {
		return nil
	}
	return tags
}

func addDerivedPages(world map[string]any, pages map[string]string) {
	delete(pages, "tags")

	if platforms := derivePlatforms(world); platforms != "" {
		pages["platforms"] = platforms
	}

	if tags := worldTags(world); len(tags) > 0 {
		if display := deriveDisplayTags(tags); display != "" {
			pages["tags"] = display
		}
		if listing := deriveListingTags(tags); listing != "" {
			pages["listing"] = listing
		}
		if content := deriveContentTags(tags); content != "" {
			pages["content"] = content
		}
	}
}

func derivePlatforms(world map[string]any) string {
	raw, ok := world["unityPackages"]
	if !ok || isEmptyValue(raw) {
		return ""
	}

	seen := make(map[string]struct{})
	var labels []string

	addLabel := func(platform string) {
		platform = strings.TrimSpace(platform)
		if platform == "" {
			return
		}
		label := platformLabels[strings.ToLower(platform)]
		if label == "" {
			label = platform
		}
		if _, ok := seen[label]; ok {
			return
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
	}

	switch pkgs := raw.(type) {
	case []any:
		for _, item := range pkgs {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if platform, ok := obj["platform"].(string); ok {
				addLabel(platform)
			}
		}
	case map[string]any:
		platforms := make([]string, 0, len(pkgs))
		for platform := range pkgs {
			platforms = append(platforms, platform)
		}
		sort.Strings(platforms)
		for _, platform := range platforms {
			addLabel(platform)
		}
	}

	return strings.Join(labels, ", ")
}
