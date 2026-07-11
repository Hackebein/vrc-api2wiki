package mediawiki

import (
	"fmt"
	"strings"
	"time"
)

type fileInformationParams struct {
	description           string
	source                string
	date                  string
	author                string
	permission            string
	otherVersions         string
	additionalInformation string
}

func fileInformationWikitext(p fileInformationParams) string {
	var b strings.Builder
	b.WriteString("== Summary ==\n")
	b.WriteString("{{File information\n")
	fmt.Fprintf(&b, "|description = %s\n", p.description)
	fmt.Fprintf(&b, "|source      = %s\n", p.source)
	fmt.Fprintf(&b, "|date        = %s\n", p.date)
	fmt.Fprintf(&b, "|author      = %s\n", p.author)
	fmt.Fprintf(&b, "|permission  = %s\n", p.permission)
	fmt.Fprintf(&b, "|other_versions = %s\n", p.otherVersions)
	fmt.Fprintf(&b, "|additional_information = %s\n", p.additionalInformation)
	b.WriteString("}}\n")
	return b.String()
}

func fileDescriptionPage(info fileInformationParams, license string) string {
	var b strings.Builder
	b.WriteString(fileInformationWikitext(info))
	if license != "" {
		b.WriteString("== Licensing ==\n")
		b.WriteString(license)
		if !strings.HasSuffix(license, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func worldLinkDescription(prefix, worldID string) string {
	return fmt.Sprintf("%s for [[%s|{{World/%s/name}}]].", prefix, WorldAliasPageTitle(worldID), worldID)
}

func formatWikiDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format("January 2, 2006")
		}
	}
	return ""
}

func worldDateFromMap(world map[string]any) string {
	for _, key := range []string{"updated_at", "releaseDate", "created_at"} {
		v, ok := world[key]
		if !ok || v == nil {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		if formatted := formatWikiDate(s); formatted != "" {
			return formatted
		}
	}
	return ""
}

// WorldImageURLFileDescription builds the File: page wikitext for a mirrored
// world preview image (imageUrl).
func WorldImageURLFileDescription(worldID, authorName, date string) string {
	return fileDescriptionPage(fileInformationParams{
		description:           worldLinkDescription("World preview image", worldID),
		source:                "VRChat API",
		date:                  strings.TrimSpace(date),
		author:                strings.TrimSpace(authorName),
		additionalInformation: strings.TrimSpace(worldID),
	}, "{{license VRC public section8}}")
}

// YouTubeThumbnailFileDescription builds the File: page wikitext for a mirrored
// YouTube preview thumbnail (previewYoutubeId).
func YouTubeThumbnailFileDescription(worldID, videoID, authorName, date string) string {
	videoID = strings.TrimSpace(videoID)
	return fileDescriptionPage(fileInformationParams{
		description:           worldLinkDescription("YouTube preview thumbnail", worldID),
		source:                "VRChat API",
		date:                  strings.TrimSpace(date),
		author:                strings.TrimSpace(authorName),
		additionalInformation: fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
	}, "")
}
