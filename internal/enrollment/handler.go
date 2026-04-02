package enrollment

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
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

// Enrollment handlers

func (h *Handler) ListEnrollments(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := EnrollmentFilters{
		OfferingID: &offeringID,
		Query:      params.Query,
	}

	if enrollmentType := c.Query("enrollment_type"); enrollmentType != "" {
		filters.EnrollmentType = &enrollmentType
	}

	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}

	enrollments, hasMore, err := h.svc.ListEnrollments(c.Request.Context(), params, filters)
	if err != nil {
		h.log.Error("list enrollments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[EnrollmentResponse]{
		Data:    ToEnrollmentsResponse(enrollments),
		HasMore: hasMore,
	}
	if hasMore && len(enrollments) > 0 {
		last := enrollments[len(enrollments)-1]
		result.NextCursor = pagination.EncodeCursor(last.EnrolledAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) EnrollStudent(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req EnrollStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	enrollment, err := h.svc.EnrollStudent(c.Request.Context(), offeringID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToEnrollmentResponse(enrollment))
}

func (h *Handler) GetMyEnrollments(c *gin.Context) {
	userID := middleware.GetUserID(c)
	status := c.Query("status")

	enrollments, err := h.svc.GetMyEnrollments(c.Request.Context(), userID, status)
	if err != nil {
		h.log.Error("get my enrollments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToMyEnrollmentsResponse(enrollments))
}

func (h *Handler) GetAccessLevel(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	userID := middleware.GetUserID(c)
	access, err := h.svc.GetAccessLevel(c.Request.Context(), offeringID, userID)
	if err != nil {
		h.log.Error("get access level failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"access_level": access.String()})
}

func (h *Handler) DropEnrollment(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		response.BadRequest(c, "invalid student id")
		return
	}

	if err := h.svc.DropEnrollment(c.Request.Context(), offeringID, studentID); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

// Project group handlers

func (h *Handler) ListProjectGroups(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	groups, err := h.svc.ListProjectGroups(c.Request.Context(), offeringID)
	if err != nil {
		h.log.Error("list project groups failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToProjectGroupsResponse(groups))
}

func (h *Handler) CreateProjectGroup(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req CreateProjectGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	group, err := h.svc.CreateProjectGroup(c.Request.Context(), offeringID, req.Type, req.Name)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToProjectGroupResponse(group))
}

func (h *Handler) AssignToProjectGroup(c *gin.Context) {
	var req AssignToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.svc.AssignToProjectGroup(c.Request.Context(), req.StudentID, req.GroupID); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) RemoveFromProjectGroup(c *gin.Context) {
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		response.BadRequest(c, "invalid student id")
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	if err := h.svc.RemoveFromProjectGroup(c.Request.Context(), studentID, groupID); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

// Cohort group handlers

func (h *Handler) ListCohortGroups(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program id")
		return
	}

	cohortYear, err := parseIntParam(c, "cohort_year")
	if err != nil {
		response.BadRequest(c, "invalid cohort_year")
		return
	}

	stage, err := parseIntParam(c, "stage")
	if err != nil {
		response.BadRequest(c, "invalid stage")
		return
	}

	groups, err := h.svc.ListCohortGroups(c.Request.Context(), programID, cohortYear, stage)
	if err != nil {
		h.log.Error("list cohort groups failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCohortGroupsResponse(groups))
}

func (h *Handler) CreateCohortGroup(c *gin.Context) {
	var req CreateCohortGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	group, err := h.svc.CreateCohortGroup(c.Request.Context(), req.ProgramID, req.CohortYear, req.Stage, req.Type, req.Name)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToCohortGroupResponse(group))
}

func (h *Handler) AssignToCohortGroup(c *gin.Context) {
	var req AssignToGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.svc.AssignToCohortGroup(c.Request.Context(), req.StudentID, req.GroupID); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) RemoveFromCohortGroup(c *gin.Context) {
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		response.BadRequest(c, "invalid student id")
		return
	}

	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	if err := h.svc.RemoveFromCohortGroup(c.Request.Context(), studentID, groupID); err != nil {
		h.handleError(c, err)
		return
	}

	response.NoContent(c)
}

func parseIntParam(c *gin.Context, name string) (int, error) {
	val := c.Query(name)
	if val == "" {
		return 0, nil
	}
	var result int
	_, err := fmt.Sscanf(val, "%d", &result)
	return result, err
}

// Request handlers (pretake/retake)

func (h *Handler) CreatePretake(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreatePretakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	request, warning, err := h.svc.CreatePretake(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := CreateRequestResponse{
		Request: ToRequestResponse(request),
		Warning: warning,
	}
	response.Created(c, resp)
}

func (h *Handler) CreateRetake(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateRetakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	request, warning, err := h.svc.CreateRetake(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := CreateRequestResponse{
		Request: ToRequestResponse(request),
		Warning: warning,
	}
	response.Created(c, resp)
}

func (h *Handler) GetMyRequests(c *gin.Context) {
	userID := middleware.GetUserID(c)

	requests, err := h.svc.ListRequestsByStudent(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, ToRequestsResponse(requests))
}

func (h *Handler) ListRequests(c *gin.Context) {
	var filters RequestFilters

	if id := c.Query("semester_id"); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			response.BadRequest(c, "invalid semester_id")
			return
		}
		filters.SemesterID = &parsed
	}

	if id := c.Query("course_id"); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			response.BadRequest(c, "invalid course_id")
			return
		}
		filters.CourseID = &parsed
	}

	if t := c.Query("type"); t != "" {
		filters.Type = &t
	}

	if s := c.Query("status"); s != "" {
		filters.Status = &s
	}

	requests, err := h.svc.ListRequests(c.Request.Context(), filters)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, ToRequestsResponse(requests))
}

func (h *Handler) GetRequestByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	request, warning, err := h.svc.GetRequestWithWarning(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToRequestResponseWithWarning(request, warning))
}

func (h *Handler) ApproveRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	reviewerID := middleware.GetUserID(c)

	request, err := h.svc.ApproveRequest(c.Request.Context(), id, reviewerID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToRequestResponse(request))
}

func (h *Handler) RejectRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req RejectRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	reviewerID := middleware.GetUserID(c)

	request, err := h.svc.RejectRequest(c.Request.Context(), id, reviewerID, req.Reason)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToRequestResponse(request))
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEnrollmentNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrAlreadyEnrolled):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrNotEnrolled):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrOfferingNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrGroupNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrRequestNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrCourseNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrSemesterNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrDuplicateRequest):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrAlreadyReviewed):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrNoPrerequisite):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrPrerequisitePassed):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrCourseNotFailed):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrNotNaturalCohort):
		response.BadRequest(c, err.Error())
	default:
		h.log.Error("enrollment handler error", zap.Error(err))
		response.InternalError(c)
	}
}
