package vrchat

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// GenerateTOTP returns a 6-digit TOTP code for the given base32 secret (RFC 6238).
func GenerateTOTP(secret string) (string, error) {
	secret = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	if strings.HasPrefix(secret, "OTPAUTH://") || strings.HasPrefix(secret, "OTPAUTH:") {
		return "", fmt.Errorf("VRCHAT_TOTP_SECRET must be the raw base32 secret, not an otpauth URI")
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		key, err = base32.StdEncoding.DecodeString(secret)
		if err != nil {
			return "", fmt.Errorf("decode TOTP secret: %w", err)
		}
	}
	counter := uint64(time.Now().Unix() / 30)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	return fmt.Sprintf("%06d", code%1000000), nil
}
