package vrchat

import (
	"regexp"
	"testing"
)

func TestGenerateTOTPRejectsURI(t *testing.T) {
	_, err := GenerateTOTP("otpauth://totp/x?secret=AB")
	if err == nil {
		t.Fatal("expected error for otpauth URI")
	}
}

func TestGenerateTOTPFormat(t *testing.T) {
	// Well-known test vector secret "12345678901234567890" in base32
	code, err := GenerateTOTP("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
		t.Fatalf("unexpected code %q", code)
	}
}
