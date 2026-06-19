package authz

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/randdotdev/e-campus-server/internal/auth"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

type PolicyRepository interface {
	GetPolicies(ctx context.Context, resource, verb string) ([]Policy, error)
	EnrichResource(ctx context.Context, resourceType string, id uuid.UUID) (EnrichedResource, error)
	CreatePolicy(ctx context.Context, p Policy) (Policy, error)
	GetPolicy(ctx context.Context, id uuid.UUID) (Policy, error)
	UpdatePolicy(ctx context.Context, id uuid.UUID, p Policy) error
	SoftDeletePolicy(ctx context.Context, id uuid.UUID) error
	ListPolicies(ctx context.Context) ([]Policy, error)
}

type CourseTeacherReader interface {
	GetTeacherOfferingsForUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]string, error)
}

type CourseEnrollmentReader interface {
	GetEnrolledOfferingsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type ApplicationEnricher interface {
	EnrichApplication(ctx context.Context, id uuid.UUID) (EnrichedResource, error)
}

type CurriculumEnricher interface {
	EnrichCurriculum(ctx context.Context, id uuid.UUID) (EnrichedResource, error)
}

type Service struct {
	repo           PolicyRepository
	courseTeachers CourseTeacherReader
	courseEnroll   CourseEnrollmentReader
	cache          *cache
}

var defaultService atomic.Pointer[Service]

func NewService(
	db *sqlx.DB,
	courseTeachers CourseTeacherReader,
	courseEnroll CourseEnrollmentReader,
	applicationEnricher ApplicationEnricher,
	curriculumEnricher CurriculumEnricher,
	rdb *redis.Client,
) *Service {
	return &Service{
		repo:           newSQLRepo(db, applicationEnricher, curriculumEnricher),
		courseTeachers: courseTeachers,
		courseEnroll:   courseEnroll,
		cache:          newCache(rdb),
	}
}

func SetDefault(s *Service) {
	if s == nil {
		panic("authz: nil Service")
	}
	defaultService.Store(s)
}

func ResetDefault() {
	defaultService.Store(nil)
}

func getDefault() *Service {
	return defaultService.Load()
}

func Check(c *gin.Context, res Entity, verb string, id ...uuid.UUID) bool {
	s := getDefault()
	if s == nil {
		return false
	}
	return s.Check(c, res, verb, id...)
}

func CheckList(c *gin.Context, res Entity) (bool, ScopeFilter) {
	s := getDefault()
	if s == nil {
		return false, ScopeFilter{}
	}
	return s.CheckList(c, res)
}

func CourseRole(c *gin.Context, offeringID uuid.UUID) string {
	s := getDefault()
	if s == nil {
		return ""
	}
	return s.CourseRole(c, offeringID)
}

func CanManageRole(ctx context.Context, actor, target *auth.RoleClaim) bool {
	s := getDefault()
	if s == nil {
		return false
	}
	return s.CanManageRole(ctx, actor, target)
}

func (s *Service) Check(c *gin.Context, res Entity, verb string, id ...uuid.UUID) bool {
	ctx := c.Request.Context()
	userID := actorID(c)
	role := actorInstitutionRole(c)

	policies, ok := s.policiesFor(ctx, res, verb)
	if !ok {
		return false
	}

	identity, ok := s.resolveIdentity(ctx, userID, role, policies)
	if !ok {
		return false
	}

	enriched := s.resolveResource(ctx, res, id, policies)

	return evaluate(identity, enriched, policies)
}

func (s *Service) CheckList(c *gin.Context, res Entity) (bool, ScopeFilter) {
	ctx := c.Request.Context()
	userID := actorID(c)
	role := actorInstitutionRole(c)

	policies, ok := s.policiesFor(ctx, res, ActionList)
	if !ok {
		return false, ScopeFilter{}
	}

	identity, ok := s.resolveIdentity(ctx, userID, role, policies)
	if !ok {
		return false, ScopeFilter{}
	}

	if !evaluate(identity, nil, policies) {
		return false, ScopeFilter{}
	}

	return true, scopeFilterFor(role)
}

func (s *Service) CourseRole(c *gin.Context, offeringID uuid.UUID) string {
	ctx := c.Request.Context()
	userID := actorID(c)
	roles, ok := s.resolveCourseRoles(ctx, userID, nil)
	if !ok {
		return ""
	}
	return roles[offeringID]
}

func (s *Service) GetEnrichedResource(ctx context.Context, res Entity, id uuid.UUID) (EnrichedResource, error) {
	return s.getResource(ctx, string(res), id)
}

func (s *Service) CanManageRole(ctx context.Context, actor, target *auth.RoleClaim) bool {
	if actor == nil || target == nil {
		return false
	}

	actorRank := scopeRank[actor.ScopeType]
	targetRank := scopeRank[target.ScopeType]

	if actorRank > targetRank {
		if permissionRank[actor.Level] < permissionRank[target.Level] {
			return false
		}
		if actor.ScopeType == ScopePlatform || actor.ScopeType == ScopeUniversity {
			return true
		}
		if target.ScopeID == nil {
			return true
		}
		parentID := s.resolveParent(ctx, actor.ScopeType, target.ScopeType, *target.ScopeID)
		return parentID != nil && actor.ScopeID != nil && *actor.ScopeID == *parentID
	}

	if actorRank == targetRank {
		if actor.ScopeID != nil && target.ScopeID != nil && *actor.ScopeID != *target.ScopeID {
			return false
		}
		return permissionRank[actor.Level] > permissionRank[target.Level]
	}

	return false
}

func (s *Service) CanGrantRole(actor, target *auth.RoleClaim) bool {
	if actor == nil || target == nil {
		return false
	}
	return CanGrantRole(actor.Level, actor.ScopeType, target.Level, target.ScopeType)
}

func (s *Service) CanManageScope(actorScope, targetScope string) bool {
	return CanManageScope(actorScope, targetScope)
}

func (s *Service) InvalidateCourseRoles(ctx context.Context, userID uuid.UUID) error {
	return s.cache.del(ctx, fmt.Sprintf("authz:courses:%s", userID))
}

func (s *Service) InvalidatePolicies(ctx context.Context) error {
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.cache.scanDel(bg, "authz:policy:*")
	}()
	return nil
}

