package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/communication"
	"github.com/randdotdev/e-campus-server/internal/identity"
	identityhttp "github.com/randdotdev/e-campus-server/internal/identity/http"
	identitypg "github.com/randdotdev/e-campus-server/internal/identity/postgres"
	identityredis "github.com/randdotdev/e-campus-server/internal/identity/redis"
	"github.com/randdotdev/e-campus-server/internal/management"
	managementpg "github.com/randdotdev/e-campus-server/internal/management/postgres"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// identitySet is what the identity context exports: auth runs the router's
// authentication middleware.
type identitySet struct {
	handler *identityhttp.Handler
	auth    *identity.AuthService
}

// wireIdentity builds the identity context: auth, users, preferences.
func wireIdentity(infra *infra, authzService *authz.Service,
	notification *communication.NotificationService, mgmt managementSet) identitySet {
	offeringReader := offeringRoleAdapter{mgmt.courseRepo}
	loginLimiter := middleware.AuthRateLimiter(middleware.AuthRateLimiterConfig{
		Enabled:       infra.cfg.AuthRate.Enabled,
		MaxAttempts:   infra.cfg.AuthRate.MaxAttempts,
		WindowSeconds: infra.cfg.AuthRate.WindowSeconds,
	})

	tokensRepo := identityredis.NewTokenRepository(infra.rdb)
	userRepo := identitypg.NewRepository(infra.db)
	prefsRepo := identitypg.NewPreferencesRepository(infra.db)

	roleManager := roleManagerAdapter{authzService}
	studentReader := studentReaderAdapter{mgmt.studentRepo}
	structureReader := structureReaderAdapter{mgmt.structure}

	auth := identity.NewAuthService(tokensRepo, userRepo, &infra.cfg.JWT, infra.slog)
	userService := identity.NewUserService(
		userRepo, tokensRepo, notification,
		roleManager, studentReader, structureReader, offeringReader, infra.slog,
	)
	prefsService := identity.NewPreferencesService(prefsRepo)

	return identitySet{
		handler: identityhttp.NewHandler(auth, userService, prefsService, infra.log, infra.cfg.IsProduction(), loginLimiter),
		auth:    auth,
	}
}

// offeringRoleAdapter answers identity.OfferingRoleReader from management's
// offering-membership read. With no separate teacher record in the schema, a
// user who holds any teaching seat is their own teacher id.
type offeringRoleAdapter struct {
	repo *managementpg.CourseRepository
}

func (a offeringRoleAdapter) OfferingRolesForUser(ctx context.Context, userID uuid.UUID) (*identity.OfferingMemberships, error) {
	rows, err := a.repo.OfferingMemberships(ctx, userID)
	if err != nil {
		return nil, err
	}
	m := &identity.OfferingMemberships{Roles: make([]identity.OfferingRole, len(rows))}
	for i, r := range rows {
		m.Roles[i] = identity.OfferingRole{
			OfferingID:      r.OfferingID,
			CourseNameEN:    r.CourseNameEN,
			CourseNameLocal: r.CourseNameLocal,
			Role:            r.Role,
		}
		if r.Role != "student" {
			m.TeacherID = &userID
		}
	}
	return m, nil
}

// studentReaderAdapter answers identity.StudentReader from management's
// student repository. Not-a-student is (nil, nil) — identity's port contract.
type studentReaderAdapter struct {
	repo *managementpg.StudentRepository
}

func (a studentReaderAdapter) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*identity.StudentInfo, error) {
	rec, err := a.repo.GetStudent(ctx, userID)
	if errors.Is(err, management.ErrStudentNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &identity.StudentInfo{ProgramID: rec.ProgramID}, nil
}

// structureReaderAdapter answers identity.UniversityReader — the slim
// structure projections /me/context is built from — from management's
// structure repository.
type structureReaderAdapter struct {
	repo *managementpg.Repository
}

func (a structureReaderAdapter) GetProgram(ctx context.Context, id uuid.UUID) (*identity.ProgramInfo, error) {
	p, err := a.repo.GetProgram(ctx, id)
	if err != nil || p == nil {
		return nil, err
	}
	return &identity.ProgramInfo{ID: p.ID, NameEN: p.NameEN, NameLocal: p.NameLocal, DepartmentID: p.DepartmentID}, nil
}

func (a structureReaderAdapter) GetDepartment(ctx context.Context, id uuid.UUID) (*identity.DepartmentInfo, error) {
	d, err := a.repo.GetDepartment(ctx, id)
	if err != nil || d == nil {
		return nil, err
	}
	return &identity.DepartmentInfo{ID: d.ID, NameEN: d.NameEN, NameLocal: d.NameLocal, CollegeID: d.CollegeID}, nil
}

func (a structureReaderAdapter) GetCollege(ctx context.Context, id uuid.UUID) (*identity.CollegeInfo, error) {
	c, err := a.repo.GetCollege(ctx, id)
	if err != nil || c == nil {
		return nil, err
	}
	return &identity.CollegeInfo{ID: c.ID, NameEN: c.NameEN, NameLocal: c.NameLocal}, nil
}

func (a structureReaderAdapter) ListActiveColleges(ctx context.Context) ([]identity.CollegeInfo, error) {
	active := true
	colleges, _, err := a.repo.ListColleges(ctx, pagination.PageParams{Limit: 100}, management.CollegeFilter{IsActive: &active})
	if err != nil {
		return nil, err
	}
	out := make([]identity.CollegeInfo, len(colleges))
	for i, c := range colleges {
		out[i] = identity.CollegeInfo{ID: c.ID, NameEN: c.NameEN, NameLocal: c.NameLocal}
	}
	return out, nil
}

// roleManagerAdapter answers identity's role questions with the authz engine.
type roleManagerAdapter struct{ svc *authz.Service }

func (a roleManagerAdapter) CanManageRole(ctx context.Context, actor, target *identity.RoleClaim) bool {
	return a.svc.CanManageRole(ctx, toAuthzClaim(actor), toAuthzClaim(target))
}

func (a roleManagerAdapter) CanGrantRole(actor, target *identity.RoleClaim) bool {
	if actor == nil || target == nil {
		return false
	}
	return authz.CanGrantRole(
		authz.Level(actor.Level), authz.Scope(actor.ScopeType),
		authz.Level(target.Level), authz.Scope(target.ScopeType),
	)
}

func toAuthzClaim(c *identity.RoleClaim) *authz.RoleClaim {
	if c == nil {
		return nil
	}
	return &authz.RoleClaim{
		Level:   authz.Level(c.Level),
		Scope:   authz.Scope(c.ScopeType),
		ScopeID: c.ScopeID,
		Domain:  authz.Domain(c.Domain),
	}
}
