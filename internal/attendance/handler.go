package attendance

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) InitializeAttendance(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), lessonID)
	if err != nil {
		handleError(c, err)
		return
	}
	if !authz.Check(c, authz.ResourceAttendance, authz.ActionUpdate, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	count, err := h.service.InitializeAttendance(c.Request.Context(), lessonID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, InitializeResponse{
		Initialized: count,
		Message:     fmt.Sprintf("Attendance initialized for %d students", count),
	})
}

func (h *Handler) MarkAttendance(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), lessonID)
	if err != nil {
		handleError(c, err)
		return
	}
	if !authz.Check(c, authz.ResourceAttendance, authz.ActionUpdate, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req MarkAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	markerID := middleware.GetUserID(c)
	if err := h.service.MarkAttendance(c.Request.Context(), lessonID, markerID, ToAttendanceUpdates(req.Records)); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateAttendance(c *gin.Context) {
	attendanceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attendance id"})
		return
	}

	offeringID, err := h.service.GetOfferingIDByAttendanceID(c.Request.Context(), attendanceID)
	if err != nil {
		handleError(c, err)
		return
	}
	if !authz.Check(c, authz.ResourceAttendance, authz.ActionUpdate, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req UpdateAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	markerID := middleware.GetUserID(c)
	record, err := h.service.UpdateAttendance(c.Request.Context(), attendanceID, markerID, req.Percentage)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, ToAttendanceRecordResponse(*record))
}

func (h *Handler) GetLessonAttendance(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), lessonID)
	if err != nil {
		handleError(c, err)
		return
	}
	if !authz.Check(c, authz.ResourceAttendance, authz.ActionGet, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	records, err := h.service.GetLessonAttendance(c.Request.Context(), lessonID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": ToAttendanceRecordResponses(records)})
}

func (h *Handler) GetOfferingAttendance(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offering id"})
		return
	}

	if !authz.Check(c, authz.ResourceAttendance, authz.ActionGet, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	records, err := h.service.GetOfferingAttendance(c.Request.Context(), offeringID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": ToAttendanceRecordResponses(records)})
}

func (h *Handler) GetAttendanceSummaries(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offering id"})
		return
	}

	if !authz.Check(c, authz.ResourceAttendance, authz.ActionGet, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	summaries, err := h.service.GetAttendanceSummaries(c.Request.Context(), offeringID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"summaries": ToAttendanceSummaryResponses(summaries)})
}

func (h *Handler) RequestExcuse(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lesson id"})
		return
	}

	var req ExcuseRequestInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	studentID := middleware.GetUserID(c)
	excuse, err := h.service.RequestExcuse(c.Request.Context(), lessonID, studentID, req.Reason)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, ToExcuseRequestResponse(*excuse))
}

func (h *Handler) ReviewExcuse(c *gin.Context) {
	excuseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid excuse request id"})
		return
	}

	offeringID, err := h.service.GetOfferingIDByExcuseID(c.Request.Context(), excuseID)
	if err != nil {
		handleError(c, err)
		return
	}
	if !authz.Check(c, authz.ResourceAttendance, authz.ActionUpdate, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	var req ReviewExcuseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	reviewerID := middleware.GetUserID(c)
	if err := h.service.ReviewExcuse(c.Request.Context(), excuseID, reviewerID, req.Status, req.Note); err != nil {
		handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetPendingExcuses(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offering id"})
		return
	}

	if !authz.Check(c, authz.ResourceAttendance, authz.ActionGet, offeringID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
		return
	}

	excuses, err := h.service.GetPendingExcuses(c.Request.Context(), offeringID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"excuses": ToExcuseRequestResponses(excuses)})
}

func (h *Handler) GetMyOfferingAttendance(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offering id"})
		return
	}

	studentID := middleware.GetUserID(c)
	records, err := h.service.GetStudentAttendance(c.Request.Context(), studentID, offeringID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"records": ToStudentAttendanceResponses(records)})
}

func (h *Handler) GetMyAttendance(c *gin.Context) {
	studentID := middleware.GetUserID(c)
	courses, err := h.service.GetMyCourseAttendances(c.Request.Context(), studentID)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"courses": ToCourseAttendanceResponses(courses)})
}

func handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrLessonNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAttendanceNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrExcuseNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, ErrAttendanceNotRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
	case errors.Is(err, ErrExcuseAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrExcuseAlreadyReviewed):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidPercentage):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
	case errors.Is(err, ErrInvalidExcuseStatus):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
	case errors.Is(err, ErrStudentNotEnrolled):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, ErrCannotExcuseOwnAttendance):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
