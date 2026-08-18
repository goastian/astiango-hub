package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	viper.Set("api.allow.origin", "https://console.example.test")
	viper.Set("api.allow.credentials", "true")
	t.Cleanup(viper.Reset)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, httptest.NewRequest(http.MethodGet, "/ok", nil))
	require.Empty(t, allowed.Header().Get("Access-Control-Allow-Origin"))

	request := httptest.NewRequest(http.MethodOptions, "/ok", nil)
	request.Header.Set("Origin", "https://console.example.test")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "https://console.example.test", response.Header().Get("Access-Control-Allow-Origin"))
	require.Equal(t, "true", response.Header().Get("Access-Control-Allow-Credentials"))

	blocked := httptest.NewRequest(http.MethodOptions, "/ok", nil)
	blocked.Header.Set("Origin", "https://evil.example.test")
	blockedResponse := httptest.NewRecorder()
	router.ServeHTTP(blockedResponse, blocked)
	require.Equal(t, http.StatusForbidden, blockedResponse.Code)
}

func TestAbuseAndCSRFProtection(t *testing.T) {
	limiter := NewRateLimiter()
	limiter.now = func() time.Time { return time.Unix(0, 0) }
	require.True(t, limiter.Allow("login:ip", 1, time.Minute))
	require.False(t, limiter.Allow("login:ip", 1, time.Minute))
	require.True(t, limiter.Allow("api:other", 1, time.Minute))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/update", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodPost, "/update", nil)
	req.AddCookie(&http.Cookie{Name: "astiango_csrf", Value: "csrf-token"})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, http.StatusForbidden, res.Code)
	req.Header.Set("X-CSRF-Token", "csrf-token")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	require.Equal(t, http.StatusNoContent, res.Code)
}

func TestRequestBodyLimitAndSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	viper.Set("api.limits.body_bytes", 4)
	t.Cleanup(viper.Reset)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware(), RequestBodyLimitMiddleware())
	router.POST("/payload", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	res := httptest.NewRecorder()
	router.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/payload", bytes.NewBufferString("12345")))
	require.Equal(t, http.StatusRequestEntityTooLarge, res.Code)
	require.Equal(t, "nosniff", res.Header().Get("X-Content-Type-Options"))
}
