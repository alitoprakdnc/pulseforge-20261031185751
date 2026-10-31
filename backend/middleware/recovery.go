package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery returns a Gin middleware that recovers from any panics,
// logs the error with stack trace using the global zap logger,
// and returns a JSON 500 response to the client.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// Capture stack trace
				stack := debug.Stack()

				// Log the panic with contextual information
				zap.L().Error("panic recovered",
					zap.Any("panic", rec),
					zap.ByteString("stack", stack),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
				)

				// If the response has not been written yet, send a JSON error.
				if !c.Writer.Written() {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": "internal server error",
					})
				} else {
					// If something was already written, just abort the request.
					c.Abort()
				}
			}
		}()

		// Continue down the chain to the next handler.
		c.Next()
	}
}