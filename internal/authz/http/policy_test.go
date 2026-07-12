package http

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/identity"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
)

// stubStore satisfies authz.PolicyStore with empty answers — the guard
// under test never reaches a real store.
type stubStore struct{}

func (stubStore) PolicyFor(context.Context, authz.PolicyKey) (authz.Policy, error) {
	return authz.Policy{}, nil
}
func (stubStore) ListPermissions(context.Context) ([]authz.Permission, error) { return nil, nil }
func (stubStore) CreatePermission(context.Context, authz.PermissionInput) (*authz.Permission, error) {
	return nil, authz.ErrInvalidPermission
}
func (stubStore) DeactivatePermission(context.Context, uuid.UUID) error { return nil }
func (stubStore) Seed(context.Context) error                            { return nil }
func (stubStore) Reset(context.Context) error                           { return nil }

type stubReaders struct{}

func (stubReaders) LineageFor(context.Context, authz.Entity, uuid.UUID) (authz.Lineage, error) {
	return authz.Lineage{}, nil
}
func (stubReaders) RelationTo(context.Context, uuid.UUID, uuid.UUID) (authz.OfferingRole, error) {
	return authz.RelationNone, nil
}
func (stubReaders) PostFacts(context.Context, uuid.UUID) (authz.PostFacts, error) {
	return authz.PostFacts{}, nil
}

// TestPolicyRoutesRequireSuperAdmin proves the compiled-in guard on the
// policy administration surface: only a super admin passes, regardless of
// scope or any stored policy.
func TestPolicyRoutesRequireSuperAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		role       *identity.RoleClaim
		wantStatus int
	}{
		{"super_admin_passes", &identity.RoleClaim{Level: string(authz.LevelSuperAdmin), ScopeType: string(authz.ScopeUniversity)}, http.StatusOK},
		{"university_admin_forbidden", &identity.RoleClaim{Level: string(authz.LevelAdmin), ScopeType: string(authz.ScopeUniversity)}, http.StatusForbidden},
		{"roleless_forbidden", nil, http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			protected := router.Group("")
			protected.Use(func(c *gin.Context) {
				c.Set(middleware.UserIDKey, uuid.New())
				if tt.role != nil {
					c.Set(middleware.UserRoleKey, tt.role)
				}
			})
			h := NewHandler(authz.NewService(stubStore{}, stubReaders{}, slog.Default()))
			h.Routes(protected)

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/authz/permissions", nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
