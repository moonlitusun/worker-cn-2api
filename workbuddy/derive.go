// Package workbuddy implements the WorkBuddy / CodeBuddy OpenAI-compatible bridge.
package workbuddy

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

const defaultSalt = "worker-cn-2api-wb-salt-2026"

// DeriveDeviceID generates a stable, deterministic hardware device fingerprint
// for a given account identifier (UID or token prefix).
// This ensures device consistency per account and prevents upstream WAF/anti-abuse rejections.
func DeriveDeviceID(accountID string) string {
	salt := os.Getenv("WORKER_WORKBUDDY_SALT")
	if salt == "" {
		salt = defaultSalt
	}
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(accountID))
	sum := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("wb-%s-%s-%s", sum[:8], sum[8:16], sum[16:32])
}
