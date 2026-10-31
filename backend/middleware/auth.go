package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"pulseforge/backend/config"
	"pulseforge/backend/logger"
	"pulseforge/backend/models"
)

// ctxKey is a private type to avoid context key collisions.
type ctxKey string

const userContextKey ctxKey = "user"

// Claims defines the JWT payload used by PulseForge.
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates a JWT (from Authorization header or HttpOnly cookie),
// loads the corresponding user from the database and injects it into the Gin context.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, ok := extractToken(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
			return
		}

		secret := []byte(config.Get().JWTSecret)

		// Parse and validate the token.
		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
			// Enforce HMAC signing method.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		})
		if err != nil {
			logger.Error("failed to parse JWT", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// Expiration check (RegisteredClaims already validates if we set jwt.WithLeeway, but we double‑check).
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			return
		}

		// Load the user from the DB to guarantee it still exists and to attach full model data.
		user, err := models.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil {
			logger.Error("user not found for token", zap.Uint("user_id", claims.UserID), zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		// Store the user in the Gin context for downstream handlers.
		c.Set(string(userContextKey), user)

		c.Next()
	}
}

// GetUser extracts the authenticated *models.User from the Gin context.
// Returns the user and a boolean indicating whether a user was present.
func GetUser(c *gin.Context) (*models.User, bool) {
	val, exists := c.Get(string(userContextKey))
	if !exists {
		return nil, false
	}
	user, ok := val.(*models.User)
	return user, ok
}

// extractToken retrieves the JWT from the Authorization header (Bearer) or
// from the HttpOnly cookie named "auth_token". Returns the token string and a
// boolean indicating whether a token was found.
func extractToken(c *gin.Context) (string, bool) {
	// Prefer Authorization header.
	if authHeader := c.GetHeader("Authorization"); authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") && parts[1] != "" {
			return parts[1], true
		}
	}

	// Fallback to cookie.
	if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
		return cookie, true
	}

	return "", false
}