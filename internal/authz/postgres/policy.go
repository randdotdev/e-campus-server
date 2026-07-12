package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/authz"
)

// seedLockKey is the advisory-lock key serialising Seed and Reset across
// concurrently booting instances. Arbitrary but stable.
const seedLockKey = 0x00417574687A5064 // "AuthzPd"

// PolicyStore is the PostgreSQL implementation of authz.PolicyStore.
type PolicyStore struct {
	db *sqlx.DB
}

// NewPolicyStore returns the policy store backed by db.
func NewPolicyStore(db *sqlx.DB) *PolicyStore {
	return &PolicyStore{db: db}
}

// PolicyFor returns the active permissions for one (resource, action) pair,
// assembled into a Policy. No rows is an empty Policy — deny by default.
func (r *PolicyStore) PolicyFor(ctx context.Context, key authz.PolicyKey) (authz.Policy, error) {
	var rows []authz.Permission
	const q = `
		SELECT id, resource, verb, type, scope_type, min_level, course_role, domain, is_active
		FROM authz_policies
		WHERE resource = $1 AND verb = $2 AND is_active = true
	`
	if err := r.db.SelectContext(ctx, &rows, q, key.Resource, key.Action); err != nil {
		return authz.Policy{}, fmt.Errorf("authz: load policy %s/%s: %w", key.Resource, key.Action, err)
	}
	return assemblePolicy(rows), nil
}

// ListPermissions returns every stored permission, active and inactive.
func (r *PolicyStore) ListPermissions(ctx context.Context) ([]authz.Permission, error) {
	var rows []authz.Permission
	const q = `
		SELECT id, resource, verb, type, scope_type, min_level, course_role, domain, is_active
		FROM authz_policies
		ORDER BY resource, verb, type
	`
	if err := r.db.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("authz: list permissions: %w", err)
	}
	return rows, nil
}

// CreatePermission inserts one permission row. The unique index on the
// permission tuple decides duplicates.
func (r *PolicyStore) CreatePermission(ctx context.Context, in authz.PermissionInput) (*authz.Permission, error) {
	var created authz.Permission
	const q = `
		INSERT INTO authz_policies (resource, verb, type, scope_type, min_level, course_role, domain)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''))
		RETURNING id, resource, verb, type, scope_type, min_level, course_role, domain, is_active
	`
	err := r.db.GetContext(ctx, &created, q,
		in.Resource, in.Action, in.Type, in.Scope, in.MinLevel, in.OfferingRole, in.Domain)
	if isUniqueViolation(err) {
		return nil, authz.ErrPermissionExists
	}
	if err != nil {
		return nil, fmt.Errorf("authz: create permission: %w", err)
	}
	return &created, nil
}

// DeactivatePermission soft-deletes one permission row (guarded UPDATE).
// Administration never hard-deletes: a pair whose rows are all inactive
// means "nobody may do this", and Seed respects that across boots.
func (r *PolicyStore) DeactivatePermission(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE authz_policies SET is_active = false, updated_at = NOW() WHERE id = $1 AND is_active = true`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("authz: deactivate permission: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return authz.ErrPermissionNotFound
	}
	return nil
}

// Seed installs the compiled-in defaults for every pair that has no stored
// rows at all (active or inactive — admin edits survive boots). One
// transaction under an advisory lock, so concurrently booting instances
// serialise instead of double-inserting.
func (r *PolicyStore) Seed(ctx context.Context) error {
	return r.inTxWithLock(ctx, func(tx *sqlx.Tx) error {
		known, err := knownPairs(ctx, tx)
		if err != nil {
			return err
		}
		for key, policy := range authz.DefaultPolicies() {
			if known[key] {
				continue
			}
			if err := insertPolicy(ctx, tx, key, policy); err != nil {
				return err
			}
		}
		return nil
	})
}

// Reset discards every stored permission and reinstalls the full defaults —
// the one sanctioned hard delete, and the recovery path from bad edits.
func (r *PolicyStore) Reset(ctx context.Context) error {
	return r.inTxWithLock(ctx, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM authz_policies`); err != nil {
			return fmt.Errorf("authz: reset: clear policies: %w", err)
		}
		for key, policy := range authz.DefaultPolicies() {
			if err := insertPolicy(ctx, tx, key, policy); err != nil {
				return err
			}
		}
		return nil
	})
}

// inTxWithLock runs fn inside one transaction holding the seed advisory
// lock; the lock releases with the transaction.
func (r *PolicyStore) inTxWithLock(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("authz: begin policy tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, seedLockKey); err != nil {
		return fmt.Errorf("authz: acquire seed lock: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// knownPairs returns every (resource, action) pair the table has rows for —
// active or not — in one query, so seeding costs one round-trip plus the
// missing inserts instead of one probe per default pair.
func knownPairs(ctx context.Context, tx *sqlx.Tx) (map[authz.PolicyKey]bool, error) {
	var rows []struct {
		Resource authz.Entity `db:"resource"`
		Action   authz.Action `db:"verb"`
	}
	const q = `SELECT DISTINCT resource, verb FROM authz_policies`
	if err := tx.SelectContext(ctx, &rows, q); err != nil {
		return nil, fmt.Errorf("authz: list known pairs: %w", err)
	}
	known := make(map[authz.PolicyKey]bool, len(rows))
	for _, row := range rows {
		known[authz.PolicyKey{Resource: row.Resource, Action: row.Action}] = true
	}
	return known, nil
}

// insertPolicy writes every permission of one policy as rows of the pair.
func insertPolicy(ctx context.Context, tx *sqlx.Tx, key authz.PolicyKey, policy authz.Policy) error {
	const staffQ = `
		INSERT INTO authz_policies (resource, verb, type, scope_type, min_level, domain)
		VALUES ($1, $2, 'staff', $3, $4, NULLIF($5, ''))
	`
	for _, p := range policy.Staff {
		if _, err := tx.ExecContext(ctx, staffQ, key.Resource, key.Action, p.Scope, p.MinLevel, p.Domain); err != nil {
			return fmt.Errorf("authz: seed %s/%s staff: %w", key.Resource, key.Action, err)
		}
	}
	const courseQ = `
		INSERT INTO authz_policies (resource, verb, type, course_role)
		VALUES ($1, $2, 'offering', $3)
	`
	for _, role := range policy.Offering {
		if _, err := tx.ExecContext(ctx, courseQ, key.Resource, key.Action, role); err != nil {
			return fmt.Errorf("authz: seed %s/%s course: %w", key.Resource, key.Action, err)
		}
	}
	return nil
}

// assemblePolicy folds permission rows into the evaluator's Policy shape.
// Malformed rows (wrong type/field combination — prevented by the schema
// CHECKs) are skipped rather than misread.
func assemblePolicy(rows []authz.Permission) authz.Policy {
	var p authz.Policy
	for _, row := range rows {
		switch row.Type {
		case authz.TypeStaff:
			if row.MinLevel == nil || row.Scope == nil {
				continue
			}
			perm := authz.StaffPermission{MinLevel: *row.MinLevel, Scope: *row.Scope}
			if row.Domain != nil {
				perm.Domain = *row.Domain
			}
			p.Staff = append(p.Staff, perm)
		case authz.TypeOffering:
			if row.OfferingRole == nil {
				continue
			}
			p.Offering = append(p.Offering, *row.OfferingRole)
		default:
			// Owner-type rows never persist (schema CHECK); skip if seen.
		}
	}
	return p
}

var _ authz.PolicyStore = (*PolicyStore)(nil)
