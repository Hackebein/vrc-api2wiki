package steam

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const steamGuardAlphabet = "23456789BCDFGHJKMNPQRTVWXY"

func GenerateSteamGuardCode(sharedSecret string, at time.Time) (string, error) {
	secret := strings.TrimSpace(sharedSecret)
	if secret == "" {
		return "", fmt.Errorf("empty STEAM_SHARED_SECRET")
	}
	key, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		key, err = base64.RawStdEncoding.DecodeString(secret)
		if err != nil {
			return "", fmt.Errorf("decode STEAM_SHARED_SECRET (expected base64): %w", err)
		}
	}
	if at.IsZero() {
		at = time.Now()
	}
	counter := uint64(at.Unix() / 30)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	fullCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)

	var code [5]byte
	for i := 0; i < 5; i++ {
		code[i] = steamGuardAlphabet[fullCode%uint32(len(steamGuardAlphabet))]
		fullCode /= uint32(len(steamGuardAlphabet))
	}
	return string(code[:]), nil
}
