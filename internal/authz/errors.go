package authz

import "errors"

var (
	// ErrTargetNotFound maps to 404, not 403 — campus IDs are not secrets,
	// and a missing row is a friendlier answer than a denial.
	ErrTargetNotFound = errors.New("authz: target resource not found")

	ErrPermissionNotFound = errors.New("authz: permission not found")

	// ErrPermissionExists is decided by the unique index on the
	// permission tuple, not a pre-read.
	ErrPermissionExists = errors.New("authz: permission already exists")

	// ErrInvalidPermission covers a shape wrong for its type (a staff
	// permission missing level or scope, an offering permission naming
	// one) or any value outside the closed vocabulary.
	ErrInvalidPermission = errors.New("authz: invalid permission")

	// ErrPoliciesReadOnly is the code-only store refusing edits: in that
	// mode a policy change is a deploy.
	ErrPoliciesReadOnly = errors.New("authz: policies are read-only")
)
