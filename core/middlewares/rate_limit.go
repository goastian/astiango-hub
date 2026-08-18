package middlewares

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateWindow struct {
	start time.Time
	count int
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateWindow
	now     func() time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{entries: make(map[string]rateWindow), now: time.Now}
}

func (l *RateLimiter) Allow(key string, limit int, window time.Duration) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entries[key]
	if entry.start.IsZero() || now.Sub(entry.start) >= window {
		entry = rateWindow{start: now}
	}
	if entry.count >= limit {
		return false
	}
	entry.count++
	l.entries[key] = entry
	return true
}

var apiRateLimiter = NewRateLimiter()

func AbuseProtectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		category, limit := requestRateCategory(c.Request.URL.Path)
		clientIP := c.ClientIP()
		if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil && c.GetHeader("X-Forwarded-For") == "" {
			clientIP = host
		}
		if !apiRateLimiter.Allow(category+":"+clientIP, limit, time.Minute) {
			c.Header("Retry-After", "60")
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}

func requestRateCategory(path string) (string, int) {
	switch {
	case strings.Contains(path, "/login"), strings.Contains(path, "/refresh"):
		return "login", 10
	case strings.HasPrefix(path, "/sync/"):
		return "sync", 120
	case strings.HasPrefix(path, "/export/"):
		return "export", 10
	default:
		return "api", 300
	}
}
