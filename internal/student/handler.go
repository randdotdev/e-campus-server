package student

import (
	"errors"

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

func (h *Handler) ListStudents(c *gin.Context) {
	params := pagination.ParsePageParams(c)

	var filters StudentFilters
	if id := c.Query("program_id"); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			response.BadRequest(c, "invalid program_id")
			return
		}
		filters.ProgramID = &parsed
	}
	if year := c.Query("cohort_year"); year != "" {
		var val int
		if _, err := scanInt(year, &val); err != nil {
			response.BadRequest(c, "invalid cohort_year")
			return
		}
		filters.CohortYear = &val
	}
	if stage := c.Query("stage"); stage != "" {
		var val int
		if _, err := scanInt(stage, &val); err != nil {
			response.BadRequest(c, "invalid stage")
			return
		}
		filters.Stage = &val
	}
	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}
	if shift := c.Query("shift"); shift != "" {
		filters.Shift = &shift
	}
	if params.Query != "" {
		filters.Query = &params.Query
	}

	students, hasMore, err := h.svc.ListStudents(c.Request.Context(), params, filters)
	if err != nil {
		h.log.Error("list students failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[StudentResponse]{
		Data:    ToStudentsResponse(students),
		HasMore: hasMore,
	}
	if hasMore && len(students) > 0 {
		last := students[len(students)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) CreateStudent(c *gin.Context) {
	var req CreateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	student, err := h.svc.CreateStudent(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToStudentResponse(student))
}

func (h *Handler) GetStudent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	student, err := h.svc.GetStudent(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToStudentResponse(student))
}

func (h *Handler) GetMyStudentRecord(c *gin.Context) {
	userID := middleware.GetUserID(c)

	student, err := h.svc.GetStudentByUserID(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToStudentResponse(student))
}

func (h *Handler) UpdateStudent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	student, err := h.svc.UpdateStudent(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToStudentResponse(student))
}

func (h *Handler) UpdateStudentStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	student, err := h.svc.UpdateStudentStatus(c.Request.Context(), id, req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToStudentResponse(student))
}

func (h *Handler) RequestLeave(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req RequestLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	leave, semesterIDs, err := h.svc.RequestLeave(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Created(c, ToLeaveResponse(leave, semesterIDs))
}

func (h *Handler) ApproveLeave(c *gin.Context) {
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid leave_id")
		return
	}

	approverID := middleware.GetUserID(c)

	leave, semesterIDs, err := h.svc.ApproveLeave(c.Request.Context(), leaveID, approverID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToLeaveResponse(leave, semesterIDs))
}

func (h *Handler) EndLeave(c *gin.Context) {
	leaveID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid leave_id")
		return
	}

	leave, err := h.svc.EndLeave(c.Request.Context(), leaveID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, ToLeaveResponse(leave, nil))
}

func (h *Handler) ListLeaves(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	leaves, err := h.svc.ListLeaves(c.Request.Context(), id)
	if err != nil {
		h.log.Error("list leaves failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToLeavesResponse(leaves))
}

func (h *Handler) ListCohortHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	history, err := h.svc.ListCohortHistory(c.Request.Context(), id)
	if err != nil {
		h.log.Error("list cohort history failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCohortHistoriesResponse(history))
}

func (h *Handler) GetTranscript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	transcript, err := h.svc.GetTranscript(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, transcript)
}

func (h *Handler) GetMyTranscript(c *gin.Context) {
	userID := middleware.GetUserID(c)

	student, err := h.svc.GetStudentByUserID(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	transcript, err := h.svc.GetTranscript(c.Request.Context(), student.ID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.OK(c, transcript)
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
	case errors.Is(err, ErrStudentNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrLeaveNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrProgramNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrDuplicateStudent):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrInvalidStatus):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrInvalidLeaveType):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrAlreadyOnLeave):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrNotOnLeave):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrLeaveEnded):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrLeaveAlreadyApproved):
		response.Conflict(c, err.Error())
	default:
		h.log.Error("student handler error", zap.Error(err))
		response.InternalError(c)
	}
}
