// Package permission provides role-based access control utilities.
package permission

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
)

// Permission levels
const (
	SuperAdmin = "super_admin"
	Admin      = "admin"
	Operator   = "operator"
	Viewer     = "viewer"
)

// Scope levels
const (
	ScopeUniversity = "university"
	ScopeCollege    = "college"
	ScopeDepartment = "department"
	ScopeProgram    = "program"
)

var permissionRank = map[string]int{
	SuperAdmin: 4,
	Admin:      3,
	Operator:   2,
	Viewer:     1,
}

var scopeRank = map[string]int{
	ScopeUniversity: 4,
	ScopeCollege:    3,
	ScopeDepartment: 2,
	ScopeProgram:    1,
}

// Check verifies if any of the given roles grants the required permission at the specified scope.
// For university scope, broader access is automatically granted.
// For other scopes, the scopeID must match exactly unless the role is at university level.
func Check(roles []auth.RoleClaim, requiredPermission, scopeType string, scopeID *uuid.UUID) bool {
	requiredRank := permissionRank[requiredPermission]
	requiredScopeRank := scopeRank[scopeType]

	for _, role := range roles {
		roleRank := permissionRank[role.Permission]
		roleScopeRank := scopeRank[role.ScopeType]

		if roleRank < requiredRank {
			continue
		}

		// Only university scope (no specific ID) can auto-allow broader access
		// Other broader scopes need explicit hierarchy verification in handlers
		if roleScopeRank > requiredScopeRank && role.ScopeType == ScopeUniversity {
			return true
		}

		if roleScopeRank == requiredScopeRank {
			if scopeID == nil || role.ScopeID == nil {
				return true
			}
			if *role.ScopeID == *scopeID {
				return true
			}
		}
	}

	return false
}

// CheckFromContext is a convenience function that extracts roles from gin context.
func CheckFromContext(c *gin.Context, requiredPermission, scopeType string, scopeID *uuid.UUID) bool {
	roles := middleware.GetUserRoles(c)
	return Check(roles, requiredPermission, scopeType, scopeID)
}

// HasAdminAccess checks if user has admin or higher access at the given scope.
func HasAdminAccess(c *gin.Context, scopeType string, scopeID *uuid.UUID) bool {
	return CheckFromContext(c, Admin, scopeType, scopeID)
}

// HasUniversityAdmin checks if user has university-level admin access.
func HasUniversityAdmin(c *gin.Context) bool {
	return CheckFromContext(c, Admin, ScopeUniversity, nil)
}
