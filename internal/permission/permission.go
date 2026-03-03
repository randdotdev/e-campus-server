// Package permission provides centralized role-based access control.
package permission

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
)

const (
	SuperAdmin = "super_admin"
	Admin      = "admin"
	Operator   = "operator"
	Viewer     = "viewer"
)

const (
	ScopePlatform   = "platform"
	ScopeUniversity = "university"
	ScopeCollege    = "college"
	ScopeDepartment = "department"
	ScopeProgram    = "program"
)

const (
	RoleTeacher   = "teacher"
	RoleAssistant = "assistant"
)

var permissionRank = map[string]int{
	SuperAdmin: 4,
	Admin:      3,
	Operator:   2,
	Viewer:     1,
}

var scopeRank = map[string]int{
	ScopePlatform:   5,
	ScopeUniversity: 4,
	ScopeCollege:    3,
	ScopeDepartment: 2,
	ScopeProgram:    1,
}

func Check(role *auth.RoleClaim, requiredPermission, scopeType string, scopeID *uuid.UUID) bool {
	if role == nil {
		return false
	}

	requiredRank := permissionRank[requiredPermission]
	requiredScopeRank := scopeRank[scopeType]

	roleRank := permissionRank[role.Permission]
	roleScopeRank := scopeRank[role.ScopeType]

	if roleRank < requiredRank {
		return false
	}

	if roleScopeRank > requiredScopeRank && (role.ScopeType == ScopePlatform || role.ScopeType == ScopeUniversity) {
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

	return false
}

func checkFromContext(c *gin.Context, permission, scope string, scopeID *uuid.UUID) bool {
	role := middleware.GetUserRole(c)
	return Check(role, permission, scope, scopeID)
}

func CanAdminPlatform(c *gin.Context) bool {
	return checkFromContext(c, Admin, ScopePlatform, nil)
}

func CanOperatePlatform(c *gin.Context) bool {
	return checkFromContext(c, Operator, ScopePlatform, nil)
}

func CanViewPlatform(c *gin.Context) bool {
	return checkFromContext(c, Viewer, ScopePlatform, nil)
}

func CanAdminUniversity(c *gin.Context) bool {
	return checkFromContext(c, Admin, ScopeUniversity, nil)
}

func CanOperateUniversity(c *gin.Context) bool {
	return checkFromContext(c, Operator, ScopeUniversity, nil)
}

func CanViewUniversity(c *gin.Context) bool {
	return checkFromContext(c, Viewer, ScopeUniversity, nil)
}

func CanAdminCollege(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Admin, ScopeCollege, &id)
}

func CanOperateCollege(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Operator, ScopeCollege, &id)
}

func CanViewCollege(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Viewer, ScopeCollege, &id)
}

func CanAdminDepartment(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Admin, ScopeDepartment, &id)
}

func CanOperateDepartment(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Operator, ScopeDepartment, &id)
}

func CanViewDepartment(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Viewer, ScopeDepartment, &id)
}

func CanAdminProgram(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Admin, ScopeProgram, &id)
}

func CanOperateProgram(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Operator, ScopeProgram, &id)
}

func CanViewProgram(c *gin.Context, id uuid.UUID) bool {
	return checkFromContext(c, Viewer, ScopeProgram, &id)
}

func CanAdmin(c *gin.Context, scope string, scopeID *uuid.UUID) bool {
	return checkFromContext(c, Admin, scope, scopeID)
}

func CanOperate(c *gin.Context, scope string, scopeID *uuid.UUID) bool {
	return checkFromContext(c, Operator, scope, scopeID)
}

func CanView(c *gin.Context, scope string, scopeID *uuid.UUID) bool {
	return checkFromContext(c, Viewer, scope, scopeID)
}

func CanManageRole(actorPermission, targetPermission string) bool {
	return permissionRank[actorPermission] >= permissionRank[targetPermission]
}

func CanManageScope(actorScopeType, targetScopeType string) bool {
	actorRank := scopeRank[actorScopeType]
	targetRank := scopeRank[targetScopeType]
	return actorRank >= targetRank
}

// Course role checking

type CourseRoleChecker interface {
	GetTeacherRole(ctx context.Context, offeringID, userID uuid.UUID) (string, error)
	IsEnrolled(ctx context.Context, offeringID, userID uuid.UUID) (bool, error)
}

var courseChecker CourseRoleChecker

func SetCourseChecker(cc CourseRoleChecker) {
	courseChecker = cc
}

func getTeachingRole(c *gin.Context, offeringID uuid.UUID) string {
	if courseChecker == nil {
		return ""
	}
	userID := middleware.GetUserID(c)
	role, err := courseChecker.GetTeacherRole(c.Request.Context(), offeringID, userID)
	if err != nil {
		return ""
	}
	return role
}

func IsOfferingTeacher(c *gin.Context, offeringID uuid.UUID) bool {
	return getTeachingRole(c, offeringID) == RoleTeacher
}

func IsOfferingAssistant(c *gin.Context, offeringID uuid.UUID) bool {
	return getTeachingRole(c, offeringID) == RoleAssistant
}

func IsOfferingStaff(c *gin.Context, offeringID uuid.UUID) bool {
	role := getTeachingRole(c, offeringID)
	return role == RoleTeacher || role == RoleAssistant
}

func IsOfferingStudent(c *gin.Context, offeringID uuid.UUID) bool {
	if courseChecker == nil {
		return false
	}
	userID := middleware.GetUserID(c)
	enrolled, err := courseChecker.IsEnrolled(c.Request.Context(), offeringID, userID)
	return err == nil && enrolled
}

func IsOfferingMember(c *gin.Context, offeringID uuid.UUID) bool {
	return IsOfferingStaff(c, offeringID) || IsOfferingStudent(c, offeringID)
}
