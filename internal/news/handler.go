package news

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/permission"
	"github.com/ranjdotdev/e-campus-server/internal/response"
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
	news := rg.Group("/news")
	news.Use(auth)
	{
		news.GET("", h.ListNews)
		news.POST("", h.CreateNews)
		news.GET("/:id", h.GetNews)
		news.GET("/:id/translation", h.GetTranslation)
		news.PUT("/:id", h.UpdateNews)
		news.DELETE("/:id", h.DeleteNews)
		news.PUT("/:id/pin", h.PinNews)
		news.POST("/:id/attachments", h.AddAttachment)
	}

	attachments := rg.Group("/news-attachments")
	attachments.Use(auth)
	{
		attachments.DELETE("/:id", h.RemoveAttachment)
	}
}

func (h *Handler) CreateNews(c *gin.Context) {
	var req CreateNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !h.canPublish(c, req.PublisherType, req.PublisherID) {
		response.Forbidden(c, "not authorized to publish news")
		return
	}

	userID := middleware.GetUserID(c)

	news, err := h.service.CreateNews(
		c.Request.Context(),
		userID,
		req.PublisherType,
		req.PublisherID,
		req.Category,
		req.TitleEN,
		req.TitleLocal,
		req.BodyEN,
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
		if errors.Is(err, ErrInvalidCategory) {
			response.BadRequest(c, "invalid category")
			return
		}
		h.log.Error("create news failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	defaultLang, _ := h.service.GetDefaultLanguage(c.Request.Context())
	prefLang := h.getPreferredLang(c, defaultLang)

	resp := NewsResponse{
		ID:            news.ID,
		PublisherType: news.PublisherType,
		PublisherID:   news.PublisherID,
		Category:      news.Category,
		Title:         ResolveTitle(news, prefLang, defaultLang),
		Body:          ResolveBody(news, prefLang, defaultLang),
		CoverImageID:  news.CoverImageID,
		IsPinned:      news.IsPinned,
		PublishAt:     news.PublishAt,
		ExpiresAt:     news.ExpiresAt,
		Status:        GetStatus(news, time.Now()),
		AuthorID:      news.AuthorID,
		CreatedAt:     news.CreatedAt,
	}
	response.Created(c, resp)
}

func (h *Handler) GetNews(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid news id")
		return
	}

	userID := middleware.GetUserID(c)

	news, attachments, err := h.service.GetNews(c.Request.Context(), newsID, true)
	if err != nil {
		if errors.Is(err, ErrNewsNotFound) {
			response.NotFound(c, "news not found")
			return
		}
		h.log.Error("get news failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	isAdmin := h.canPublish(c, news.PublisherType, news.PublisherID)
	isAuthor := news.AuthorID == userID

	if !isAdmin && !isAuthor && !CanView(&news.News, false, time.Now()) {
		response.NotFound(c, "news not found")
		return
	}

	defaultLang, _ := h.service.GetDefaultLanguage(c.Request.Context())
	prefLang := h.getPreferredLang(c, defaultLang)

	resp := ToNewsResponse(news, attachments, prefLang, defaultLang, time.Now())
	response.OK(c, resp)
}

func (h *Handler) GetTranslation(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid news id")
		return
	}

	lang := c.Query("lang")
	if lang == "" {
		lang = LangLocal
	}

	userID := middleware.GetUserID(c)

	existingNews, err := h.service.GetNewsByID(c.Request.Context(), newsID)
	if err != nil {
		h.log.Error("get news failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingNews == nil {
		response.NotFound(c, "news not found")
		return
	}

	isAdmin := h.canPublish(c, existingNews.PublisherType, existingNews.PublisherID)
	isAuthor := existingNews.AuthorID == userID

	title, body, err := h.service.GetTranslation(c.Request.Context(), newsID, lang, isAdmin || isAuthor)
	if err != nil {
		if errors.Is(err, ErrNewsNotFound) {
			response.NotFound(c, "news not found")
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

func (h *Handler) ListNews(c *gin.Context) {
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

	category := c.Query("category")

	isAdmin := h.canPublish(c, publisherType, publisherID)
	params := pagination.ParsePageParams(c)

	newsList, hasMore, err := h.service.ListNews(c.Request.Context(), publisherType, publisherID, category, isAdmin, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list news failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	newsIDs := make([]uuid.UUID, len(newsList))
	for i := range newsList {
		newsIDs[i] = newsList[i].ID
	}

	attachmentsMap, err := h.service.GetAttachmentsForNews(c.Request.Context(), newsIDs)
	if err != nil {
		h.log.Error("get attachments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	defaultLang, _ := h.service.GetDefaultLanguage(c.Request.Context())
	prefLang := h.getPreferredLang(c, defaultLang)

	now := time.Now()
	result := pagination.PageResult[NewsResponse]{
		Data:    ToNewsResponses(newsList, attachmentsMap, prefLang, defaultLang, now),
		HasMore: hasMore,
	}
	if hasMore && len(newsList) > 0 {
		last := newsList[len(newsList)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateNews(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid news id")
		return
	}

	var req UpdateNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	existingNews, err := h.service.GetNewsByID(c.Request.Context(), newsID)
	if err != nil {
		h.log.Error("get news failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingNews == nil {
		response.NotFound(c, "news not found")
		return
	}

	isAdmin := h.canPublish(c, existingNews.PublisherType, existingNews.PublisherID)

	news, err := h.service.UpdateNews(
		c.Request.Context(),
		newsID,
		userID,
		isAdmin,
		req.TitleEN,
		req.BodyEN,
		req.TitleLocal,
		req.BodyLocal,
		req.Category,
		req.CoverImageID,
		req.PublishAt,
		req.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, ErrNewsNotFound) {
			response.NotFound(c, "news not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to update this news")
			return
		}
		if errors.Is(err, ErrInvalidCategory) {
			response.BadRequest(c, "invalid category")
			return
		}
		h.log.Error("update news failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	defaultLang, _ := h.service.GetDefaultLanguage(c.Request.Context())
	prefLang := h.getPreferredLang(c, defaultLang)

	resp := NewsResponse{
		ID:            news.ID,
		PublisherType: news.PublisherType,
		PublisherID:   news.PublisherID,
		Category:      news.Category,
		Title:         ResolveTitle(news, prefLang, defaultLang),
		Body:          ResolveBody(news, prefLang, defaultLang),
		CoverImageID:  news.CoverImageID,
		IsPinned:      news.IsPinned,
		PublishAt:     news.PublishAt,
		ExpiresAt:     news.ExpiresAt,
		Status:        GetStatus(news, time.Now()),
		AuthorID:      news.AuthorID,
		CreatedAt:     news.CreatedAt,
		UpdatedAt:     news.UpdatedAt,
	}
	response.OK(c, resp)
}

func (h *Handler) DeleteNews(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid news id")
		return
	}

	userID := middleware.GetUserID(c)

	existingNews, err := h.service.GetNewsByID(c.Request.Context(), newsID)
	if err != nil {
		h.log.Error("get news failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingNews == nil {
		response.NotFound(c, "news not found")
		return
	}

	isAdmin := h.canPublish(c, existingNews.PublisherType, existingNews.PublisherID)

	if err := h.service.DeleteNews(c.Request.Context(), newsID, userID, isAdmin); err != nil {
		if errors.Is(err, ErrNewsNotFound) {
			response.NotFound(c, "news not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to delete this news")
			return
		}
		h.log.Error("delete news failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) PinNews(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid news id")
		return
	}

	var req PinNewsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	existingNews, err := h.service.GetNewsByID(c.Request.Context(), newsID)
	if err != nil {
		h.log.Error("get news failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingNews == nil {
		response.NotFound(c, "news not found")
		return
	}

	isAdmin := h.canPublish(c, existingNews.PublisherType, existingNews.PublisherID)

	if err := h.service.PinNews(c.Request.Context(), newsID, isAdmin, req.Pinned); err != nil {
		if errors.Is(err, ErrNewsNotFound) {
			response.NotFound(c, "news not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "only admins can pin news")
			return
		}
		h.log.Error("pin news failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) AddAttachment(c *gin.Context) {
	newsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid news id")
		return
	}

	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	userID := middleware.GetUserID(c)

	existingNews, err := h.service.GetNewsByID(c.Request.Context(), newsID)
	if err != nil {
		h.log.Error("get news failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingNews == nil {
		response.NotFound(c, "news not found")
		return
	}

	isAdmin := h.canPublish(c, existingNews.PublisherType, existingNews.PublisherID)

	attachment, err := h.service.AddAttachment(
		c.Request.Context(),
		newsID,
		userID,
		isAdmin,
		req.StoredFileID,
		req.DisplayName,
		req.FileType,
		req.OrderIndex,
	)
	if err != nil {
		if errors.Is(err, ErrNewsNotFound) {
			response.NotFound(c, "news not found")
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

	existingNews, err := h.service.GetNewsByID(c.Request.Context(), attachment.NewsID)
	if err != nil {
		h.log.Error("get news failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingNews == nil {
		response.NotFound(c, "news not found")
		return
	}

	isAdmin := h.canPublish(c, existingNews.PublisherType, existingNews.PublisherID)

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
		return permission.CanAdminUniversity(c)
	case PublisherCollege:
		if publisherID == nil {
			return false
		}
		return permission.CanAdminCollege(c, *publisherID)
	case PublisherDepartment:
		if publisherID == nil {
			return false
		}
		return permission.CanAdminDepartment(c, *publisherID)
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
