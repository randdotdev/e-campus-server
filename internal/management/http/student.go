package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateStudentRequest binds a direct student admission.
type CreateStudentRequest struct {
	UserID        uuid.UUID `json:"user_id" binding:"required"`
	ProgramID     uuid.UUID `json:"program_id" binding:"required"`
	AdmissionYear int       `json:"admission_year" binding:"required,gte=2000,lte=2100"`
	Shift         string    `json:"shift" binding:"required,oneof=day evening"`
	Tuition       string    `json:"tuition" binding:"required,oneof=free paid"`
}

// UpdateStudentRequest binds a student patch; absent fields stay unchanged.
type UpdateStudentRequest struct {
	CurrentYear       *int    `json:"current_year" binding:"omitempty,gte=1,lte=8"`
	CurrentCohortYear *int    `json:"current_cohort_year" binding:"omitempty,gte=2000,lte=2100"`
	Shift             *string `json:"shift" binding:"omitempty,oneof=day evening"`
	Tuition           *string `json:"tuition" binding:"omitempty,oneof=free paid"`
}

// UpdateStudentStatusRequest binds a student status change.
type UpdateStudentStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active graduated withdrawn suspended on_leave"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// StudentResponse is the student's JSON shape with display columns. ID is
// the account id (users.id) — a student record has no id of its own.
type StudentResponse struct {
	ID                uuid.UUID                `json:"id"`
	ProgramID         uuid.UUID                `json:"program_id"`
	AdmissionYear     int                      `json:"admission_year"`
	CurrentCohortYear int                      `json:"current_cohort_year"`
	CurrentYear       int                      `json:"current_year"`
	Shift             management.Shift         `json:"shift"`
	Tuition           management.Tuition       `json:"tuition"`
	Status            management.StudentStatus `json:"status"`
	NameEN            string                   `json:"name_en"`
	NameLocal         *string                  `json:"name_local,omitempty"`
	EnrolledAt        string                   `json:"enrolled_at"`
	CreatedAt         string                   `json:"created_at"`
}

// CohortYearResponse is one cohort head-count row.
type CohortYearResponse struct {
	CohortYear   int `json:"cohort_year"`
	StudentCount int `json:"student_count"`
}

// CohortHistoryResponse is one cohort move row.
type CohortHistoryResponse struct {
	ID             uuid.UUID                     `json:"id"`
	StudentID      uuid.UUID                     `json:"student_id"`
	FromCohortYear int                           `json:"from_cohort_year"`
	ToCohortYear   int                           `json:"to_cohort_year"`
	FromYear       int                           `json:"from_year"`
	ToYear         int                           `json:"to_year"`
	Reason         management.CohortChangeReason `json:"reason"`
	Notes          *string                       `json:"notes,omitempty"`
	ChangedAt      string                        `json:"changed_at"`
}

func toStudentResponse(s *management.StudentSummary) *StudentResponse {
	if s == nil {
		return nil
	}
	return &StudentResponse{
		ID:                s.UserID,
		ProgramID:         s.ProgramID,
		AdmissionYear:     s.AdmissionYear,
		CurrentCohortYear: s.CurrentCohortYear,
		CurrentYear:       s.CurrentYear,
		Shift:             s.Shift,
		Tuition:           s.Tuition,
		Status:            s.Status,
		NameEN:            s.NameEN,
		NameLocal:         s.NameLocal,
		EnrolledAt:        s.EnrolledAt.Format(time.RFC3339),
		CreatedAt:         s.CreatedAt.Format(time.RFC3339),
	}
}

func toStudentsResponse(students []management.StudentSummary) []StudentResponse {
	result := make([]StudentResponse, len(students))
	for i := range students {
		result[i] = *toStudentResponse(&students[i])
	}
	return result
}

func toCohortYearsResponse(summaries []management.CohortYearSummary) []CohortYearResponse {
	result := make([]CohortYearResponse, len(summaries))
	for i, s := range summaries {
		result[i] = CohortYearResponse(s)
	}
	return result
}

func toCohortHistoriesResponse(histories []management.CohortHistory) []CohortHistoryResponse {
	result := make([]CohortHistoryResponse, len(histories))
	for i, hst := range histories {
		result[i] = CohortHistoryResponse{
			ID:             hst.ID,
			StudentID:      hst.StudentID,
			FromCohortYear: hst.FromCohortYear,
			ToCohortYear:   hst.ToCohortYear,
			FromYear:       hst.FromYear,
			ToYear:         hst.ToYear,
			Reason:         hst.Reason,
			Notes:          hst.Notes,
			ChangedAt:      hst.ChangedAt.Format(time.RFC3339),
		}
	}
	return result
}

