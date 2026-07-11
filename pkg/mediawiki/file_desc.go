package mediawiki

import (
	"fmt"
	"strings"
)

// WorldImageURLFileDescription builds the File: page wikitext for a mirrored
// world preview image (imageUrl).
func WorldImageURLFileDescription(worldID, authorName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "World preview image for [[Community:{{World/%s/name}}]].\n", worldID)
	fmt.Fprintf(&b, "Source: [https://vrchat.com/home/world/%s VRChat world listing]\n", worldID)
	if author := strings.TrimSpace(authorName); author != "" {
		fmt.Fprintf(&b, "Author: %s\n", author)
	}
	b.WriteString("\n{{ license VRC public section8}}")
	return b.String()
}

// YouTubeThumbnailFileDescription builds the File: page wikitext for a mirrored
// YouTube preview thumbnail (previewYoutubeId).
func YouTubeThumbnailFileDescription(worldID, videoID string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "YouTube preview thumbnail for [[Community:{{World/%s/name}}]].\n", worldID)
	fmt.Fprintf(&b, "Video: https://www.youtube.com/watch?v=%s\n", strings.TrimSpace(videoID))
	b.WriteString("Used for informational display on wiki.vrchat.com.\n")
	b.WriteString("\n{{ license 3rd-party-permission}}")
	return b.String()
}
