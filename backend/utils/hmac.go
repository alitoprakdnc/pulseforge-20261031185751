package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateSignature creates a SHA‑256 HMAC for the given payload using the provided secret.
// It returns the signature as a lower‑case hex‑encoded string.
// An error is returned if the secret is empty or the HMAC cannot be computed.
func GenerateSignature(secret, payload []byte) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("secret cannot be empty")
	}
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(payload); err != nil {
		return "", fmt.Errorf("failed to write payload to HMAC: %w", err)
	}
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum), nil
}

// VerifySignature validates that the supplied hex‑encoded signature matches the HMAC
// of the payload computed with the given secret. The comparison is performed in
// constant time using hmac.Equal to mitigate timing attacks.
// It returns true when the signatures match, false otherwise.
// An error is returned if the secret is empty, the signature cannot be decoded,
// or the internal generation of the expected signature fails.
func VerifySignature(secret, payload []byte, signatureHex string) (bool, error) {
	if len(secret) == 0 {
		return false, fmt.Errorf("secret cannot be empty")
	}
	// Decode the provided signature from hex.
	providedSig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Compute the expected signature.
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(payload); err != nil {
		return false, fmt.Errorf("failed to write payload to HMAC: %w", err)
	}
	expectedSig := mac.Sum(nil)

	// Constant‑time comparison.
	if hmac.Equal(expectedSig, providedSig) {
		return true, nil
	}
	return false, nil
}