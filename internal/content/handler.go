package content

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/authz"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Section handlers

func (h *Handler) CreateSection(c *gin.Context) {
	var req CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, req.OfferingID) {
		response.Forbidden(c, "forbidden")
		return
	}

	section, err := h.service.CreateSection(c.Request.Context(), req.OfferingID, req.Title, req.UnlockAt)
	if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if err != nil {
		h.log.Error("create section failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToSectionResponse(section))
	}
}

func (h *Handler) GetSection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid section id")
		return
	}

	offeringID, err := h.service.GetOfferingIDBySectionID(c.Request.Context(), id)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionGet, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	section, err := h.service.GetSection(c.Request.Context(), id)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
	} else if err != nil {
		h.log.Error("get section failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToSectionResponse(section))
	}
}

func (h *Handler) ListSections(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !authz.Check(c, authz.ResourceOffering, authz.ActionGet, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	sections, err := h.service.ListSections(c.Request.Context(), offeringID)
	if err != nil {
		h.log.Error("list sections failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToSectionListResponse(sections))
}

func (h *Handler) UpdateSection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid section id")
		return
	}

	offeringID, err := h.service.GetOfferingIDBySectionID(c.Request.Context(), id)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	var req UpdateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	section, err := h.service.UpdateSection(c.Request.Context(), id, req.Title, req.UnlockAt)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
	} else if err != nil {
		h.log.Error("update section failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToSectionResponse(section))
	}
}

func (h *Handler) DeleteSection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid section id")
		return
	}

	offeringID, err := h.service.GetOfferingIDBySectionID(c.Request.Context(), id)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	err = h.service.DeleteSection(c.Request.Context(), id)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
	} else if errors.Is(err, ErrSectionNotEmpty) {
		response.Conflict(c, "section contains lessons")
	} else if err != nil {
		h.log.Error("delete section failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

// Lesson handlers

func (h *Handler) CreateLesson(c *gin.Context) {
	var req CreateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	offeringID, err := h.service.GetOfferingIDBySectionID(c.Request.Context(), req.SectionID)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	lesson, err := h.service.CreateLesson(c.Request.Context(), req.SectionID, req.Title)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
	} else if err != nil {
		h.log.Error("create lesson failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToLessonResponse(lesson))
	}
}

func (h *Handler) GetLesson(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid lesson id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), id)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionGet, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	var studentID *uuid.UUID
	if uid := middleware.GetUserID(c); uid != uuid.Nil {
		studentID = &uid
	}

	lesson, err := h.service.GetLessonWithMeta(c.Request.Context(), id, studentID)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
	} else if err != nil {
		h.log.Error("get lesson failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToLessonWithMetaResponse(lesson))
	}
}

func (h *Handler) ListLessons(c *gin.Context) {
	sectionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid section id")
		return
	}

	offeringID, err := h.service.GetOfferingIDBySectionID(c.Request.Context(), sectionID)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionGet, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	lessons, err := h.service.ListLessons(c.Request.Context(), sectionID)
	if errors.Is(err, ErrSectionNotFound) {
		response.NotFound(c, "section not found")
	} else if err != nil {
		h.log.Error("list lessons failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToLessonListResponse(lessons))
	}
}

func (h *Handler) UpdateLesson(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid lesson id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), id)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	var req UpdateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	lesson, err := h.service.UpdateLesson(c.Request.Context(), id, req.Title, req.Body, req.Mode, req.Type, req.UnlockAt, req.DurationHours, req.AttendanceRequired, req.AllowDownload)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
	} else if errors.Is(err, ErrInvalidMode) {
		response.BadRequest(c, "invalid lesson mode")
	} else if errors.Is(err, ErrInvalidType) {
		response.BadRequest(c, "invalid lesson type")
	} else if err != nil {
		h.log.Error("update lesson failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToLessonResponse(lesson))
	}
}

func (h *Handler) DeleteLesson(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid lesson id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), id)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	err = h.service.DeleteLesson(c.Request.Context(), id)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
	} else if err != nil {
		h.log.Error("delete lesson failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

// Attachment handlers

func (h *Handler) AddAttachment(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid lesson id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), lessonID)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	attachment, err := h.service.AddAttachment(c.Request.Context(), lessonID, req.StoredFileID, userID, req.DisplayName)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
	} else if errors.Is(err, ErrStoredFileNotFound) {
		response.NotFound(c, "stored file not found")
	} else if errors.Is(err, ErrDuplicateDisplayName) {
		response.Conflict(c, "attachment with this name already exists")
	} else if err != nil {
		h.log.Error("add attachment failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, AttachmentResponse{ID: attachment.ID, DisplayName: attachment.DisplayName})
	}
}

