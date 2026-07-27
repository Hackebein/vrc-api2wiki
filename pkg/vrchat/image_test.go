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
