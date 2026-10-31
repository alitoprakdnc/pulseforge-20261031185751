package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/pulseforge/backend/logger"
)

// HMACMiddleware validates incoming webhook requests using a constant‑time HMAC comparison.
// It expects the request body to be signed with the shared secret (WEBHOOK_SECRET env var).
// Supported signature headers:
//   - X-Hub-Signature-256 (GitHub, format: sha256=abcdef...)
//   - X-Signature (generic, raw hex string)
// If validation fails the request is aborted with 401 Unauthorized.
func HMACMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the signature from the request headers.
		sigHeader := c.GetHeader("X-Hub-Signature-256")
		if sigHeader == "" {
			sigHeader = c.GetHeader("X-Signature")
		}
		if sigHeader == "" {
			logger.Error("missing HMAC signature header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing signature"})
			return
		}

		// GitHub sends signatures as "sha256=abcdef...", strip the prefix if present.
		var providedSig string
		if parts := bytes.Split([]byte(sigHeader), []byte("=")); len(parts) == 2 {
			providedSig = string(parts[1])
		} else {
			providedSig = sigHeader
		}

		// Read the raw request body.
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logger.Error("failed to read request body", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		// Restore the body so downstream handlers can read it.
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Load the shared secret from environment (or secret manager via config).
		secret := os.Getenv("WEBHOOK_SECRET")
		if secret == "" {
			logger.Error("webhook secret not configured")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server misconfiguration"})
			return
		}

		// Compute HMAC‑SHA256 of the payload.
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(bodyBytes)
		expectedMAC := mac.Sum(nil)
		expectedHex := hex.EncodeToString(expectedMAC)

		// Constant‑time comparison to mitigate timing attacks.
		if !hmac.Equal([]byte(expectedHex), []byte(providedSig)) {
			logger.Warn("HMAC validation failed",
				zap.String("provided", providedSig),
				zap.String("expected", expectedHex),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}

		// Signature is valid – continue processing.
		c.Next()
	}
}