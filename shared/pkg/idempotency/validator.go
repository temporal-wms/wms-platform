package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	// keyPattern matches valid idempotency key formats
	// Allows alphanumeric characters, hyphens, and underscores
	keyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
)

// ValidateKey validates an idempotency key format and length
func ValidateKey(key string) error {
	if key == "" {
		return ErrKeyRequired
	}

	if len(key) > DefaultMaxKeyLength {
		return ErrKeyTooLong
	}

	// Check if key contains only valid characters
	if !keyPattern.MatchString(key) {
		return ErrKeyInvalid
	}

	return nil
}

// ValidateKeyWithMaxLength validates an idempotency key with a custom max length
func ValidateKeyWithMaxLength(key string, maxLength int) error {
	if key == "" {
		return ErrKeyRequired
	}

	if len(key) > maxLength {
		return ErrKeyTooLong
	}

	if !keyPattern.MatchString(key) {
		return ErrKeyInvalid
	}

	return nil
}

// ComputeFingerprint computes a SHA256 fingerprint of the request body
// This is used to detect if retry requests have different parameters
func ComputeFingerprint(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}

// NormalizeKey normalizes an idempotency key by trimming whitespace
func NormalizeKey(key string) string {
	return strings.TrimSpace(key)
}

// IsValidKeyChar returns true if the character is valid in an idempotency key
func IsValidKeyChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_'
}
