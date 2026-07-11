package mediawiki

import (
	"strings"
	"testing"
)

func TestWorldImageURLFileDescription(t *testing.T) {
	got := WorldImageURLFileDescription(
		"wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300",
		"RootGentle",
		"July 3, 2026",
	)
	for _, want := range []string{
		"== Summary ==",
		"{{File information",
		"|description = World preview image for [[Community:wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300|{{World/wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300/name}}]].",
		"|source      = VRChat API",
		"|date        = July 3, 2026",
		"|author      = RootGentle",
		"|permission  = ",
		"|other_versions = ",
		"|additional_information = wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300",
		"== Licensing ==",
		"{{license VRC public section8}}",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in description:\n%s", want, got)
		}
	}
}

func TestWorldImageURLFileDescriptionNoAuthor(t *testing.T) {
	got := WorldImageURLFileDescription("wrld_00000000-0000-4000-8000-000000000001", "", "")
	if strings.Contains(got, "RootGentle") {
		t.Fatalf("expected no author, got:\n%s", got)
	}
	if !strings.Contains(got, "|author      = \n") {
		t.Fatalf("expected empty author field, got:\n%s", got)
	}
	if !strings.Contains(got, "{{license VRC public section8}}") {
		t.Fatalf("expected license template, got:\n%s", got)
	}
}

func TestYouTubeThumbnailFileDescription(t *testing.T) {
	got := YouTubeThumbnailFileDescription(
		"wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300",
		"ixGR9XFNfZ4",
		"RootGentle",
		"July 3, 2026",
	)
	for _, want := range []string{
		"== Summary ==",
		"{{File information",
		"|description = YouTube preview thumbnail for [[Community:wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300|{{World/wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300/name}}]].",
		"|source      = VRChat API",
		"|date        = July 3, 2026",
		"|author      = RootGentle",
		"|additional_information = https://www.youtube.com/watch?v=ixGR9XFNfZ4",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in description:\n%s", want, got)
		}
	}
}

func TestFormatWikiDate(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"2026-07-03T12:34:56.789Z", "July 3, 2026"},
		{"2026-07-03T12:34:56Z", "July 3, 2026"},
		{"2026-07-03", "July 3, 2026"},
		{"", ""},
		{"invalid", ""},
	}
	for _, tc := range tests {
		if got := formatWikiDate(tc.raw); got != tc.want {
			t.Fatalf("formatWikiDate(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestWorldDateFromMap(t *testing.T) {
	world := map[string]any{
		"created_at":  "2018-01-01T00:00:00.000Z",
		"releaseDate": "2019-06-15T00:00:00.000Z",
		"updated_at":  "2026-07-03T12:00:00.000Z",
	}
	if got := worldDateFromMap(world); got != "July 3, 2026" {
		t.Fatalf("worldDateFromMap() = %q, want July 3, 2026", got)
	}
}
