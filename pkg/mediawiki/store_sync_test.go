package mediawiki

import (
	"strings"
	"testing"

	"github.com/Hackebein/vrc-api2wiki/pkg/vrchat"
)

func TestStoreListingsPageTitle(t *testing.T) {
	got := StoreListingsPageTitle(2026, `{"MinVersion":1878} Open Beta Profile Decorations`)
	want := "Template:VRChatStoreListings/2026/Open Beta Profile Decorations"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if StoreListingsPageTitle(2026, "New Accessories") != "Template:VRChatStoreListings/2026/New Accessories" {
		t.Fatal(StoreListingsPageTitle(2026, "New Accessories"))
	}
}

func TestStoreAvatarStylePageTitles(t *testing.T) {
	if got := StoreAvatarStylePageTitle(2026, "Anime"); got != "Template:VRChatStoreListings/2026/Avatars/Anime" {
		t.Fatal(got)
	}
	if got := StoreAvatarStylesIndexPageTitle(2026); got != "Template:VRChatStoreListings/2026/Avatars" {
		t.Fatal(got)
	}
}

func TestStoreAvatarAuthorPageTitles(t *testing.T) {
	if got := StoreAvatarAuthorsIndexPageTitle(2026); got != "Template:VRChatStoreListings/2026/Avatars/Authors" {
		t.Fatal(got)
	}
	if got := StoreAvatarAuthorPageTitle(2026, "PackAuthor"); got != "Template:VRChatStoreListings/2026/Avatars/Authors/PackAuthor" {
		t.Fatal(got)
	}
	if got := StoreAvatarAuthorPageTitle(2026, `Name:With|Chars`); got != "Template:VRChatStoreListings/2026/Avatars/Authors/Name With Chars" {
		t.Fatal(got)
	}
}

func TestDisplayShelfTitleStripsMinVersion(t *testing.T) {
	got := vrchat.DisplayShelfTitle(`{"MinVersion":1878} Open Beta Profile Decorations`)
	if got != "Open Beta Profile Decorations" {
		t.Fatalf("got %q", got)
	}
}

func TestListingImageFileDescriptionUsesWorldLicense(t *testing.T) {
	got := ListingImageFileDescription("Vega’s Tiara", "prod_abc", "file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1")
	for _, want := range []string{
		"== Summary ==",
		"{{File information",
		"|description = Store listing image for Vega’s Tiara.",
		"|source      = VRChat API",
		"|author      = VRChat",
		"|additional_information = prod_abc file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@1",
		"== Licensing ==",
		"{{license VRC public section8}}",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "fairuse") || strings.Contains(got, "{{License|") {
		t.Fatalf("unexpected fairuse/License template:\n%s", got)
	}
}
