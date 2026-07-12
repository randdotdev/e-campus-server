package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/identity"
)

// Gin context keys under which the auth middleware (identity/http) stores the
// authenticated caller, and from which every context's handlers read it.
const (
	// UserIDKey holds the caller's uuid.UUID.
	UserIDKey = "user_id"
	// UserEmailKey holds the caller's email string.
	UserEmailKey = "user_email"
	// UserRoleKey holds the caller's *identity.RoleClaim (nil if roleless).
	UserRoleKey = "user_role"
	// UserClaimsKey holds the caller's full *identity.JWTClaims.
	UserClaimsKey = "user_claims"
)

// GetUserID returns the authenticated caller's id, or uuid.Nil outside an
// authenticated request.
func GetUserID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(UserIDKey); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

// GetUserEmail returns the authenticated caller's email, or "" outside an
// authenticated request.
func GetUserEmail(c *gin.Context) string {
	if email, exists := c.Get(UserEmailKey); exists {
		return email.(string)
	}
	return ""
}

// GetUserRole returns the caller's institutional role claim, or nil if they
// hold none (or outside an authenticated request).
func GetUserRole(c *gin.Context) *identity.RoleClaim {
	if role, exists := c.Get(UserRoleKey); exists {
		if role == nil {
			return nil
		}
		return role.(*identity.RoleClaim)
	}
	return nil
}

// GetUserClaims returns the caller's full token claims, or nil outside an
// authenticated request.
func GetUserClaims(c *gin.Context) *identity.JWTClaims {
	if claims, exists := c.Get(UserClaimsKey); exists {
		return claims.(*identity.JWTClaims)
	}
	return nil
}
