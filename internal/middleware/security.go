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

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
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

type ipLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

func newIPLimiter(rps int, burst int) *ipLimiter {
	return &ipLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

func (l *ipLimiter) getLimiter(ip string) *rate.Limiter {
	l.mu.RLock()
	limiter, exists := l.limiters[ip]
	l.mu.RUnlock()

	if exists {
		return limiter
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists = l.limiters[ip]
	if exists {
		return limiter
	}

	limiter = rate.NewLimiter(l.rps, l.burst)
	l.limiters[ip] = limiter
	return limiter
}

// RateLimiter limits requests per IP.
func RateLimiter(cfg RateLimiterConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	limiter := newIPLimiter(cfg.RPS, cfg.Burst)

	// Cleanup old entries every 10 minutes
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			limiter.mu.Lock()
			limiter.limiters = make(map[string]*rate.Limiter)
			limiter.mu.Unlock()
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
