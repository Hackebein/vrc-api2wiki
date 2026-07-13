package vrchat

import "testing"

func TestFlattenWorld(t *testing.T) {
	world := map[string]any{
		"id":   "wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300",
		"name": "Japan Shrine",
		"tags": []any{"author_tag_japan", "author_tag_shrine"},
		"instances": []any{
			[]any{"ignored", "data"},
		},
		"unityPackages": []any{
			map[string]any{
				"platform":   "standalonewindows",
				"created_at": "2018-03-17T08:58:42.296Z",
				"fileSize":   float64(12345),
				"assetUrlObject": map[string]any{
					"ignored": true,
				},
			},
			map[string]any{
				"platform":   "android",
				"created_at": "2019-01-01T00:00:00.000Z",
			},
		},
		"defaultContentSettings": map[string]any{
			"stickers": false,
			"drones":   true,
		},
	}

	pages := FlattenWorld(world)

	if pages["name"] != "Japan Shrine" {
		t.Fatalf("unexpected name: %q", pages["name"])
	}
	if pages["tags"] != "japan, shrine" {
		t.Fatalf("unexpected tags: %q", pages["tags"])
	}
	if _, ok := pages["systemTags"]; ok {
		t.Fatalf("unexpected systemTags: %q", pages["systemTags"])
	}
	if pages["platforms"] != "PC, Android" {
		t.Fatalf("unexpected platforms: %q", pages["platforms"])
	}
	if pages["unityPackages/standalonewindows/created_at"] != "2018-03-17T08:58:42.296Z" {
		t.Fatalf("unexpected unity package created_at: %q", pages["unityPackages/standalonewindows/created_at"])
	}
	if pages["defaultContentSettings/stickers"] != "false" {
		t.Fatalf("unexpected stickers setting: %q", pages["defaultContentSettings/stickers"])
	}
	if _, ok := pages["instances"]; ok {
		t.Fatal("instances should be excluded")
	}
	if _, ok := pages["unityPackages/standalonewindows/assetUrlObject"]; ok {
		t.Fatal("assetUrlObject should be excluded")
	}
}

func TestFlattenWorldSkipsVolatileAndRedundantFields(t *testing.T) {
	world := map[string]any{
		"id":                "wrld_00000000-0000-4000-8000-000000000001",
		"name":              "Test World",
		"thumbnailImageUrl": "https://example.com/thumb.png",
		"occupants":         float64(3),
		"privateOccupants":  float64(1),
		"publicOccupants":   float64(2),
	}

	pages := FlattenWorld(world)
	for _, key := range []string{"thumbnailImageUrl", "occupants", "privateOccupants", "publicOccupants"} {
		if _, ok := pages[key]; ok {
			t.Fatalf("%s should be excluded, got %q", key, pages[key])
		}
	}
}

func TestFlattenWorldObjectKeyedUnityPackages(t *testing.T) {
	world := map[string]any{
		"id": "wrld_00000000-0000-4000-8000-000000000001",
		"unityPackages": map[string]any{
			"standalonewindows": map[string]any{
				"created_at": "2018-03-17T08:58:42.296Z",
				"fileSize":   float64(999),
			},
			"android": map[string]any{
				"created_at": "2019-01-01T00:00:00.000Z",
			},
		},
	}

	pages := FlattenWorld(world)
	if pages["unityPackages/standalonewindows/fileSize"] != "999" {
		t.Fatalf("unexpected fileSize: %q", pages["unityPackages/standalonewindows/fileSize"])
	}
	if pages["platforms"] != "Android, PC" {
		t.Fatalf("unexpected platforms: %q", pages["platforms"])
	}
}

func TestFlattenWorldSplitsDisplayAndSystemTags(t *testing.T) {
	world := map[string]any{
		"id": "wrld_dd036610-a246-4f52-bf01-9d7cea3405d7",
		"tags": []any{
			"author_tag_game",
			"author_tag_udon",
			"author_tag_quest",
			"system_approved",
			"admin_approved",
			"feature_drones_disabled",
		},
	}

	pages := FlattenWorld(world)

	if pages["tags"] != "udon, quest" {
		t.Fatalf("unexpected tags: %q", pages["tags"])
	}
	if pages["listing"] != "Games" {
		t.Fatalf("unexpected listing: %q", pages["listing"])
	}
	if _, ok := pages["systemTags"]; ok {
		t.Fatalf("unexpected systemTags: %q", pages["systemTags"])
	}
}

func TestFlattenWorldDisplayTagsOnly(t *testing.T) {
	world := map[string]any{
		"id": "wrld_5edb4d7d-2400-49f8-bae1-a8eb48a258e2",
		"tags": []any{
			"author_tag_visualizer",
			"author_tag_music",
			"admin_spotlight_pc",
			"system_approved",
		},
	}

	pages := FlattenWorld(world)

	if pages["tags"] != "visualizer, music" {
		t.Fatalf("unexpected tags: %q", pages["tags"])
	}
	if _, ok := pages["listing"]; ok {
		t.Fatalf("unexpected listing: %q", pages["listing"])
	}
	if _, ok := pages["content"]; ok {
		t.Fatalf("unexpected content: %q", pages["content"])
	}
	if _, ok := pages["systemTags"]; ok {
		t.Fatalf("unexpected systemTags: %q", pages["systemTags"])
	}
}

func TestFlattenWorldListingTags(t *testing.T) {
	world := map[string]any{
		"id": "wrld_00000000-0000-4000-8000-000000000002",
		"tags": []any{
			"author_tag_avatar",
			"content_featured",
		},
	}

	pages := FlattenWorld(world)

	if _, ok := pages["tags"]; ok {
		t.Fatalf("tags should be omitted, got %q", pages["tags"])
	}
	if pages["listing"] != "Avatar Worlds, Featured" {
		t.Fatalf("unexpected listing: %q", pages["listing"])
	}
}

func TestFlattenWorldContentTags(t *testing.T) {
	world := map[string]any{
		"id": "wrld_00000000-0000-4000-8000-000000000003",
		"tags": []any{
			"content_adult",
			"content_horror",
			"admin_approved",
		},
	}

	pages := FlattenWorld(world)

	if pages["content"] != "Adult, Horror" {
		t.Fatalf("unexpected content: %q", pages["content"])
	}
	if _, ok := pages["systemTags"]; ok {
		t.Fatalf("unexpected systemTags: %q", pages["systemTags"])
	}
}

func TestFlattenWorldEmptyAuthorTags(t *testing.T) {
	world := map[string]any{
		"id":   "wrld_00000000-0000-4000-8000-000000000001",
		"tags": []any{"system_approved", "admin_featured"},
	}

	pages := FlattenWorld(world)

	if _, ok := pages["tags"]; ok {
		t.Fatalf("tags should be omitted when no author_tag_* entries, got %q", pages["tags"])
	}
	if _, ok := pages["listing"]; ok {
		t.Fatalf("listing should be omitted, got %q", pages["listing"])
	}
	if _, ok := pages["content"]; ok {
		t.Fatalf("content should be omitted, got %q", pages["content"])
	}
	if _, ok := pages["systemTags"]; ok {
		t.Fatalf("unexpected systemTags: %q", pages["systemTags"])
	}
}
