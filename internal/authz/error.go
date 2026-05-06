package authz

import "errors"

var (
	ErrInvalidPolicy  = errors.New("authz: policy cannot combine course_role with scope constraints")
	ErrPolicyNotFound = errors.New("authz: policy not found")
)
