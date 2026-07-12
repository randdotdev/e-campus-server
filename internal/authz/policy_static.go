package authz

import (
	"context"
	"sort"

	"github.com/google/uuid"
)

// StaticPolicyStore serves policies straight from the compiled-in
// defaults — no database, no cache, no seeding; a policy change is a
// deploy. Picking between it and the DB-backed store is one wiring line
// in main.go (see doc.go). It lives in the domain because it has zero
// infrastructure: the defaults map is its whole storage.
type StaticPolicyStore struct{}

// PolicyFor returns the compiled-in policy for the pair; a pair with no
// entry is an empty Policy, which denies.
func (StaticPolicyStore) PolicyFor(_ context.Context, key PolicyKey) (Policy, error) {
	return defaultPolicies[key], nil
}

// ListPermissions renders the defaults as rows so inspection endpoints keep
// working in code-only mode. IDs are zero: these rows are not addressable.
func (StaticPolicyStore) ListPermissions(context.Context) ([]Permission, error) {
	var out []Permission
	for key, policy := range defaultPolicies {
		for _, p := range policy.Staff {
			row := Permission{Resource: key.Resource, Action: key.Action, Type: TypeStaff, Active: true}
			minLevel, scope := p.MinLevel, p.Scope
			row.MinLevel, row.Scope = &minLevel, &scope
			if p.Domain != "" {
				domain := p.Domain
				row.Domain = &domain
			}
			out = append(out, row)
		}
		for _, seat := range policy.Offering {
			role := seat
			out = append(out, Permission{
				Resource: key.Resource, Action: key.Action,
				Type: TypeOffering, OfferingRole: &role, Active: true,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Resource != out[j].Resource {
			return out[i].Resource < out[j].Resource
		}
		if out[i].Action != out[j].Action {
			return out[i].Action < out[j].Action
		}
		return out[i].Type < out[j].Type
	})
	return out, nil
}

// CreatePermission refuses: in code-only mode a policy change is a deploy.
func (StaticPolicyStore) CreatePermission(context.Context, PermissionInput) (*Permission, error) {
	return nil, ErrPoliciesReadOnly
}

// DeactivatePermission refuses: in code-only mode a policy change is a deploy.
func (StaticPolicyStore) DeactivatePermission(context.Context, uuid.UUID) error {
	return ErrPoliciesReadOnly
}

// Seed is a no-op: the defaults are the store.
func (StaticPolicyStore) Seed(context.Context) error { return nil }

// Reset is a no-op: there are no edits to discard.
func (StaticPolicyStore) Reset(context.Context) error { return nil }

var _ PolicyStore = StaticPolicyStore{}
