package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// SecurityHeaders adds security headers to responses.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
}

// CORS handles Cross-Origin Resource Sharing.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]bool)
	for _, origin := range cfg.AllowedOrigins {
		allowedOrigins[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowedOrigins[origin] || allowedOrigins["*"] {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Vary", "Origin")

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Expose-Headers", ContextVersionHeader)
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimiterConfig holds rate limiter configuration.
type RateLimiterConfig struct {
	Enabled bool
	RPS     int
	Burst   int
}

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipLimiter struct {
	entries map[string]*ipEntry
	mu      sync.Mutex
	rps     rate.Limit
	burst   int
}

func newIPLimiter(rps int, burst int) *ipLimiter {
	return &ipLimiter{
		entries: make(map[string]*ipEntry),
		rps:     rate.Limit(rps),
		burst:   burst,
	}
}

func (l *ipLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	e, exists := l.entries[ip]
	if !exists {
		e = &ipEntry{limiter: rate.NewLimiter(l.rps, l.burst)}
		l.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter
}

func (l *ipLimiter) cleanup(ttl time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-ttl)
	for ip, e := range l.entries {
		if e.lastSeen.Before(cutoff) {
			delete(l.entries, ip)
		}
	}
}

// RateLimiter limits requests per IP.
func RateLimiter(cfg RateLimiterConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	limiter := newIPLimiter(cfg.RPS, cfg.Burst)

	go func() {
		for {
			time.Sleep(10 * time.Minute)
			limiter.cleanup(20 * time.Minute)
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.getLimiter(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}
		c.Next()
	}
}

// AuthRateLimiterConfig holds auth-specific rate limiter configuration.
type AuthRateLimiterConfig struct {
	Enabled       bool
	MaxAttempts   int
	WindowSeconds int
}

type authLimiter struct {
	attempts map[string][]time.Time
	mu       sync.Mutex
	max      int
	window   time.Duration
}

func newAuthLimiter(max int, windowSec int) *authLimiter {
	return &authLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   time.Duration(windowSec) * time.Second,
	}
}

func (l *authLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Filter old attempts
	attempts := l.attempts[key]
	valid := attempts[:0]
	for _, t := range attempts {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= l.max {
		l.attempts[key] = valid
		return false
	}

	l.attempts[key] = append(valid, now)
	return true
}

// AuthRateLimiter limits authentication attempts per IP.
func AuthRateLimiter(cfg AuthRateLimiterConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	limiter := newAuthLimiter(cfg.MaxAttempts, cfg.WindowSeconds)

	// Cleanup every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			limiter.mu.Lock()
			now := time.Now()
			cutoff := now.Add(-limiter.window)
			for key, attempts := range limiter.attempts {
				valid := attempts[:0]
				for _, t := range attempts {
					if t.After(cutoff) {
						valid = append(valid, t)
					}
				}
				if len(valid) == 0 {
					delete(limiter.attempts, key)
				} else {
					limiter.attempts[key] = valid
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many attempts, try again later",
			})
			return
		}
		c.Next()
	}
}
