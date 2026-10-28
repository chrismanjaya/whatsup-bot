package integration

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const (
	mcpTokenPrefix    = "wsb_"
	mcpTokenByteCount = 32
)

// newMCPToken returns a random token with 256 bits of entropy.
func newMCPToken() (string, error) {
	buf := make([]byte, mcpTokenByteCount)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return mcpTokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

// hashMCPToken is what gets stored and looked up, so a leaked database does not leak usable tokens.
func hashMCPToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
