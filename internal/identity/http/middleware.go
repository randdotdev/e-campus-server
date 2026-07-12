package http

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/randdotdev/e-campus-server/internal/identity"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ExtractToken pulls the bearer token from the Authorization header, falling
// back to a query param (browser WebSockets can't set headers).
func ExtractToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return c.Query("access_token")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}

// Auth is the authentication middleware guarding every protected route: it
// verifies the access token (stateless JWT parsing, no I/O) and stores the
// caller's identity under the shared/middleware context keys, where every
// context's handlers read it.
func Auth(auth *identity.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractToken(c)
		if token == "" {
			response.Unauthorized(c, "authorization required")
			c.Abort()
			return
		}

		claims, err := auth.ValidateAccessToken(token)
		if err != nil {
			if err == identity.ErrTokenExpired {
				response.Err(c, 401, "TOKEN_EXPIRED", "access token expired")
				c.Abort()
				return
			}
			response.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}

		c.Set(middleware.UserIDKey, claims.UserID)
		c.Set(middleware.UserEmailKey, claims.Email)
		c.Set(middleware.UserRoleKey, claims.Role)
		c.Set(middleware.UserClaimsKey, claims)
		c.Next()
	}
}
