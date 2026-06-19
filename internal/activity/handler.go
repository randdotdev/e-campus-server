package activity

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/pagination"
	"github.com/randdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, auth gin.HandlerFunc) {
	activities := rg.Group("/activities")
	activities.Use(auth)
	{
		activities.GET("", h.ListActivities)
		activities.POST("", h.CreateActivity)
		activities.GET("/:id", h.GetActivity)
		activities.GET("/:id/translation", h.GetTranslation)
		activities.PUT("/:id", h.UpdateActivity)
		activities.DELETE("/:id", h.DeleteActivity)
		activities.PUT("/:id/pin", h.PinActivity)
		activities.POST("/:id/attachments", h.AddAttachment)
	}

	attachments := rg.Group("/activity-attachments")
	attachments.Use(auth)
	{
		attachments.DELETE("/:id", h.RemoveAttachment)
	}
}

func (h *Handler) CreateActivity(c *gin.Context) {
	var req CreateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !h.canPublish(c, req.PublisherType, req.PublisherID) {
		response.Forbidden(c, "not authorized to publish activity")
		return
	}

	userID := middleware.GetUserID(c)

	titleEN := ""
	if req.TitleEN != nil {
		titleEN = *req.TitleEN
	}
	bodyEN := ""
	if req.BodyEN != nil {
		bodyEN = *req.BodyEN
	}

	a, err := h.service.CreateActivity(
		c.Request.Context(),
		userID,
		req.PublisherType,
		req.PublisherID,
		req.Type,
		titleEN,
		req.TitleLocal,
		bodyEN,
		req.BodyLocal,
		req.CoverImageID,
		req.PublishAt,
		req.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidPublisher) {
			response.BadRequest(c, "invalid publisher")
			return
		}
		if errors.Is(err, ErrInvalidType) {
			response.BadRequest(c, "invalid activity type")
			return
		}
		h.log.Error("create activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	resp := ActivityResponse{
		ID:            a.ID,
		PublisherType: a.PublisherType,
		PublisherID:   a.PublisherID,
		Type:          a.Type,
		TitleEN:       a.TitleEN,
		TitleLocal:    a.TitleLocal,
		BodyEN:        a.BodyEN,
		BodyLocal:     a.BodyLocal,
		CoverImageID:  a.CoverImageID,
		IsPinned:      a.IsPinned,
		PublishAt:     a.PublishAt,
		ExpiresAt:     a.ExpiresAt,
		Status:        GetStatus(a, time.Now()),
		AuthorID:      a.AuthorID,
		CreatedAt:     a.CreatedAt,
	}
	response.Created(c, resp)
}

