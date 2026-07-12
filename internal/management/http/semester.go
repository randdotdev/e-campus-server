package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateSemesterRequest binds a semester creation.
type CreateSemesterRequest struct {
	AcademicYearID    uuid.UUID  `json:"academic_year_id" binding:"required"`
	Semester          string     `json:"semester" binding:"required,oneof=fall spring summer annual"`
	StartDate         time.Time  `json:"start_date" binding:"required"`
	EndDate           time.Time  `json:"end_date" binding:"required,gtfield=StartDate"`
	RegistrationStart *time.Time `json:"registration_start"`
	RegistrationEnd   *time.Time `json:"registration_end"`
	GradeEntryStart   *time.Time `json:"grade_entry_start"`
	GradeEntryEnd     *time.Time `json:"grade_entry_end"`
	PassThreshold     int        `json:"pass_threshold" binding:"omitempty,gte=0,lte=100"`
}

// UpdateSemesterRequest binds a semester patch; absent fields stay unchanged.
type UpdateSemesterRequest struct {
	Semester          *string    `json:"semester" binding:"omitempty,oneof=fall spring summer annual"`
	StartDate         *time.Time `json:"start_date"`
	EndDate           *time.Time `json:"end_date"`
	RegistrationStart *time.Time `json:"registration_start"`
	RegistrationEnd   *time.Time `json:"registration_end"`
	GradeEntryStart   *time.Time `json:"grade_entry_start"`
	GradeEntryEnd     *time.Time `json:"grade_entry_end"`
	PassThreshold     *int       `json:"pass_threshold" binding:"omitempty,gte=0,lte=100"`
}

// UpdateSemesterStatusRequest binds a semester status transition.
type UpdateSemesterStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=upcoming active grading finalized archived"`
}

// GenerateOfferingsRequest binds the optional scope of an offering-generation
// run.
type GenerateOfferingsRequest struct {
	ProgramID  *uuid.UUID `json:"program_id"`
	CohortYear *int       `json:"cohort_year"`
	Shift      *string    `json:"shift" binding:"omitempty,oneof=day evening"`
}

