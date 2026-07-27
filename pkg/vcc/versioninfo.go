package vcc

import (
	"fmt"
	"unicode/utf16"
)

// productVersionFromPE reads ProductVersion (or FileVersion) from a PE VERSIONINFO block.
func productVersionFromPE(data []byte) (string, error) {
	for _, key := range []string{"ProductVersion", "FileVersion"} {
		if v, ok := utf16StringAfterKey(data, key); ok {
			return v, nil
		}
	}
	return "", fmt.Errorf("ProductVersion/FileVersion not found")
}

func utf16StringAfterKey(data []byte, key string) (string, bool) {
	needle := encodeUTF16LE(key)
	idx := indexBytes(data, needle)
	if idx < 0 {
		return "", false
	}
	i := idx + len(needle)
	// Skip padding nulls between key and value (VERSIONINFO alignment).
	for i+1 < len(data) && data[i] == 0 && data[i+1] == 0 {
		i += 2
	}
	var u16 []uint16
	for i+1 < len(data) {
		unit := uint16(data[i]) | uint16(data[i+1])<<8
		i += 2
		if unit == 0 {
			break
		}
		u16 = append(u16, unit)
	}
	if len(u16) == 0 {
		return "", false
	}
	s := string(utf16.Decode(u16))
	if s == "" {
		return "", false
	}
	return s, true
}

func encodeUTF16LE(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	out := make([]byte, len(u16)*2)
	for i, u := range u16 {
		out[i*2] = byte(u)
		out[i*2+1] = byte(u >> 8)
	}
	return out
}

func indexBytes(haystack, needle []byte) int {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return -1
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
