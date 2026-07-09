package mediawiki

import (
	"fmt"
	"strings"
)

const maxEditSummaryLen = 120

func clipSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "sync wiki page"
	}
	runes := []rune(summary)
	if len(runes) <= maxEditSummaryLen {
		return summary
	}
	return string(runes[:maxEditSummaryLen-3]) + "..."
}

func BuildEditSummary(title, newText string) string {
	const prefix = "Template:World/"
	if !strings.HasPrefix(title, prefix) {
		return clipSummary(fmt.Sprintf("sync %s", title))
	}

	remainder := strings.TrimPrefix(title, prefix)
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 {
		return clipSummary(fmt.Sprintf("sync %s", title))
	}

	worldID := parts[0]
	if len(parts) == 1 {
		return clipSummary(fmt.Sprintf("world %s marker", worldID))
	}

	field := strings.Join(parts[1:], "/")
	if strings.TrimSpace(newText) == "" {
		return clipSummary(fmt.Sprintf("world %s %s", worldID, field))
	}
	return clipSummary(fmt.Sprintf("world %s %s", worldID, field))
}
