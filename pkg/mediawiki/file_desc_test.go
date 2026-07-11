package mediawiki

import (
	"strings"
	"testing"
)

func TestWorldImageURLFileDescription(t *testing.T) {
	got := WorldImageURLFileDescription("wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300", "RootGentle")
	for _, want := range []string{
		"World preview image for [[Community:{{World/wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300/name}}]].",
		"Source: [https://vrchat.com/home/world/wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300 VRChat world listing]",
		"Author: RootGentle",
		"{{ license VRC public section8}}",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in description:\n%s", want, got)
		}
	}
}

func TestWorldImageURLFileDescriptionNoAuthor(t *testing.T) {
	got := WorldImageURLFileDescription("wrld_00000000-0000-4000-8000-000000000001", "")
	if strings.Contains(got, "Author:") {
		t.Fatalf("expected no author line, got:\n%s", got)
	}
	if !strings.Contains(got, "{{ license VRC public section8}}") {
		t.Fatalf("expected license template, got:\n%s", got)
	}
}

func TestYouTubeThumbnailFileDescription(t *testing.T) {
	got := YouTubeThumbnailFileDescription("wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300", "ixGR9XFNfZ4")
	for _, want := range []string{
		"YouTube preview thumbnail for [[Community:{{World/wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300/name}}]].",
		"Video: https://www.youtube.com/watch?v=ixGR9XFNfZ4",
		"Used for informational display on wiki.vrchat.com.",
		"{{ license 3rd-party-permission}}",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in description:\n%s", want, got)
		}
	}
}
