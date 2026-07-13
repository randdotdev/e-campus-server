package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateCohortGroupRequest binds a cohort group creation.
type CreateCohortGroupRequest struct {
	ProgramID  uuid.UUID `json:"program_id" binding:"required"`
	CohortYear int       `json:"cohort_year" binding:"required,min=2000,max=2100"`
	Stage      int       `json:"stage" binding:"required,min=1,max=10"`
	Type       string    `json:"type" binding:"required,oneof=theory practice"`
	Name       string    `json:"name" binding:"required,min=1,max=10"`
}

// AssignToCohortGroupRequest binds a cohort group membership assignment;
// the group comes from the path.
type AssignToCohortGroupRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// CohortGroupResponse is the cohort group's JSON shape.
type CohortGroupResponse struct {
	ID          uuid.UUID                  `json:"id"`
	ProgramID   uuid.UUID                  `json:"program_id"`
	CohortYear  int                        `json:"cohort_year"`
	Stage       int                        `json:"stage"`
	Type        management.CohortGroupType `json:"type"`
	Name        string                     `json:"name"`
	MemberCount int                        `json:"member_count,omitempty"`
	CreatedAt   time.Time                  `json:"created_at"`
}

func toCohortGroupResponse(g *management.CohortGroup) CohortGroupResponse {
	return CohortGroupResponse{
		ID:         g.ID,
		ProgramID:  g.ProgramID,
		CohortYear: g.CohortYear,
		Stage:      g.Stage,
		Type:       g.Type,
		Name:       g.Name,
		CreatedAt:  g.CreatedAt,
	}
}

func toCohortGroupsWithCountResponse(groups []management.CohortGroupWithCount) []CohortGroupResponse {
	result := make([]CohortGroupResponse, len(groups))
	for i := range groups {
		g := &groups[i]
		resp := toCohortGroupResponse(&g.CohortGroup)
		resp.MemberCount = g.MemberCount
		result[i] = resp
	}
	return result
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListCohortGroups handles GET /programs/:id/cohort-groups.
func (h *Handler) ListCohortGroups(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program id")
		return
	}

	cohortYear, err := parseIntQuery(c, "cohort_year")
	if err != nil {
		response.BadRequest(c, "invalid cohort_year")
		return
	}
	stage, err := parseIntQuery(c, "stage")
	if err != nil {
		response.BadRequest(c, "invalid stage")
		return
	}

	groups, err := h.cohortGroups.ListCohortGroupsWithCounts(c.Request.Context(), programID, cohortYear, stage)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCohortGroupsWithCountResponse(groups))
}

// CreateCohortGroup handles POST /cohort-groups.
func (h *Handler) CreateCohortGroup(c *gin.Context) {
	var req CreateCohortGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	group, err := h.cohortGroups.CreateCohortGroup(c.Request.Context(), req.ProgramID, req.CohortYear, req.Stage, management.CohortGroupType(req.Type), req.Name)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toCohortGroupResponse(group))
}

// AssignToCohortGroup handles POST /cohort-groups/:id/members.
func (h *Handler) AssignToCohortGroup(c *gin.Context) {
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	var req AssignToCohortGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if err := h.cohortGroups.AssignToCohortGroup(c.Request.Context(), req.StudentID, groupID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// RemoveFromCohortGroup handles DELETE /cohort-groups/:id/members/:studentId.
func (h *Handler) RemoveFromCohortGroup(c *gin.Context) {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		response.BadRequest(c, "invalid student id")
		return
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	if err := h.cohortGroups.DeleteCohortGroupMember(c.Request.Context(), studentID, groupID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// parseIntQuery parses an optional integer query parameter; absence is zero.
func parseIntQuery(c *gin.Context, name string) (int, error) {
	val := c.Query(name)
	if val == "" {
		return 0, nil
	}
	return strconv.Atoi(val)
}