func (h *Handler) GetActivity(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	userID := middleware.GetUserID(c)

	a, attachments, err := h.service.GetActivity(c.Request.Context(), activityID, true)
	if err != nil {
		if errors.Is(err, ErrActivityNotFound) {
			response.NotFound(c, "activity not found")
			return
		}
		h.log.Error("get activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	isAdmin := h.canPublish(c, a.PublisherType, a.PublisherID)
	isAuthor := a.AuthorID == userID

	if !isAdmin && !isAuthor && !CanView(&a.Activity, false, time.Now()) {
		response.NotFound(c, "activity not found")
		return
	}

	resp := ToActivityResponse(a, attachments, time.Now())
	response.OK(c, resp)
}

func (h *Handler) GetTranslation(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	lang := c.Query("lang")
	if lang == "" {
		lang = LangLocal
	}

	userID := middleware.GetUserID(c)

	existing, err := h.service.GetActivityByID(c.Request.Context(), activityID)
	if err != nil {
		h.log.Error("get activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	isAdmin := h.canPublish(c, existing.PublisherType, existing.PublisherID)
	isAuthor := existing.AuthorID == userID

	title, body, err := h.service.GetTranslation(c.Request.Context(), activityID, lang, isAdmin || isAuthor)
	if err != nil {
		if errors.Is(err, ErrActivityNotFound) {
			response.NotFound(c, "activity not found")
			return
		}
		if errors.Is(err, ErrInvalidLanguage) {
			response.BadRequest(c, "invalid language")
			return
		}
		if errors.Is(err, ErrTranslationMissing) {
			response.NotFound(c, "translation not available")
			return
		}
		h.log.Error("get translation failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, TranslationResponse{Title: title, Body: body})
}

func (h *Handler) ListActivities(c *gin.Context) {
	publisherType := c.Query("publisher_type")
	if publisherType == "" {
		publisherType = PublisherUniversity
	}

	var publisherID *uuid.UUID
	if publisherIDStr := c.Query("publisher_id"); publisherIDStr != "" {
		id, err := uuid.Parse(publisherIDStr)
		if err != nil {
			response.BadRequest(c, "invalid publisher_id")
			return
		}
		publisherID = &id
	}

	activityType := c.Query("type")

	isAdmin := h.canPublish(c, publisherType, publisherID)
	params := pagination.ParsePageParams(c)

	activities, hasMore, err := h.service.ListActivities(c.Request.Context(), publisherType, publisherID, activityType, isAdmin, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list activities failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	activityIDs := make([]uuid.UUID, len(activities))
	for i := range activities {
		activityIDs[i] = activities[i].ID
	}

	attachmentsMap, err := h.service.GetAttachmentsForActivities(c.Request.Context(), activityIDs)
	if err != nil {
		h.log.Error("get attachments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	now := time.Now()
	result := pagination.PageResult[ActivityResponse]{
		Data:    ToActivityResponses(activities, attachmentsMap, now),
		HasMore: hasMore,
	}
	if hasMore && len(activities) > 0 {
		last := activities[len(activities)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateActivity(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	var req UpdateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	existing, err := h.service.GetActivityByID(c.Request.Context(), activityID)
	if err != nil {
		h.log.Error("get activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	isAdmin := h.canPublish(c, existing.PublisherType, existing.PublisherID)

	a, err := h.service.UpdateActivity(
		c.Request.Context(),
		activityID,
		userID,
		isAdmin,
		req.TitleEN,
		req.BodyEN,
		req.TitleLocal,
		req.BodyLocal,
		req.Type,
		req.CoverImageID,
		req.PublishAt,
		req.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, ErrActivityNotFound) {
			response.NotFound(c, "activity not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to update this activity")
			return
		}
		if errors.Is(err, ErrInvalidType) {
			response.BadRequest(c, "invalid activity type")
			return
		}
		h.log.Error("update activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	resp := ActivityResponse{
		ID:            a.ID,
		PublisherType: a.PublisherType,
		PublisherID:   a.PublisherID,
		Type:          a.Type,
		TitleEN:       a.TitleEN,
		TitleLocal:    a.TitleLocal,
		BodyEN:        a.BodyEN,
		BodyLocal:     a.BodyLocal,
		CoverImageID:  a.CoverImageID,
		IsPinned:      a.IsPinned,
		PublishAt:     a.PublishAt,
		ExpiresAt:     a.ExpiresAt,
		Status:        GetStatus(a, time.Now()),
		AuthorID:      a.AuthorID,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
	response.OK(c, resp)
}

func (h *Handler) DeleteActivity(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	userID := middleware.GetUserID(c)

	existing, err := h.service.GetActivityByID(c.Request.Context(), activityID)
	if err != nil {
		h.log.Error("get activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	isAdmin := h.canPublish(c, existing.PublisherType, existing.PublisherID)

	if err := h.service.DeleteActivity(c.Request.Context(), activityID, userID, isAdmin); err != nil {
		if errors.Is(err, ErrActivityNotFound) {
			response.NotFound(c, "activity not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to delete this activity")
			return
		}
		h.log.Error("delete activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) PinActivity(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	var req PinActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	existing, err := h.service.GetActivityByID(c.Request.Context(), activityID)
	if err != nil {
		h.log.Error("get activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	isAdmin := h.canPublish(c, existing.PublisherType, existing.PublisherID)

	if err := h.service.PinActivity(c.Request.Context(), activityID, isAdmin, req.Pinned); err != nil {
		if errors.Is(err, ErrActivityNotFound) {
			response.NotFound(c, "activity not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "only admins can pin activities")
			return
		}
		h.log.Error("pin activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) AddAttachment(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	existing, err := h.service.GetActivityByID(c.Request.Context(), activityID)
	if err != nil {
		h.log.Error("get activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	isAdmin := h.canPublish(c, existing.PublisherType, existing.PublisherID)

	attachment, err := h.service.AddAttachment(
		c.Request.Context(),
		activityID,
		userID,
		isAdmin,
		req.StoredFileID,
		req.DisplayName,
		req.FileType,
		req.SizeBytes,
		req.OrderIndex,
	)
	if err != nil {
		if errors.Is(err, ErrActivityNotFound) {
			response.NotFound(c, "activity not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to add attachment")
			return
		}
		if errors.Is(err, ErrInvalidFileType) {
			response.BadRequest(c, "invalid file type")
			return
		}
		if errors.Is(err, ErrFileTooLarge) {
			response.BadRequest(c, "file too large")
			return
		}
		h.log.Error("add attachment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToAttachmentResponse(attachment))
}

func (h *Handler) RemoveAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	userID := middleware.GetUserID(c)

	attachment, err := h.service.GetAttachmentByID(c.Request.Context(), attachmentID)
	if err != nil {
		h.log.Error("get attachment failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if attachment == nil {
		response.NotFound(c, "attachment not found")
		return
	}

	existing, err := h.service.GetActivityByID(c.Request.Context(), attachment.ActivityID)
	if err != nil {
		h.log.Error("get activity failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	isAdmin := h.canPublish(c, existing.PublisherType, existing.PublisherID)

	if err := h.service.RemoveAttachment(c.Request.Context(), attachmentID, userID, isAdmin); err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			response.NotFound(c, "attachment not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to remove attachment")
			return
		}
		h.log.Error("remove attachment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) canPublish(c *gin.Context, publisherType string, publisherID *uuid.UUID) bool {
	switch publisherType {
	case PublisherUniversity:
		return authz.Check(c, authz.ResourceActivity, authz.ActionCreate)
	case PublisherCollege:
		if publisherID == nil {
			return false
		}
		return authz.Check(c, authz.ResourceActivity, authz.ActionCreate, *publisherID)
	case PublisherDepartment:
		if publisherID == nil {
			return false
		}
		return authz.Check(c, authz.ResourceActivity, authz.ActionCreate, *publisherID)
	}
	return false
}

func (h *Handler) getPreferredLang(c *gin.Context, defaultLang string) string {
	lang := c.Query("lang")
	if lang == LangEN || lang == LangLocal {
		return lang
	}
	return defaultLang
}
