package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		if path == "/health" {
			return
		}

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("request_id", GetRequestID(c)),
		}

		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		switch {
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}
