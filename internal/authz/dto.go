package authz

import "github.com/google/uuid"

type CreatePolicyRequest struct {
	Resource   string  `json:"resource" binding:"required"`
	Verb       string  `json:"verb" binding:"required"`
	ScopeType  *string `json:"scope_type,omitempty"`
	MinLevel   *string `json:"min_level,omitempty"`
	CourseRole *string `json:"course_role,omitempty"`
	Domain     *string `json:"domain,omitempty"`
}

type UpdatePolicyRequest = CreatePolicyRequest

type PolicyResponse struct {
	ID         uuid.UUID `json:"id"`
	Resource   string    `json:"resource"`
	Verb       string    `json:"verb"`
	ScopeType  *string   `json:"scope_type,omitempty"`
	MinLevel   *string   `json:"min_level,omitempty"`
	CourseRole *string   `json:"course_role,omitempty"`
	Domain     *string   `json:"domain,omitempty"`
	IsActive   bool      `json:"is_active"`
}

type ListPoliciesResponse struct {
	Policies []PolicyResponse `json:"policies"`
}
