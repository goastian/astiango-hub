package middlewares

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goastian/astiango-hub/core/utils"
)

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")
		c.Next()
	}
}

func RequestBodyLimitMiddleware() gin.HandlerFunc {
	limit := utils.GetAPIRequestBodyLimit()
	return func(c *gin.Context) {
		if c.Request.ContentLength > limit {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

// CSRFMiddleware protects browser cookie sessions without imposing a CSRF token
// on bearer-token API clients. Stateful browser clients must send a matching
// X-CSRF-Token header and astiango_csrf cookie for unsafe requests.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions || c.GetHeader("Authorization") != "" {
			c.Next()
			return
		}
		csrfCookie, err := c.Request.Cookie("astiango_csrf")
		if err != nil || csrfCookie.Value == "" || !constantTimeEqual(csrfCookie.Value, c.GetHeader("X-CSRF-Token")) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Next()
	}
}

func constantTimeEqual(left, right string) bool {
	if len(left) != len(right) || left == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
