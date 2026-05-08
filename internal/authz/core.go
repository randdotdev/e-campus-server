package authz

import "github.com/google/uuid"

func evaluate(identity *ResolvedIdentity, enriched *EnrichedResource, policies []Policy) bool {
	if identity == nil {
		return false
	}
	for _, policy := range policies {
		if policyMatches(identity, enriched, policy) {
			return true
		}
	}
	return false
}

func policyMatches(identity *ResolvedIdentity, enriched *EnrichedResource, p Policy) bool {
	if policyRequiresCourseRole(p) {
		if !actorHasCourseRole(identity, enriched, p) {
			return false
		}
	}
	if policyRequiresScope(p) {
		if identity.InstitutionRole == nil {
			return false
		}
		r := identity.InstitutionRole
		if permissionRank[r.Level] < permissionRank[*p.MinLevel] {
			return false
		}
		if !scopeCovers(r.ScopeType, r.ScopeID, *p.ScopeType, enriched) {
			return false
		}
	}
	if policyRequiresDomain(p) {
		if !actorHasDomain(identity, p) {
			return false
		}
	}
	return true
}

func policyRequiresCourseRole(p Policy) bool {
	return p.CourseRole != nil
}

func actorHasCourseRole(identity *ResolvedIdentity, enriched *EnrichedResource, p Policy) bool {
	if enriched == nil {
		return false
	}
	role, enrolled := identity.CourseRoles[enriched.ID]
	return enrolled && role == *p.CourseRole
}

func policyRequiresScope(p Policy) bool {
	return p.ScopeType != nil && p.MinLevel != nil
}

func scopeCovers(actorScopeType string, actorScopeID *uuid.UUID, requiredScopeType string, enriched *EnrichedResource) bool {
	actorRank := scopeRank[actorScopeType]
	requiredRank := scopeRank[requiredScopeType]

	if actorRank > requiredRank {
		return true
	}
	if actorRank < requiredRank {
		return false
	}

	if actorScopeID == nil {
		return requiredScopeType == ScopeUniversity || requiredScopeType == ScopePlatform
	}
	if enriched == nil {
		return true
	}
	resourceScopeID := resourceScopeFor(enriched, requiredScopeType)
	return resourceScopeID != nil && *actorScopeID == *resourceScopeID
}

func policyRequiresDomain(p Policy) bool {
	return p.Domain != nil
}

func actorHasDomain(identity *ResolvedIdentity, p Policy) bool {
	return identity.InstitutionRole != nil && identity.InstitutionRole.Domain == *p.Domain
}

func resourceScopeFor(enriched *EnrichedResource, scopeType string) *uuid.UUID {
	if enriched == nil {
		return nil
	}
	switch scopeType {
	case ScopeDepartment:
		if enriched.DepartmentID != nil {
			return enriched.DepartmentID
		}
		if enriched.Type == ScopeDepartment {
			return &enriched.ID
		}
	case ScopeCollege:
		if enriched.CollegeID != nil {
			return enriched.CollegeID
		}
		if enriched.Type == ScopeCollege {
			return &enriched.ID
		}
	case ScopeProgram:
		if enriched.ProgramID != nil {
			return enriched.ProgramID
		}
		if enriched.Type == ScopeProgram {
			return &enriched.ID
		}
	case ScopeUniversity, ScopePlatform:
		return &enriched.ID
	}
	return nil
}

func needsEnrichment(policies []Policy) bool {
	for _, p := range policies {
		if policyNeedsResourceLookup(p) {
			return true
		}
	}
	return false
}

func policyNeedsResourceLookup(p Policy) bool {
	if p.ScopeType == nil {
		return false
	}
	return scopeRequiresResourceID(*p.ScopeType)
}

func scopeRequiresResourceID(scopeType string) bool {
	switch scopeType {
	case ScopeDepartment, ScopeCollege, ScopeProgram:
		return true
	}
	return false
}

func needsCourseRoleCheck(policies []Policy) bool {
	for _, p := range policies {
		if policyRequiresCourseRole(p) {
			return true
		}
	}
	return false
}

func ValidatePolicy(p Policy) error {
	if p.CourseRole != nil && (p.ScopeType != nil || p.MinLevel != nil) {
		return ErrInvalidPolicy
	}
	if (p.ScopeType != nil) != (p.MinLevel != nil) {
		return ErrInvalidPolicy
	}
	return nil
}

func CanGrantRole(actorLevel, actorScopeType, targetLevel, targetScopeType string) bool {
	if actorLevel == "" || targetLevel == "" || actorScopeType == "" || targetScopeType == "" {
		return false
	}
	if permissionRank[actorLevel] < permissionRank[targetLevel] {
		return false
	}
	return scopeRank[actorScopeType] > scopeRank[targetScopeType]
}

func CanManageScope(actorScope, targetScope string) bool {
	return scopeRank[actorScope] > scopeRank[targetScope]
}