func parseStudentFilter(c *gin.Context) (management.StudentFilter, bool) {
	var filter management.StudentFilter
	if id := c.Query("program_id"); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			response.BadRequest(c, "invalid program_id")
			return filter, false
		}
		filter.ProgramID = &parsed
	}
	if year := c.Query("cohort_year"); year != "" {
		val, err := strconv.Atoi(year)
		if err != nil {
			response.BadRequest(c, "invalid cohort_year")
			return filter, false
		}
		filter.CohortYear = &val
	}
	if stage := c.Query("stage"); stage != "" {
		val, err := strconv.Atoi(stage)
		if err != nil {
			response.BadRequest(c, "invalid stage")
			return filter, false
		}
		filter.Stage = &val
	}
	if s := c.Query("status"); s != "" {
		status := management.StudentStatus(s)
		if !management.ValidStudentStatus(status) {
			response.BadRequest(c, "invalid status")
			return filter, false
		}
		filter.Status = &status
	}
	if s := c.Query("shift"); s != "" {
		shift := management.Shift(s)
		if !management.ValidShift(shift) {
			response.BadRequest(c, "invalid shift")
			return filter, false
		}
		filter.Shift = &shift
	}
	if id := c.Query("cohort_group_id"); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			response.BadRequest(c, "invalid cohort_group_id")
			return filter, false
		}
		filter.CohortGroupID = &parsed
	}
	return filter, true
}

// ── Student handlers ──────────────────────────────────────────────────────────

// ListStudents handles GET /students.
func (h *Handler) ListStudents(c *gin.Context) {

	params := pagination.ParsePageParams(c)
	filter, ok := parseStudentFilter(c)
	if !ok {
		return
	}
	if params.Query != "" {
		filter.Query = &params.Query
	}
	filter.Scope = scopeFrom(c)

	students, hasMore, err := h.students.ListStudents(c.Request.Context(), params, filter)
	if err != nil {
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[StudentResponse]{Data: toStudentsResponse(students), HasMore: hasMore}
	if hasMore && len(students) > 0 {
		last := students[len(students)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.UserID)
	}
	response.OK(c, result)
}

// ListCohortYears handles GET /programs/:id/cohorts.
func (h *Handler) ListCohortYears(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program id")
		return
	}

	summaries, err := h.students.ListCohortYears(c.Request.Context(), programID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCohortYearsResponse(summaries))
}

// CreateStudent handles POST /students.
func (h *Handler) CreateStudent(c *gin.Context) {

	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	// The gate passed the rank; narrow to the body's parent unit (§18a, §21).
	if !h.gates.CheckStaffOn(c, authz.ResourceStudent, authz.ActionCreate, authz.ResourceProgram, req.ProgramID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	student, err := h.students.CreateStudent(c.Request.Context(),
		req.UserID, req.ProgramID, req.AdmissionYear,
		management.Shift(req.Shift), management.Tuition(req.Tuition))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toStudentResponse(student))
}

// GetStudent handles GET /students/:id.
func (h *Handler) GetStudent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	student, err := h.students.GetStudent(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toStudentResponse(student))
}

// GetMyStudentRecord handles GET /me/student.
func (h *Handler) GetMyStudentRecord(c *gin.Context) {
	student, err := h.students.GetStudent(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toStudentResponse(student))
}

// UpdateStudent handles PUT /students/:id.
func (h *Handler) UpdateStudent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	upd := management.StudentUpdate{
		CurrentYear:       req.CurrentYear,
		CurrentCohortYear: req.CurrentCohortYear,
	}
	if req.Shift != nil {
		shift := management.Shift(*req.Shift)
		upd.Shift = &shift
	}
	if req.Tuition != nil {
		tuition := management.Tuition(*req.Tuition)
		upd.Tuition = &tuition
	}

	student, err := h.students.UpdateStudent(c.Request.Context(), id, upd)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toStudentResponse(student))
}

// UpdateStudentStatus handles PUT /students/:id/status.
func (h *Handler) UpdateStudentStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateStudentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	student, err := h.students.UpdateStudentStatus(c.Request.Context(), id, management.StudentStatus(req.Status))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toStudentResponse(student))
}

// ListCohortHistory handles GET /students/:id/history.
func (h *Handler) ListCohortHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	history, err := h.students.ListCohortHistory(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCohortHistoriesResponse(history))
}
