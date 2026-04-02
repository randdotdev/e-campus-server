package academic

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	svc *Service
	log *zap.Logger
}

func NewHandler(svc *Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) ListAcademicYears(c *gin.Context) {
	years, err := h.svc.ListAcademicYears(c.Request.Context())
	if err != nil {
		h.log.Error("list academic years failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, ToAcademicYearsResponse(years))
}

func (h *Handler) CreateAcademicYear(c *gin.Context) {
	var req CreateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	year, err := h.svc.CreateAcademicYear(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToAcademicYearResponse(year))
}

func (h *Handler) GetAcademicYear(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	year, err := h.svc.GetAcademicYear(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToAcademicYearResponse(year))
}

func (h *Handler) UpdateAcademicYear(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	year, err := h.svc.UpdateAcademicYear(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToAcademicYearResponse(year))
}

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

	semesters, err := h.svc.ListSemesters(c.Request.Context(), academicYearID)
	if err != nil {
		h.log.Error("list semesters failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToSemestersResponse(semesters))
}

func (h *Handler) CreateSemester(c *gin.Context) {
	var req CreateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	semester, err := h.svc.CreateSemester(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToSemesterResponse(semester))
}

func (h *Handler) GetSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	semester, err := h.svc.GetSemester(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToSemesterResponse(semester))
}

func (h *Handler) UpdateSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	semester, err := h.svc.UpdateSemester(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToSemesterResponse(semester))
}

func (h *Handler) UpdateSemesterStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateSemesterStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	semester, err := h.svc.UpdateSemesterStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToSemesterResponse(semester))
}

func (h *Handler) DefinalizeSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	semester, err := h.svc.DefinalizeSemester(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToSemesterResponse(semester))
}

func (h *Handler) GenerateOfferings(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req GenerateOfferingsRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.GenerateOfferings(c.Request.Context(), id, req.ProgramID, req.CohortYear)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *Handler) BulkEnroll(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req BulkEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.BulkEnroll(c.Request.Context(), id, req.ProgramID, req.CohortYear)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *Handler) EndSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	result, err := h.svc.EndSemester(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, result)
}

func (h *Handler) ListCurriculum(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	cohortYear := 0
	if yearStr := c.Query("cohort_year"); yearStr != "" {
		if _, err := parseIntParam(yearStr, &cohortYear); err != nil {
			response.BadRequest(c, "invalid cohort_year")
			return
		}
	}

	curriculum, err := h.svc.ListCurriculum(c.Request.Context(), programID, cohortYear)
	if err != nil {
		h.log.Error("list curriculum failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCurriculumsResponse(curriculum))
}

func (h *Handler) AddToCurriculum(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	var req AddCurriculumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	req.ProgramID = programID

	curriculum, err := h.svc.AddToCurriculum(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToCurriculumResponse(curriculum))
}

func (h *Handler) RemoveFromCurriculum(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	if err := h.svc.RemoveFromCurriculum(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) ListRequirements(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	cohortYear := 0
	if yearStr := c.Query("cohort_year"); yearStr != "" {
		if _, err := parseIntParam(yearStr, &cohortYear); err != nil {
			response.BadRequest(c, "invalid cohort_year")
			return
		}
	}

	requirements, err := h.svc.ListRequirements(c.Request.Context(), programID, cohortYear)
	if err != nil {
		h.log.Error("list requirements failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToRequirementsResponse(requirements))
}

func (h *Handler) SetRequirement(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	var req SetRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	req.ProgramID = programID
	req.CreatedBy = middleware.GetUserID(c)

	requirement, err := h.svc.SetRequirement(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToRequirementResponse(requirement))
}

func parseIntParam(str string, target *int) (bool, error) {
	var val int
	_, err := parseIntVal(str, &val)
	if err != nil {
		return false, err
	}
	*target = val
	return true, nil
}

func parseIntVal(str string, target *int) (bool, error) {
	if str == "" {
		return false, nil
	}
	_, err := scanInt(str, target)
	return err == nil, err
}

func scanInt(str string, target *int) (int, error) {
	n := 0
	for _, c := range str {
		if c < '0' || c > '9' {
			return 0, errors.New("invalid integer")
		}
		n = n*10 + int(c-'0')
	}
	*target = n
	return n, nil
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrAcademicYearNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrSemesterNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrCurriculumNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrRequirementNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrProgramNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrCourseNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrDuplicateYear):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrDuplicateSemester):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrDuplicateCurriculum):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrInvalidStatus):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrInvalidStatusTransition):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrSemesterNotFinalized):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrSemesterArchived):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrOfferingsNotFinalized):
		response.BadRequest(c, err.Error())
	default:
		h.log.Error("academic handler error", zap.Error(err))
		response.InternalError(c)
	}
}
