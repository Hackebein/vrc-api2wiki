package vrchat

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type compactUnit struct {
	threshold int64
	divisor   float64
	unit      string
}

// compactUnits is the short-scale ladder from thousand through quintillion.
// Order matters: first matching threshold wins (largest first).
var compactUnits = []compactUnit{
	{threshold: 1e18, divisor: 1e18, unit: "Qi"},
	{threshold: 1e15, divisor: 1e15, unit: "Qa"},
	{threshold: 1e12, divisor: 1e12, unit: "T"},
	{threshold: 1e9, divisor: 1e9, unit: "B"},
	{threshold: 1e6, divisor: 1e6, unit: "M"},
	{threshold: 1e3, divisor: 1e3, unit: "k"},
}

// FormatCompactCount returns a coefficient string and unit for n using ~3
// significant figures (e.g. 28817 → "28.8","k"; 1413125 → "1.41","M").
// For n < 1000 the coefficient is the full integer and unit is empty.
func FormatCompactCount(n int64) (coeff string, unit string) {
	if n < 0 {
		n = -n
	}
	if n < 1000 {
		return strconv.FormatInt(n, 10), ""
	}

	for _, u := range compactUnits {
		if n >= u.threshold {
			scaled := float64(n) / u.divisor
			return formatCompactCoeff(scaled), u.unit
		}
	}
	return strconv.FormatInt(n, 10), ""
}

func formatCompactCoeff(scaled float64) string {
	var decimals int
	switch {
	case scaled < 10:
		decimals = 2
	case scaled < 100:
		decimals = 1
	default:
		decimals = 0
	}

	factor := math.Pow(10, float64(decimals))
	rounded := math.Round(scaled*factor) / factor
	s := strconv.FormatFloat(rounded, 'f', decimals, 64)
	if decimals > 0 {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// CompactCountPageKeys are FlattenWorld subpaths stored as Compact number
// template calls. Their page bodies must not run through |/= sanitization.
var CompactCountPageKeys = []string{"visits", "favorites"}

// IsCompactCountPage reports whether subpath is written as Compact number wikitext.
func IsCompactCountPage(subpath string) bool {
	for _, key := range CompactCountPageKeys {
		if subpath == key {
			return true
		}
	}
	return false
}

// CompactCountWikitext returns a Compact number template call for n.
func CompactCountWikitext(n int64) string {
	coeff, unit := FormatCompactCount(n)
	if unit == "" {
		return fmt.Sprintf("{{Compact number|%s}}", coeff)
	}
	return fmt.Sprintf("{{Compact number|%s|%s}}", coeff, unit)
}

func int64FromAny(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) {
			return 0, false
		}
		return int64(math.Round(n)), true
	case int64:
		return n, true
	case int:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			f, err := n.Float64()
			if err != nil {
				return 0, false
			}
			return int64(math.Round(f)), true
		}
		return i, true
	default:
		return 0, false
	}
}
