package steam

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestReadUnityVersionFromSerializedFile(t *testing.T) {
	for _, tc := range []struct {
		name       string
		format     uint32
		headerSize int
		version    string
		want       string
	}{
		{"Unity 6 custom suffix", 22, 48, "6000.0.67f1-DWR\x00", "6000.0.67f1-DWR"},
		{"legacy format", 21, 20, "2022.3.22f1\x00", "2022.3.22f1"},
		{"unsupported format", 23, 48, "6000.0.67f1\x00", ""},
		{"missing terminator", 22, 48, "6000.0.67f1-DWR", ""},
		{"not an engine version", 22, 48, "2026.4.1p3-1939-4803d92ede\x00", ""},
		{"unterminated metadata", 22, 48, strings.Repeat("0", 200), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.headerSize)
			binary.BigEndian.PutUint32(data[8:12], tc.format)
			data = append(data, tc.version...)
			got, err := readUnityVersion(bytes.NewReader(data))
			if got != tc.want || (err != nil) != (tc.want == "") {
				t.Fatalf("got %q, %v, want %q", got, err, tc.want)
			}
		})
	}
}
