package http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/announcements"
	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// CreateActivityRequest is the body of a new activity.
type CreateActivityRequest struct {
	PublisherType string     `json:"publisher_type" binding:"required,oneof=university college department"`
	PublisherID   *uuid.UUID `json:"publisher_id"`
	Type          string     `json:"type" binding:"required,oneof=news announcement webinar workshop conference symposium training_course"`
	TitleEN       *string    `json:"title_en" binding:"omitempty,min=1,max=255"`
	TitleLocal    *string    `json:"title_local" binding:"omitempty,max=255"`
	BodyEN        *string    `json:"body_en" binding:"omitempty,min=1"`
	BodyLocal     *string    `json:"body_local"`
	CoverUploadID *uuid.UUID `json:"cover_upload_id"`
	PublishAt     *time.Time `json:"publish_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

// Validate enforces that each language provided is complete (title and body
// together) and that at least one language is present.
func (r *CreateActivityRequest) Validate() error {
	enTitle := r.TitleEN != nil && strings.TrimSpace(*r.TitleEN) != ""
	enBody := r.BodyEN != nil && strings.TrimSpace(*r.BodyEN) != ""
	localTitle := r.TitleLocal != nil && strings.TrimSpace(*r.TitleLocal) != ""
	localBody := r.BodyLocal != nil && strings.TrimSpace(*r.BodyLocal) != ""

	if enTitle != enBody {
		return errors.New("english title and body must both be provided")
	}
	if localTitle != localBody {
		return errors.New("kurdish title and body must both be provided")
	}
	if !enTitle && !localTitle {
		return errors.New("provide title and body in at least one language")
	}
	return nil
}

// UpdateActivityRequest is a partial edit of an activity.
type UpdateActivityRequest struct {
	TitleEN       *string    `json:"title_en" binding:"omitempty,min=1,max=255"`
	TitleLocal    *string    `json:"title_local" binding:"omitempty,max=255"`
	BodyEN        *string    `json:"body_en" binding:"omitempty,min=1"`
	BodyLocal     *string    `json:"body_local"`
	Type          *string    `json:"type" binding:"omitempty,oneof=news announcement webinar workshop conference symposium training_course"`
	CoverUploadID *uuid.UUID `json:"cover_upload_id"`
	PublishAt     *time.Time `json:"publish_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

// ActivityResponse is an activity with its author and attachments.
type ActivityResponse struct {
	ID              uuid.UUID                   `json:"id"`
	PublisherType   announcements.PublisherType `json:"publisher_type"`
	PublisherID     *uuid.UUID                  `json:"publisher_id,omitempty"`
	Type            announcements.ActivityType  `json:"type"`
	TitleEN         string                      `json:"title_en"`
	TitleLocal      *string                     `json:"title_local,omitempty"`
	BodyEN          string                      `json:"body_en"`
	BodyLocal       *string                     `json:"body_local,omitempty"`
	CoverImageID    *uuid.UUID                  `json:"cover_image_id,omitempty"`
	IsPinned        bool                        `json:"is_pinned"`
	PublishAt       *time.Time                  `json:"publish_at,omitempty"`
	ExpiresAt       *time.Time                  `json:"expires_at,omitempty"`
	Status          announcements.Status        `json:"status"`
	AuthorID        uuid.UUID                   `json:"author_id"`
	AuthorName      string                      `json:"author_name"`
	AuthorNameLocal *string                     `json:"author_name_local,omitempty"`
	AuthorAvatar    *string                     `json:"author_avatar,omitempty"`
	Attachments     []AttachmentResponse        `json:"attachments,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       *time.Time                  `json:"updated_at,omitempty"`
}

// TranslationResponse is one activity's title and body in a single language.
type TranslationResponse struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

func baseActivityResponse(a *announcements.Activity, now time.Time) ActivityResponse {
	return ActivityResponse{
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
		Status:        a.Status(now),
		AuthorID:      a.AuthorID,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

func activityResponse(a *announcements.Activity, now time.Time) ActivityResponse {
	return baseActivityResponse(a, now)
}

func activityWithAuthorResponse(a *announcements.ActivityWithAuthor, attachments []announcements.ActivityAttachment, now time.Time) ActivityResponse {
	resp := baseActivityResponse(&a.Activity, now)
	resp.AuthorName = a.AuthorName
	resp.AuthorNameLocal = a.AuthorNameLocal
	resp.AuthorAvatar = a.AuthorAvatar
	resp.Attachments = activityAttachmentResponses(attachments)
	return resp
}

func activityResponses(activities []announcements.ActivityWithAuthor, attachmentsMap map[uuid.UUID][]announcements.ActivityAttachment, now time.Time) []ActivityResponse {
	result := make([]ActivityResponse, len(activities))
	for i := range activities {
		result[i] = activityWithAuthorResponse(&activities[i], attachmentsMap[activities[i].ID], now)
	}
	return result
}

// canPublish reports whether the caller may publish an activity from the
// given publisher; the policy decides the bar. University-wide resolves by
// rank; a college or department activity authorises through that org entity's
// lineage.
func (h *Handler) canPublish(c *gin.Context, pt announcements.PublisherType, publisherID *uuid.UUID) bool {
	if pt == announcements.PublisherUniversity {
		return h.gates.CheckStaffAtLeast(c, authz.ResourceActivity, authz.ActionCreate, authz.ScopeUniversity)
	}
	if publisherID == nil {
		return false
	}
	on, ok := publisherEntity(pt)
	if !ok {
		return false
	}
	return h.gates.CheckStaffOn(c, authz.ResourceActivity, authz.ActionCreate, on, *publisherID)
}

// publisherEntity maps an activity publisher onto the authz entity whose
// lineage governs publishing under it.
func publisherEntity(pt announcements.PublisherType) (authz.Entity, bool) {
	switch pt {
	case announcements.PublisherCollege:
		return authz.ResourceCollege, true
	case announcements.PublisherDepartment:
		return authz.ResourceDepartment, true
	default:
		// University publishing has no lineage entity; it is checked by rank.
		return "", false
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

	pt := announcements.PublisherType(req.PublisherType)
	if !h.canPublish(c, pt, req.PublisherID) {
		response.Forbidden(c, "not authorized to publish activity")
		return
	}

	titleEN, bodyEN := "", ""
	if req.TitleEN != nil {
		titleEN = *req.TitleEN
	}
	if req.BodyEN != nil {
		bodyEN = *req.BodyEN
	}

	a, err := h.act.Create(c.Request.Context(), announcements.CreateActivityInput{
		AuthorID:      middleware.GetUserID(c),
		PublisherType: pt,
		PublisherID:   req.PublisherID,
		Type:          announcements.ActivityType(req.Type),
		TitleEN:       titleEN,
		TitleLocal:    req.TitleLocal,
		BodyEN:        bodyEN,
		BodyLocal:     req.BodyLocal,
		CoverUploadID: req.CoverUploadID,
		PublishAt:     req.PublishAt,
		ExpiresAt:     req.ExpiresAt,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.Created(c, activityResponse(a, time.Now()))
}

func (h *Handler) GetActivity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	userID := middleware.GetUserID(c)
	a, attachments, err := h.act.Get(c.Request.Context(), id, true)
	if err != nil {
		h.respondError(c, err)
		return
	}

	authority := h.canPublish(c, a.PublisherType, a.PublisherID)
	if !authority && a.AuthorID != userID && !a.CanView(false, time.Now()) {
		response.NotFound(c, "activity not found")
		return
	}

	response.OK(c, activityWithAuthorResponse(a, attachments, time.Now()))
}

func (h *Handler) GetTranslation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	lang := announcements.Lang(c.Query("lang"))
	if lang == "" {
		lang = announcements.LangLocal
	}

	userID := middleware.GetUserID(c)
	existing, err := h.act.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	authority := h.canPublish(c, existing.PublisherType, existing.PublisherID)

	title, body, err := h.act.Translate(c.Request.Context(), id, lang, authority || existing.AuthorID == userID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.OK(c, TranslationResponse{Title: title, Body: body})
}

func (h *Handler) ListActivities(c *gin.Context) {
	pt := announcements.PublisherType(c.Query("publisher_type"))
	if pt == "" {
		pt = announcements.PublisherUniversity
	}

	var publisherID *uuid.UUID
	if s := c.Query("publisher_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid publisher_id")
			return
		}
		publisherID = &id
	}

	activityType := announcements.ActivityType(c.Query("type"))
	authority := h.canPublish(c, pt, publisherID)
	params := pagination.ParsePageParams(c)

	activities, hasMore, err := h.act.List(c.Request.Context(), pt, publisherID, activityType, authority, params)
	if err != nil {
		h.respondError(c, err)
		return
	}

	ids := make([]uuid.UUID, len(activities))
	for i := range activities {
		ids[i] = activities[i].ID
	}
	attachmentsMap, err := h.act.AttachmentsFor(c.Request.Context(), ids)
	if err != nil {
		h.respondError(c, err)
		return
	}

	now := time.Now()
	result := pagination.PageResult[ActivityResponse]{
		Data:    activityResponses(activities, attachmentsMap, now),
		HasMore: hasMore,
	}
	if hasMore && len(activities) > 0 {
		last := activities[len(activities)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateActivity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	var req UpdateActivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	existing, err := h.act.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	userID := middleware.GetUserID(c)
	if existing.AuthorID != userID && !h.canPublish(c, existing.PublisherType, existing.PublisherID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	var activityType *announcements.ActivityType
	if req.Type != nil {
		t := announcements.ActivityType(*req.Type)
		activityType = &t
	}

	a, err := h.act.Update(c.Request.Context(), announcements.UpdateActivityInput{
		ID:            id,
		ActorID:       userID,
		TitleEN:       req.TitleEN,
		TitleLocal:    req.TitleLocal,
		BodyEN:        req.BodyEN,
		BodyLocal:     req.BodyLocal,
		Type:          activityType,
		CoverUploadID: req.CoverUploadID,
		PublishAt:     req.PublishAt,
		ExpiresAt:     req.ExpiresAt,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.OK(c, activityResponse(a, time.Now()))
}

func (h *Handler) DeleteActivity(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	existing, err := h.act.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	if existing.AuthorID != middleware.GetUserID(c) && !h.canPublish(c, existing.PublisherType, existing.PublisherID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}
	if err := h.act.Delete(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}

	response.NoContent(c)
}

// ActivityCustom dispatches POST /activities/:id — :pin, :attach. Activities
// carry no gate (authorization is in-handler per publisher unit), so the
// colon suffix is split here instead of by a gate's attribution.
func (h *Handler) ActivityCustom(c *gin.Context) {
	rawID, action, hasAction := strings.Cut(c.Param("id"), ":")
	if !hasAction {
		response.NotFound(c, "unknown action")
		return
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}

	switch authz.Action(action) {
	case authz.ActionPin:
		h.pinActivity(c, id)
	case authz.ActionAttach:
		h.addActivityAttachment(c, id)
	default:
		response.NotFound(c, "unknown action")
	}
}

func (h *Handler) pinActivity(c *gin.Context, id uuid.UUID) {
	var req PinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	existing, err := h.act.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	if !h.canPublish(c, existing.PublisherType, existing.PublisherID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}
	if err := h.act.Pin(c.Request.Context(), id, req.Pinned); err != nil {
		h.respondError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) addActivityAttachment(c *gin.Context, id uuid.UUID) {
	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	existing, err := h.act.GetByID(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	userID := middleware.GetUserID(c)
	if existing.AuthorID != userID && !h.canPublish(c, existing.PublisherType, existing.PublisherID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	att, err := h.act.AddAttachment(c.Request.Context(), announcements.AddActivityAttachmentInput{
		ActivityID:  id,
		ActorID:     userID,
		UploadID:    req.UploadID,
		DisplayName: req.DisplayName,
		FileType:    req.FileType,
		OrderIndex:  req.OrderIndex,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.Created(c, AttachmentResponse{
		ID: att.ID, DisplayName: att.DisplayName, FileType: att.FileType, OrderIndex: att.OrderIndex,
	})
}

func (h *Handler) RemoveActivityAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	att, err := h.act.AttachmentByID(c.Request.Context(), attachmentID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if att == nil {
		response.NotFound(c, "attachment not found")
		return
	}

	existing, err := h.act.GetByID(c.Request.Context(), att.ActivityID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	if existing.AuthorID != middleware.GetUserID(c) && !h.canPublish(c, existing.PublisherType, existing.PublisherID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}
	if err := h.act.RemoveAttachment(c.Request.Context(), attachmentID); err != nil {
		h.respondError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) DownloadActivityAttachment(c *gin.Context) {
	activityID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid activity id")
		return
	}
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	existing, err := h.act.GetByID(c.Request.Context(), activityID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if existing == nil {
		response.NotFound(c, "activity not found")
		return
	}

	authority := h.canPublish(c, existing.PublisherType, existing.PublisherID)
	url, err := h.act.PresignAttachment(c.Request.Context(), activityID, attachmentID,
		authority || existing.AuthorID == middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}
