package vrchat

import "testing"

func TestFileIDFromURL(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{
			"https://api.vrchat.cloud/api/1/file/file_20bc75b1-b0f8-4707-967d-8de1ed98c2c9/1/file",
			"file_20bc75b1-b0f8-4707-967d-8de1ed98c2c9",
		},
		{
			"https://images.vrchat.cloud/file_20bc75b1-b0f8-4707-967d-8de1ed98c2c9/1",
			"file_20bc75b1-b0f8-4707-967d-8de1ed98c2c9",
		},
		{"https://example.com/no-file-id.png", ""},
		{"", ""},
	}
	for _, tc := range tests {
		if got := FileIDFromURL(tc.raw); got != tc.want {
			t.Fatalf("FileIDFromURL(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestImageExtensionFromBytes(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"png", append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, 0x00), "png"},
		{"jpeg", []byte{0xff, 0xd8, 0xff, 0xe0, 0x00}, "jpg"},
		{"gif87a", []byte("GIF87a.........."), "gif"},
		{"gif89a", []byte("GIF89a.........."), "gif"},
		{"webp", append(append([]byte("RIFF"), 0, 0, 0, 0), []byte("WEBP....")...), "webp"},
		{"empty", nil, ""},
		{"unknown", []byte("not-an-image"), ""},
	}
	for _, tc := range tests {
		if got := ImageExtensionFromBytes(tc.data); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

func TestResolveImageExtPrefersMagicOverFallback(t *testing.T) {
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0}
	if got := ResolveImageExt(jpeg, "png"); got != "jpg" {
		t.Fatalf("got %q want jpg (API said png but bytes are jpeg)", got)
	}
	if got := ResolveImageExt([]byte("nope"), "jpeg"); got != "jpg" {
		t.Fatalf("fallback normalize: got %q", got)
	}
	if got := ResolveImageExt(nil, ""); got != "png" {
		t.Fatalf("default: got %q", got)
	}
}

func TestNormalizeImageExt(t *testing.T) {
	if NormalizeImageExt(".JPEG") != "jpg" {
		t.Fatal(NormalizeImageExt(".JPEG"))
	}
	if NormalizeImageExt(" PNG ") != "png" {
		t.Fatal(NormalizeImageExt(" PNG "))
	}
}

func TestReplaceImageExt(t *testing.T) {
	tests := []struct {
		name, ext, want string
	}{
		{"ListingImage The Dub Sub.png", "jpg", "ListingImage The Dub Sub.jpg"},
		{"ListingImage The Dub Sub.jpg", "jpg", "ListingImage The Dub Sub.jpg"},
		{"ShelfIcon EDM", "png", "ShelfIcon EDM.png"},
		{"", "jpg", ""},
	}
	for _, tc := range tests {
		if got := ReplaceImageExt(tc.name, tc.ext); got != tc.want {
			t.Fatalf("ReplaceImageExt(%q, %q) = %q, want %q", tc.name, tc.ext, got, tc.want)
		}
	}
}

func TestImageExtensionHeaderAndURL(t *testing.T) {
	ext, err := imageExtension("image/jpeg; charset=binary", "https://cdn.example/x.bin")
	if err != nil || ext != "jpg" {
		t.Fatalf("got %q err %v", ext, err)
	}
	ext, err = imageExtension("application/octet-stream", "https://cdn.example/photo.PNG")
	if err != nil || ext != "png" {
		t.Fatalf("got %q err %v", ext, err)
	}
	_, err = imageExtension("text/plain", "https://cdn.example/noext")
	if err == nil {
		t.Fatal("expected error")
	}
}
