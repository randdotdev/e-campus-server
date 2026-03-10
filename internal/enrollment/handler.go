package enrollment

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

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

	resp := CreateResponse{
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

	resp := CreateResponse{
		Request: ToRequestResponse(request),
		Warning: warning,
	}
	response.Created(c, resp)
}

func (h *Handler) GetMyRequests(c *gin.Context) {
	userID := middleware.GetUserID(c)

	requests, err := h.svc.ListByStudent(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, ToRequestsResponse(requests))
}

func (h *Handler) List(c *gin.Context) {
	var filters Filters

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

	requests, err := h.svc.List(c.Request.Context(), filters)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, ToRequestsResponse(requests))
}

func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	request, warning, err := h.svc.GetByIDWithWarning(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToRequestResponseWithWarning(request, warning))
}

func (h *Handler) Approve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	reviewerID := middleware.GetUserID(c)

	request, err := h.svc.Approve(c.Request.Context(), id, reviewerID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToRequestResponse(request))
}

func (h *Handler) Reject(c *gin.Context) {
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

	request, err := h.svc.Reject(c.Request.Context(), id, reviewerID, req.Reason)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToRequestResponse(request))
}

func (h *Handler) handleError(c *gin.Context, err error) {
	switch {
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
		response.InternalError(c)
	}
}
