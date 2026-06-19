// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/auth"
	"github.com/randdotdev/e-campus-server/internal/response"
)

const (
	UserIDKey     = "user_id"
	UserEmailKey  = "user_email"
	UserRoleKey   = "user_role"
	UserClaimsKey = "user_claims"
)

func Auth(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := auth.ExtractToken(c)
		if token == "" {
			response.Unauthorized(c, "authorization required")
			c.Abort()
			return
		}

		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			if err == auth.ErrTokenExpired {
				response.Err(c, 401, "TOKEN_EXPIRED", "access token expired")
				c.Abort()
				return
			}
			response.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UserEmailKey, claims.Email)
		c.Set(UserRoleKey, claims.Role)
		c.Set(UserClaimsKey, claims)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(UserIDKey); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get(UserEmailKey); exists {
		return email.(string)
	}
	return ""
}

func GetUserRole(c *gin.Context) *auth.RoleClaim {
	if role, exists := c.Get(UserRoleKey); exists {
		if role == nil {
			return nil
		}
		return role.(*auth.RoleClaim)
	}
	return nil
}

func GetUserClaims(c *gin.Context) *auth.JWTClaims {
	if claims, exists := c.Get(UserClaimsKey); exists {
		return claims.(*auth.JWTClaims)
	}
	return nil
}
