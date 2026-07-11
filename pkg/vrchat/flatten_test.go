package vrchat

import "testing"

func TestFlattenWorld(t *testing.T) {
	world := map[string]any{
		"id":   "wrld_b2d24c29-1ded-4990-a90d-dd6dcc440300",
		"name": "Japan Shrine",
		"tags": []any{"japan", "shrine"},
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