func (h *Handler) RemoveAttachment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByAttachmentID(c.Request.Context(), id)
	if errors.Is(err, ErrAttachmentNotFound) {
		response.NotFound(c, "attachment not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	err = h.service.RemoveAttachment(c.Request.Context(), id)
	if errors.Is(err, ErrAttachmentNotFound) {
		response.NotFound(c, "attachment not found")
	} else if err != nil {
		h.log.Error("remove attachment failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) GetAttachmentURL(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid lesson id")
		return
	}

	displayName := c.Param("display_name")
	if displayName == "" {
		response.BadRequest(c, "display name required")
		return
	}

	attachment, err := h.service.GetAttachmentByName(c.Request.Context(), lessonID, displayName)
	if errors.Is(err, ErrAttachmentNotFound) {
		response.NotFound(c, "attachment not found")
	} else if err != nil {
		h.log.Error("get attachment failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"stored_file_id": attachment.StoredFileID})
	}
}

// Schedule handlers

func (h *Handler) AddSchedule(c *gin.Context) {
	lessonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid lesson id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByLessonID(c.Request.Context(), lessonID)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	var req AddScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	schedule, err := h.service.AddSchedule(c.Request.Context(), lessonID, req.CohortGroupID, req.ScheduledAt, req.Room)
	if errors.Is(err, ErrLessonNotFound) {
		response.NotFound(c, "lesson not found")
	} else if errors.Is(err, ErrGroupNotFound) {
		response.NotFound(c, "cohort group not found")
	} else if errors.Is(err, ErrDuplicateSchedule) {
		response.Conflict(c, "schedule for this group already exists")
	} else if err != nil {
		h.log.Error("add schedule failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, gin.H{"id": schedule.ID, "scheduled_at": schedule.ScheduledAt, "room": schedule.Room})
	}
}

func (h *Handler) UpdateSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid schedule id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByScheduleID(c.Request.Context(), id)
	if errors.Is(err, ErrScheduleNotFound) {
		response.NotFound(c, "schedule not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	schedule, err := h.service.UpdateSchedule(c.Request.Context(), id, req.ScheduledAt, req.Room)
	if errors.Is(err, ErrScheduleNotFound) {
		response.NotFound(c, "schedule not found")
	} else if err != nil {
		h.log.Error("update schedule failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, gin.H{"id": schedule.ID, "scheduled_at": schedule.ScheduledAt, "room": schedule.Room})
	}
}

func (h *Handler) RemoveSchedule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid schedule id")
		return
	}

	offeringID, err := h.service.GetOfferingIDByScheduleID(c.Request.Context(), id)
	if errors.Is(err, ErrScheduleNotFound) {
		response.NotFound(c, "schedule not found")
		return
	}
	if err != nil {
		h.log.Error("resolve offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if !authz.Check(c, authz.ResourceOffering, authz.ActionUpdate, offeringID) {
		response.Forbidden(c, "forbidden")
		return
	}

	err = h.service.RemoveSchedule(c.Request.Context(), id)
	if errors.Is(err, ErrScheduleNotFound) {
		response.NotFound(c, "schedule not found")
	} else if err != nil {
		h.log.Error("remove schedule failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) GetMyClasses(c *gin.Context) {
	studentID := middleware.GetUserID(c)
	dateFormat := "02-01-2006"

	dateStr := c.Query("date")
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time
	var err error

	if dateStr != "" {
		date, err := time.Parse(dateFormat, dateStr)
		if err != nil {
			response.BadRequest(c, "invalid date format, use DD-MM-YYYY")
			return
		}
		from = date
		to = date.Add(24 * time.Hour)
	} else if fromStr != "" && toStr != "" {
		from, err = time.Parse(dateFormat, fromStr)
		if err != nil {
			response.BadRequest(c, "invalid from date format, use DD-MM-YYYY")
			return
		}
		to, err = time.Parse(dateFormat, toStr)
		if err != nil {
			response.BadRequest(c, "invalid to date format, use DD-MM-YYYY")
			return
		}
		to = to.Add(24 * time.Hour)
	} else {
		response.BadRequest(c, "date or from/to query params required")
		return
	}

	entries, err := h.service.GetMyClasses(c.Request.Context(), studentID, from, to)
	if err != nil {
		h.log.Error("get my classes failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToCalendarListResponse(entries))
}
