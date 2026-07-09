package vrchat

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var excludedKeys = map[string]struct{}{
	"instances":            {},
	"slimInstances":        {},
	"unityPackageUrl":      {},
	"unityPackageUrlObject": {},
	"assetUrlObject":       {},
	"pluginUrlObject":      {},
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

func addDerivedPages(world map[string]any, pages map[string]string) {
	if platforms := derivePlatforms(world); platforms != "" {
		pages["platforms"] = platforms
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
