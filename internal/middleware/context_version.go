package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/randdotdev/e-campus-server/internal/ctxversion"
)

const ContextVersionHeader = "X-Context-Version"

// ContextVersion writes the user's current context version to X-Context-Version on every protected response.
func ContextVersion(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID != uuid.Nil {
			v := ctxversion.Get(c.Request.Context(), rdb, userID)
			c.Header(ContextVersionHeader, ctxversion.Header(v))
		}
		c.Next()
	}
}
