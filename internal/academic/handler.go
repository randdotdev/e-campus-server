package academic

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

func (h *Handler) ListAcademicYears(c *gin.Context) {
	years, err := h.service.ListAcademicYears(c.Request.Context())
	if err != nil {
		h.log.Error("list academic years failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, ToAcademicYearsResponse(years))
}

func (h *Handler) CreateAcademicYear(c *gin.Context) {
	if !authz.Check(c, authz.ResourceAcademicYear, authz.ActionCreate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req CreateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	year, err := h.service.CreateAcademicYear(c.Request.Context(), req)
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

	year, err := h.service.GetAcademicYear(c.Request.Context(), id)
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

	if !authz.Check(c, authz.ResourceAcademicYear, authz.ActionUpdate, id) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	year, err := h.service.UpdateAcademicYear(c.Request.Context(), id, req)
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

	semesters, err := h.service.ListSemesters(c.Request.Context(), academicYearID)
	if err != nil {
		h.log.Error("list semesters failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToSemestersResponse(semesters))
}

func (h *Handler) CreateSemester(c *gin.Context) {
	if !authz.Check(c, authz.ResourceSemester, authz.ActionCreate) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req CreateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	semester, err := h.service.CreateSemester(c.Request.Context(), req)
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

	semester, err := h.service.GetSemester(c.Request.Context(), id)
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

	if !authz.Check(c, authz.ResourceSemester, authz.ActionUpdate, id) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateSemesterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	semester, err := h.service.UpdateSemester(c.Request.Context(), id, req)
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

	if !authz.Check(c, authz.ResourceSemester, authz.ActionUpdate, id) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req UpdateSemesterStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	semester, err := h.service.UpdateSemesterStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToSemesterResponse(semester))
}

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

	if err := h.service.DeleteSemester(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) DefinalizeSemester(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	if !authz.Check(c, authz.ResourceSemester, authz.ActionUpdate, id) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	semester, err := h.service.DefinalizeSemester(c.Request.Context(), id)
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

	if !authz.Check(c, authz.ResourceSemester, authz.ActionUpdate, id) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req GenerateOfferingsRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, "invalid request body")
		return
	}

	result, err := h.service.GenerateOfferings(c.Request.Context(), id, req.ProgramID, req.CohortYear, req.Shift)
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

	if !authz.Check(c, authz.ResourceSemester, authz.ActionUpdate, id) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req BulkEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, "invalid request body")
		return
	}

	result, err := h.service.BulkEnroll(c.Request.Context(), id, req.ProgramID, req.CohortYear)
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

	if !authz.Check(c, authz.ResourceSemester, authz.ActionUpdate, id) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	result, err := h.service.EndSemester(c.Request.Context(), id)
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

	items, err := h.service.ListCurriculumItems(c.Request.Context(), programID, cohortYear)
	if err != nil {
		h.log.Error("list curriculum failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCurriculumItemsResponse(items))
}

func (h *Handler) AddToCurriculum(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	if !authz.Check(c, authz.ResourceCurriculum, authz.ActionUpdate, programID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req AddCurriculumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	req.ProgramID = programID

	curriculum, err := h.service.AddToCurriculum(c.Request.Context(), req)
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

	curriculum, err := h.service.GetCurriculumByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	if !authz.Check(c, authz.ResourceCurriculum, authz.ActionUpdate, curriculum.ProgramID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	if err := h.service.RemoveFromCurriculum(c.Request.Context(), id); err != nil {
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

	requirements, err := h.service.ListRequirements(c.Request.Context(), programID, cohortYear)
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

	if !authz.Check(c, authz.ResourceCurriculum, authz.ActionUpdate, programID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var req SetRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	req.ProgramID = programID
	req.CreatedBy = middleware.GetUserID(c)

	requirement, err := h.service.SetRequirement(c.Request.Context(), req)
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
	case errors.Is(err, ErrSemesterNotActive):
		response.BadRequest(c, err.Error())
	default:
		h.log.Error("academic handler error", zap.Error(err))
		response.InternalError(c)
	}
}
