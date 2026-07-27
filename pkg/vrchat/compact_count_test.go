package vrchat

import "testing"

func TestFormatCompactCount(t *testing.T) {
	tests := []struct {
		n     int64
		coeff string
		unit  string
	}{
		{0, "0", ""},
		{999, "999", ""},
		{1000, "1", "k"},
		{28817, "28.8", "k"},
		{475079, "475", "k"},
		{1000000, "1", "M"},
		{1413125, "1.41", "M"},
		{2500000000, "2.5", "B"},
		{1000000000000, "1", "T"},
		{1e15, "1", "Qa"},
		{1e18, "1", "Qi"},
	}

	for _, tt := range tests {
		coeff, unit := FormatCompactCount(tt.n)
		if coeff != tt.coeff || unit != tt.unit {
			t.Fatalf("FormatCompactCount(%d) = (%q, %q), want (%q, %q)",
				tt.n, coeff, unit, tt.coeff, tt.unit)
		}
	}
}

func TestCompactCountWikitext(t *testing.T) {
	tests := []struct {
		n    int64
		want string
	}{
		{999, "{{Compact number|999}}"},
		{28817, "{{Compact number|28.8|k}}"},
		{475079, "{{Compact number|475|k}}"},
		{1413125, "{{Compact number|1.41|M}}"},
		{1000000000000, "{{Compact number|1|T}}"},
	}

	for _, tt := range tests {
		got := CompactCountWikitext(tt.n)
		if got != tt.want {
			t.Fatalf("CompactCountWikitext(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
