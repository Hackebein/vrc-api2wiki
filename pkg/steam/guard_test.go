package steam

import (
	"testing"
	"time"
)

func TestGenerateSteamGuardCodeKnownVector(t *testing.T) {
	// 20 zero bytes — typical Steam shared_secret length after base64 decode.
	secret := "AAAAAAAAAAAAAAAAAAAAAAAAAAA="
	code, err := GenerateSteamGuardCode(secret, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 5 {
		t.Fatalf("len %d code %q", len(code), code)
	}
	for _, r := range code {
		if !containsRune(steamGuardAlphabet, r) {
			t.Fatalf("invalid char %q in %q", r, code)
		}
	}
}

func TestGenerateSteamGuardCodeRequiresSecret(t *testing.T) {
	if _, err := GenerateSteamGuardCode("", time.Now()); err == nil {
		t.Fatal("expected error for empty secret")
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
