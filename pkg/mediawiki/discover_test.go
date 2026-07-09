package mediawiki

import "testing"

func TestParseInfoboxWorldIDs(t *testing.T) {
	wikitext := `Some intro text.
{{Infobox/World
|name=Japan Shrine
|image=File:Japan_Shrine.webp
|author=RootGentle
|platforms=PC, Android
|tags=japan, shrine, jp
|id=wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300
|labs=
|published=March 17, 2018
}}
More text.
{{Infobox/World|name=Other|id=wrld_00000000-0000-4000-8000-000000000001}}
{{Infobox/Avatar|id=avtr_ignored}}
{{Infobox/World|name=No ID}}
{{Infobox/Official World
|name=Official Hub
|id=wrld_00000000-0000-4000-8000-000000000002
}}
`

	ids := ParseInfoboxWorldIDs(wikitext)
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d: %v", len(ids), ids)
	}
	if ids[0] != "wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300" {
		t.Fatalf("unexpected first id: %s", ids[0])
	}
	if ids[1] != "wrld_00000000-0000-4000-8000-000000000001" {
		t.Fatalf("unexpected second id: %s", ids[1])
	}
	if ids[2] != "wrld_00000000-0000-4000-8000-000000000002" {
		t.Fatalf("unexpected third id: %s", ids[2])
	}
}

func TestParseInfoboxWorldIDsFromLinkParam(t *testing.T) {
	wikitext := `{{Infobox/Official World
|name=VRChat Home
|image=VrcHomeThumb.png
|developer=VRChat
|platform= PC, Android, iOS
|link={{World link|wrld_4432ea9b-729c-46e3-8eaf-846aa0a37fdd|VRChat Home}}
|published=Sep 30, 2022
}}
{{Infobox/Official World
|name=Steel 'n' Gold
|link={{VRC link|https://vrchat.com/home/world/wrld_d0a68029-96da-4e9f-aa0a-e4f8c6f5d6d9|Steel 'n' Gold}}
}}
{{Infobox/World
|name=Explicit wins
|id=wrld_00000000-0000-4000-8000-00000000000a
|link={{World link|wrld_00000000-0000-4000-8000-00000000000b|Ignored}}
}}
`

	ids := ParseInfoboxWorldIDs(wikitext)
	if len(ids) != 3 {
		t.Fatalf("expected 3 ids, got %d: %v", len(ids), ids)
	}
	if ids[0] != "wrld_4432ea9b-729c-46e3-8eaf-846aa0a37fdd" {
		t.Fatalf("unexpected first id: %s", ids[0])
	}
	if ids[1] != "wrld_d0a68029-96da-4e9f-aa0a-e4f8c6f5d6d9" {
		t.Fatalf("unexpected second id: %s", ids[1])
	}
	if ids[2] != "wrld_00000000-0000-4000-8000-00000000000a" {
		t.Fatalf("explicit id should take precedence, got: %s", ids[2])
	}
}

func TestWorldAliasWikitextSingleTarget(t *testing.T) {
	got := WorldAliasWikitext("wrld_00000000-0000-4000-8000-000000000001", []string{"Community:Prison Escape"})
	want := "#REDIRECT [[Community:Prison Escape]]\n"
	if got != want {
		t.Fatalf("unexpected redirect wikitext:\n got: %q\nwant: %q", got, want)
	}
}

func TestWorldAliasWikitextMultipleTargets(t *testing.T) {
	id := "wrld_00000000-0000-4000-8000-000000000001"
	got := WorldAliasWikitext(id, []string{"Community:Zeta", "Community:Alpha"})
	want := "{{#ifexist:Template:World/" + id + "/name|{{DISPLAYTITLE:{{World/" + id + "/name}}}}}}\n" +
		"{{Disambiguation}}\n* [[Community:Alpha]]\n* [[Community:Zeta]]\n"
	if got != want {
		t.Fatalf("unexpected disambiguation wikitext:\n got: %q\nwant: %q", got, want)
	}
}

func TestIsValidWorldID(t *testing.T) {
	if !IsValidWorldID("wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300") {
		t.Fatal("expected valid world id")
	}
	if IsValidWorldID("avtr_b2d24c29-1ded-4990-a90d-dd6dcc440300") {
		t.Fatal("expected avatar id to be invalid")
	}
	if IsValidWorldID("wrld_short") {
		t.Fatal("expected short id to be invalid")
	}
}
