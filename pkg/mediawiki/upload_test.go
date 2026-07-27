package mediawiki

import "testing"

func TestFileSourceRef(t *testing.T) {
	tests := []struct {
		fileID  string
		version int
		want    string
	}{
		{"file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", 3, "file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee@3"},
		{"file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", 0, "file_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"},
		{"", 1, ""},
		{"  file_x  ", 2, "file_x@2"},
	}
	for _, tc := range tests {
		if got := FileSourceRef(tc.fileID, tc.version); got != tc.want {
			t.Fatalf("FileSourceRef(%q, %d) = %q, want %q", tc.fileID, tc.version, got, tc.want)
		}
	}
}