// BulkEnrollRequest binds the optional scope of a bulk-enrollment run.
type BulkEnrollRequest struct {
	ProgramID  *uuid.UUID `json:"program_id"`
	CohortYear *int       `json:"cohort_year"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// SemesterResponse is the semester's JSON shape.
type SemesterResponse struct {
	ID                uuid.UUID                 `json:"id"`
	AcademicYearID    uuid.UUID                 `json:"academic_year_id"`
	Semester          management.SemesterType   `json:"semester"`
	StartDate         string                    `json:"start_date"`
	EndDate           string                    `json:"end_date"`
	RegistrationStart *string                   `json:"registration_start,omitempty"`
	RegistrationEnd   *string                   `json:"registration_end,omitempty"`
	GradeEntryStart   *string                   `json:"grade_entry_start,omitempty"`
	GradeEntryEnd     *string                   `json:"grade_entry_end,omitempty"`
	PassThreshold     int                       `json:"pass_threshold"`
	Status            management.SemesterStatus `json:"status"`
	CreatedAt         string                    `json:"created_at"`
}

// GenerateOfferingsResponse is the offering-generation run summary.
type GenerateOfferingsResponse struct {
	Created int                      `json:"created"`
	Skipped int                      `json:"skipped"`
	Details []OfferingRecordResponse `json:"details,omitempty"`
}

// OfferingRecordResponse is one offering considered during generation.
type OfferingRecordResponse struct {
	CourseID   uuid.UUID                    `json:"course_id"`
	CourseCode string                       `json:"course_code"`
	CohortYear int                          `json:"cohort_year"`
	Shift      management.Shift             `json:"shift"`
	Status     management.OfferingGenStatus `json:"status"`
}

// BulkEnrollResponse is the bulk-enrollment run summary.
type BulkEnrollResponse struct {
	Enrolled int                    `json:"enrolled"`
	Skipped  int                    `json:"skipped"`
	Blocked  int                    `json:"blocked"`
	Errors   int                    `json:"errors"`
	Details  *EnrollDetailsResponse `json:"details,omitempty"`
}

// EnrollDetailsResponse itemises a bulk-enrollment run.
type EnrollDetailsResponse struct {
	Enrolled []EnrollRecordResponse  `json:"enrolled,omitempty"`
	Skipped  []SkipRecordResponse    `json:"skipped,omitempty"`
	Blocked  []BlockedRecordResponse `json:"blocked,omitempty"`
}

// EnrollRecordResponse is one successful bulk enrollment.
type EnrollRecordResponse struct {
	StudentID   uuid.UUID                 `json:"student_id"`
	StudentName string                    `json:"student_name"`
	OfferingID  uuid.UUID                 `json:"offering_id"`
	CourseCode  string                    `json:"course_code"`
	Type        management.EnrollmentType `json:"type"`
}

// SkipRecordResponse is one bulk enrollment skipped with a reason.
type SkipRecordResponse struct {
	StudentID   uuid.UUID `json:"student_id"`
	StudentName string    `json:"student_name"`
	CourseID    uuid.UUID `json:"course_id"`
	CourseCode  string    `json:"course_code"`
	Reason      string    `json:"reason"`
}

// BlockedRecordResponse is one bulk enrollment blocked by a prerequisite.
type BlockedRecordResponse struct {
	StudentID           uuid.UUID `json:"student_id"`
	StudentName         string    `json:"student_name"`
	CourseCode          string    `json:"course_code"`
	MissingPrerequisite string    `json:"missing_prerequisite"`
	MissingCourseID     uuid.UUID `json:"missing_course_id"`
}

// EndSemesterResponse is the year-end progression summary.
type EndSemesterResponse struct {
	Processed int    `json:"processed"`
	Promoted  int    `json:"promoted"`
	Repeated  int    `json:"repeated"`
	Unchanged int    `json:"unchanged"`
	Errors    int    `json:"errors"`
	Warning   string `json:"warning,omitempty"`
}

func toSemesterResponse(s *management.Semester) *SemesterResponse {
	if s == nil {
		return nil
	}
	resp := &SemesterResponse{
		ID:             s.ID,
		AcademicYearID: s.AcademicYearID,
		Semester:       s.Semester,
		StartDate:      s.StartDate.Format("2006-01-02"),
		EndDate:        s.EndDate.Format("2006-01-02"),
		PassThreshold:  s.PassThreshold,
		Status:         s.Status,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
	}
	resp.RegistrationStart = formatDate(s.RegistrationStart)
	resp.RegistrationEnd = formatDate(s.RegistrationEnd)
	resp.GradeEntryStart = formatDate(s.GradeEntryStart)
	resp.GradeEntryEnd = formatDate(s.GradeEntryEnd)
	return resp
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format("2006-01-02")
	return &str
}

func toSemestersResponse(sems []management.Semester) []SemesterResponse {
	result := make([]SemesterResponse, len(sems))
	for i := range sems {
		result[i] = *toSemesterResponse(&sems[i])
	}
	return result
}

func toGenerateOfferingsResponse(r *management.GenerateOfferingsResult) GenerateOfferingsResponse {
	resp := GenerateOfferingsResponse{Created: r.Created, Skipped: r.Skipped}
	for _, d := range r.Details {
		resp.Details = append(resp.Details, OfferingRecordResponse{
			CourseID:   d.CourseID,
			CourseCode: d.CourseCode,
			CohortYear: d.CohortYear,
			Shift:      d.Shift,
			Status:     d.Status,
		})
	}
	return resp
}

func toBulkEnrollResponse(r *management.BulkEnrollResult) BulkEnrollResponse {
	resp := BulkEnrollResponse{Enrolled: r.Enrolled, Skipped: r.Skipped, Blocked: r.Blocked, Errors: r.Errors}
	if r.Details == nil {
		return resp
	}
	details := &EnrollDetailsResponse{}
	for _, e := range r.Details.Enrolled {
		details.Enrolled = append(details.Enrolled, EnrollRecordResponse(e))
	}
	for _, s := range r.Details.Skipped {
		details.Skipped = append(details.Skipped, SkipRecordResponse(s))
	}
	for _, b := range r.Details.Blocked {
		details.Blocked = append(details.Blocked, BlockedRecordResponse(b))
	}
	resp.Details = details
	return resp
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListSemesters handles GET /semesters.
func (h *Handler) ListSemesters(c *gin.Context) {
	var academicYearID *uuid.UUID
	if idStr := c.Query("academic_year_id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.BadRequest(c, "invalid academic_year_id")
			return
		}
		academicYearID = &id
	}

	semesters, err := h.semesters.List(c.Request.Context(), academicYearID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toSemestersResponse(semesters))
}

// CreateSemester handles POST /semesters.
func (h *Handler) CreateSemester(c *gin.Context) {

	var req CreateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	semester, err := h.semesters.Create(c.Request.Context(), &management.Semester{
		AcademicYearID:    req.AcademicYearID,
		Semester:          management.SemesterType(req.Semester),
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		RegistrationStart: req.RegistrationStart,
		RegistrationEnd:   req.RegistrationEnd,
		GradeEntryStart:   req.GradeEntryStart,
		GradeEntryEnd:     req.GradeEntryEnd,
		PassThreshold:     req.PassThreshold,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toSemesterResponse(semester))
}

// GetSemester handles GET /semesters/:id.
func (h *Handler) GetSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	semester, err := h.semesters.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toSemesterResponse(semester))
}

// UpdateSemester handles PUT /semesters/:id.
func (h *Handler) UpdateSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	upd := management.SemesterUpdate{
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		RegistrationStart: req.RegistrationStart,
		RegistrationEnd:   req.RegistrationEnd,
		GradeEntryStart:   req.GradeEntryStart,
		GradeEntryEnd:     req.GradeEntryEnd,
		PassThreshold:     req.PassThreshold,
	}
	if req.Semester != nil {
		st := management.SemesterType(*req.Semester)
		upd.Semester = &st
	}

	semester, err := h.semesters.Update(c.Request.Context(), id, upd)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toSemesterResponse(semester))
}

// UpdateSemesterStatus handles PUT /semesters/:id/status.
func (h *Handler) UpdateSemesterStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateSemesterStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	semester, err := h.semesters.UpdateStatus(c.Request.Context(), id, management.SemesterStatus(req.Status))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toSemesterResponse(semester))
}

// DeleteSemester handles DELETE /semesters/:id. The semester is soft-deleted
// with its offerings and stays recoverable until the purge window passes.
func (h *Handler) DeleteSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	role := middleware.GetUserRole(c)
	if role == nil || role.Level != "super_admin" {
		response.Forbidden(c, "only super admins can delete semesters")
		return
	}

	if err := h.semesters.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// SemesterCustom dispatches POST /semesters/:id — :definalize,
// :generateOfferings, :bulkEnroll, :end.
func (h *Handler) SemesterCustom(c *gin.Context) {
	info := authzhttp.Access(c)
	switch info.Action() {
	case authz.ActionDefinalize:
		h.definalizeSemester(c, info.TargetID())
	case authz.ActionGenerateOfferings:
		h.generateOfferings(c, info.TargetID())
	case authz.ActionBulkEnroll:
		h.bulkEnroll(c, info.TargetID())
	case authz.ActionEnd:
		h.endSemester(c, info.TargetID())
	default:
		response.NotFound(c, "unknown action")
	}
}

func (h *Handler) definalizeSemester(c *gin.Context, id uuid.UUID) {
	semester, err := h.semesters.Definalize(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toSemesterResponse(semester))
}

func (h *Handler) generateOfferings(c *gin.Context, id uuid.UUID) {
	var req GenerateOfferingsRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, "invalid request body")
		return
	}
	var shift *management.Shift
	if req.Shift != nil {
		s := management.Shift(*req.Shift)
		shift = &s
	}

	result, err := h.semesters.GenerateOfferings(c.Request.Context(), id, req.ProgramID, req.CohortYear, shift)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toGenerateOfferingsResponse(result))
}

func (h *Handler) bulkEnroll(c *gin.Context, id uuid.UUID) {
	var req BulkEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, "invalid request body")
		return
	}

	result, err := h.semesters.BulkEnroll(c.Request.Context(), id, req.ProgramID, req.CohortYear)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toBulkEnrollResponse(result))
}

func (h *Handler) endSemester(c *gin.Context, id uuid.UUID) {
	result, err := h.semesters.EndSemester(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, EndSemesterResponse(*result))
}