func (s *Service) InvalidateResource(ctx context.Context, res Entity, id uuid.UUID) error {
	return s.cache.del(ctx, fmt.Sprintf("authz:resource:%s:%s", res, id))
}

func actorID(c *gin.Context) uuid.UUID {
	return middleware.GetUserID(c)
}

func actorInstitutionRole(c *gin.Context) *auth.RoleClaim {
	return middleware.GetUserRole(c)
}

func (s *Service) policiesFor(ctx context.Context, res Entity, verb string) ([]Policy, bool) {
	key := fmt.Sprintf("authz:policy:%s:%s", res, verb)
	var policies []Policy
	if s.cache.get(ctx, key, &policies) {
		return policies, true
	}

	policies, err := s.repo.GetPolicies(ctx, string(res), verb)
	if err != nil {
		return nil, false
	}
	if len(policies) == 0 {
		return nil, false
	}

	s.cache.set(ctx, key, policies, time.Hour)
	return policies, true
}

func (s *Service) resolveIdentity(ctx context.Context, userID uuid.UUID, role *auth.RoleClaim, policies []Policy) (*ResolvedIdentity, bool) {
	courseRoles, ok := s.resolveCourseRoles(ctx, userID, policies)
	if !ok {
		return nil, false
	}
	return &ResolvedIdentity{
		UserID:          userID,
		InstitutionRole: role,
		CourseRoles:     courseRoles,
	}, true
}

func (s *Service) resolveCourseRoles(ctx context.Context, userID uuid.UUID, policies []Policy) (map[uuid.UUID]string, bool) {
	if policies != nil && !needsCourseRoleCheck(policies) {
		return nil, true
	}

	key := fmt.Sprintf("authz:courses:%s", userID)
	var roles map[uuid.UUID]string
	if s.cache.get(ctx, key, &roles) {
		return roles, true
	}

	var teacherRoles map[uuid.UUID]string
	var enrolledIDs []uuid.UUID

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		teacherRoles, err = s.courseTeachers.GetTeacherOfferingsForUser(gctx, userID)
		return err
	})
	g.Go(func() error {
		var err error
		enrolledIDs, err = s.courseEnroll.GetEnrolledOfferingsForUser(gctx, userID)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, false
	}

	roles = make(map[uuid.UUID]string, len(teacherRoles)+len(enrolledIDs))
	for id, role := range teacherRoles {
		roles[id] = role
	}
	for _, id := range enrolledIDs {
		if _, exists := roles[id]; !exists {
			roles[id] = CourseRoleStudent
		}
	}

	s.cache.set(ctx, key, roles, 5*time.Minute)
	return roles, true
}

func (s *Service) resolveResource(ctx context.Context, res Entity, id []uuid.UUID, policies []Policy) *EnrichedResource {
	if len(id) == 0 {
		return nil
	}
	if needsEnrichment(policies) {
		return s.enrichedResource(ctx, res, id[0])
	}
	return &EnrichedResource{Type: string(res), ID: id[0]}
}

func (s *Service) enrichedResource(ctx context.Context, res Entity, id uuid.UUID) *EnrichedResource {
	key := fmt.Sprintf("authz:resource:%s:%s", res, id)
	var e EnrichedResource
	if s.cache.get(ctx, key, &e) {
		return &e
	}

	enriched, err := s.repo.EnrichResource(ctx, string(res), id)
	if err != nil {
		return nil
	}

	s.cache.set(ctx, key, enriched, 10*time.Minute)
	return &enriched
}

func (s *Service) getResource(ctx context.Context, resourceType string, id uuid.UUID) (EnrichedResource, error) {
	key := fmt.Sprintf("authz:resource:%s:%s", resourceType, id)
	var enriched EnrichedResource
	if s.cache.get(ctx, key, &enriched) {
		return enriched, nil
	}
	enriched, err := s.repo.EnrichResource(ctx, resourceType, id)
	if err != nil {
		return EnrichedResource{}, err
	}
	s.cache.set(ctx, key, enriched, 10*time.Minute)
	return enriched, nil
}

func scopeFilterFor(role *auth.RoleClaim) ScopeFilter {
	if role == nil {
		return ScopeFilter{}
	}
	var f ScopeFilter
	switch role.ScopeType {
	case ScopeProgram:
		f.ProgramID = role.ScopeID
	case ScopeDepartment:
		f.DepartmentID = role.ScopeID
	case ScopeCollege:
		f.CollegeID = role.ScopeID
	}
	return f
}

func (s *Service) resolveParent(ctx context.Context, actorScope, targetScope string, targetScopeID uuid.UUID) *uuid.UUID {
	var entity Entity
	switch targetScope {
	case ScopeDepartment:
		entity = ResourceDepartment
	case ScopeProgram:
		entity = ResourceProgram
	default:
		return nil
	}

	enriched, err := s.GetEnrichedResource(ctx, entity, targetScopeID)
	if err != nil {
		return nil
	}

	switch actorScope {
	case ScopeCollege:
		return enriched.CollegeID
	case ScopeDepartment:
		return enriched.DepartmentID
	}
	return nil
}
