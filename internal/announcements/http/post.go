package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/announcements"
	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// CreatePostRequest is the body of a new top-level post.
type CreatePostRequest struct {
	ScopeType string     `json:"scope_type" binding:"required,oneof=university college department program offering"`
	ScopeID   *uuid.UUID `json:"scope_id"`
	Body      string     `json:"body" binding:"required,min=1,max=10000"`
	PublishAt *time.Time `json:"publish_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// CreateCommentRequest is the body of a new reply.
type CreateCommentRequest struct {
	Body string `json:"body" binding:"required,min=1,max=5000"`
}

// UpdatePostRequest is a partial edit; nil fields are left unchanged and
// clear_schedule drops both schedule bounds.
type UpdatePostRequest struct {
	Body          *string    `json:"body" binding:"omitempty,min=1,max=10000"`
	PublishAt     *time.Time `json:"publish_at"`
	ExpiresAt     *time.Time `json:"expires_at"`
	ClearSchedule bool       `json:"clear_schedule"`
}

// PostResponse is a post (or comment) with its author, scope, counts, and the
// reader's like state.
type PostResponse struct {
	ID                   uuid.UUID               `json:"id"`
	ScopeType            announcements.ScopeType `json:"scope_type"`
	ScopeID              *uuid.UUID              `json:"scope_id,omitempty"`
	ScopeName            *string                 `json:"scope_name,omitempty"`
	ScopeNameLocal       *string                 `json:"scope_name_local,omitempty"`
	ParentID             *uuid.UUID              `json:"parent_id,omitempty"`
	RootID               *uuid.UUID              `json:"root_id,omitempty"`
	Body                 string                  `json:"body"`
	IsPinned             bool                    `json:"is_pinned"`
	PublishAt            *time.Time              `json:"publish_at,omitempty"`
	ExpiresAt            *time.Time              `json:"expires_at,omitempty"`
	Status               announcements.Status    `json:"status"`
	AuthorID             uuid.UUID               `json:"author_id"`
	AuthorName           string                  `json:"author_name"`
	AuthorNameLocal      *string                 `json:"author_name_local,omitempty"`
	AuthorAvatar         *string                 `json:"author_avatar,omitempty"`
	AuthorRoleTitle      *string                 `json:"author_role_title,omitempty"`
	AuthorRoleTitleLocal *string                 `json:"author_role_title_local,omitempty"`
	LikeCount            int                     `json:"like_count"`
	CommentCount         int                     `json:"comment_count"`
	IsLiked              bool                    `json:"is_liked"`
	Attachments          []AttachmentResponse    `json:"attachments,omitempty"`
	Mentions             []MentionResponse       `json:"mentions,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            *time.Time              `json:"updated_at,omitempty"`
}

// MentionResponse is one @mentioned user in a post.
type MentionResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	FullName string    `json:"full_name"`
}

func postResponse(p *announcements.PostView, attachments []announcements.PostAttachment, mentions []announcements.MentionedUser, isLiked bool, now time.Time) PostResponse {
	return PostResponse{
		ID:                   p.ID,
		ScopeType:            p.ScopeType,
		ScopeID:              p.ScopeID,
		ScopeName:            p.ScopeName,
		ScopeNameLocal:       p.ScopeNameLocal,
		ParentID:             p.ParentID,
		RootID:               p.RootID,
		Body:                 p.Body,
		IsPinned:             p.IsPinned,
		PublishAt:            p.PublishAt,
		ExpiresAt:            p.ExpiresAt,
		Status:               p.Status(now),
		AuthorID:             p.AuthorID,
		AuthorName:           p.AuthorName,
		AuthorNameLocal:      p.AuthorNameLocal,
		AuthorAvatar:         p.AuthorAvatar,
		AuthorRoleTitle:      p.AuthorRoleTitle,
		AuthorRoleTitleLocal: p.AuthorRoleTitleLocal,
		LikeCount:            p.LikeCount,
		CommentCount:         p.CommentCount,
		IsLiked:              isLiked,
		Attachments:          postAttachmentResponses(attachments),
		Mentions:             mentionResponses(mentions),
		CreatedAt:            p.CreatedAt,
		UpdatedAt:            p.UpdatedAt,
	}
}

func postResponses(posts []announcements.PostView, attachmentsMap map[uuid.UUID][]announcements.PostAttachment, mentionsMap map[uuid.UUID][]announcements.MentionedUser, likesMap map[uuid.UUID]bool, now time.Time) []PostResponse {
	result := make([]PostResponse, len(posts))
	for i := range posts {
		result[i] = postResponse(&posts[i], attachmentsMap[posts[i].ID], mentionsMap[posts[i].ID], likesMap[posts[i].ID], now)
	}
	return result
}

func mentionResponses(mentions []announcements.MentionedUser) []MentionResponse {
	if len(mentions) == 0 {
		return nil
	}
	result := make([]MentionResponse, len(mentions))
	for i, m := range mentions {
		result[i] = MentionResponse{UserID: m.UserID, Username: m.Username, FullName: m.FullName}
	}
	return result
}

// scopeAuthority reports whether the caller holds posting/moderation
// authority over a scope named in a request (create's body, list's query) —
// the routes a post-addressed gate cannot attribute. University-wide
// resolves by rank; college/department/program through the unit's lineage;
// an offering scope needs a teaching seat in it.
func (h *Handler) scopeAuthority(c *gin.Context, scopeType announcements.ScopeType, scopeID *uuid.UUID) bool {
	if scopeType == announcements.ScopeUniversity {
		return h.gates.CheckStaffAtLeast(c, authz.ResourcePost, authz.ActionCreate, authz.ScopeUniversity)
	}
	if scopeID == nil {
		return false
	}
	if scopeType == announcements.ScopeOffering {
		role := h.gates.Seat(c, *scopeID)
		return role == authz.OfferingRoleTeacher || role == authz.OfferingRoleAssistant
	}
	on, ok := postScopeEntity(scopeType)
	if !ok {
		return false
	}
	return h.gates.CheckStaffOn(c, authz.ResourcePost, authz.ActionCreate, on, *scopeID)
}

// postScopeEntity maps a post scope onto the authz entity whose lineage
// governs writing to it.
func postScopeEntity(s announcements.ScopeType) (authz.Entity, bool) {
	switch s {
	case announcements.ScopeCollege:
		return authz.ResourceCollege, true
	case announcements.ScopeDepartment:
		return authz.ResourceDepartment, true
	case announcements.ScopeProgram:
		return authz.ResourceProgram, true
	default:
		// University posts are checked by rank; offering posts by seat.
		return "", false
	}
}

func (h *Handler) CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	scopeType := announcements.ScopeType(req.ScopeType)
	authority := h.scopeAuthority(c, scopeType, req.ScopeID)

	if scopeType == announcements.ScopeOffering {
		if req.ScopeID == nil {
			response.BadRequest(c, "scope_id required for offering scope")
			return
		}
		if !authority {
			response.Forbidden(c, "only teaching staff can post in an offering")
			return
		}
	} else if !authority {
		canAccess, err := h.post.CanAccessScope(c.Request.Context(), userID, scopeType, req.ScopeID)
		if err != nil {
			h.respondError(c, err)
			return
		}
		if !canAccess {
			response.Forbidden(c, "no access to this scope")
			return
		}
	}

	post, err := h.post.CreatePost(c.Request.Context(), announcements.CreatePostInput{
		AuthorID:  userID,
		ScopeType: scopeType,
		ScopeID:   req.ScopeID,
		Body:      req.Body,
		PublishAt: req.PublishAt,
		ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	full, atts, ments, _, err := h.post.GetPost(c.Request.Context(), post.ID, userID, true)
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.Created(c, postResponse(full, atts, ments, false, time.Now()))
}

func (h *Handler) GetPost(c *gin.Context) {
	info := authzhttp.Access(c)
	post, attachments, mentions, isLiked, err := h.post.GetPost(c.Request.Context(),
		info.TargetID(), middleware.GetUserID(c), info.Authority())
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, postResponse(post, attachments, mentions, isLiked, time.Now()))
}

func (h *Handler) ListPosts(c *gin.Context) {
	scopeType := announcements.ScopeType(c.Query("scope_type"))
	if scopeType == "" {
		scopeType = announcements.ScopeUniversity
	}

	var scopeID *uuid.UUID
	if s := c.Query("scope_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid scope_id")
			return
		}
		scopeID = &id
	}

	userID := middleware.GetUserID(c)
	authority := h.scopeAuthority(c, scopeType, scopeID)

	if !authority {
		canAccess, err := h.post.CanAccessScope(c.Request.Context(), userID, scopeType, scopeID)
		if err != nil {
			h.respondError(c, err)
			return
		}
		if !canAccess {
			response.Forbidden(c, "no access to this scope")
			return
		}
	}

	params := pagination.ParsePageParams(c)
	posts, hasMore, err := h.post.ListPosts(c.Request.Context(), scopeType, scopeID, authority, params)
	if err != nil {
		h.respondError(c, err)
		return
	}

	ids := make([]uuid.UUID, len(posts))
	for i := range posts {
		ids[i] = posts[i].ID
	}

	attachmentsMap, err := h.post.AttachmentsFor(c.Request.Context(), ids)
	if err != nil {
		h.respondError(c, err)
		return
	}
	mentionsMap, err := h.post.MentionsFor(c.Request.Context(), ids)
	if err != nil {
		h.respondError(c, err)
		return
	}
	likesMap, err := h.post.UserLikesFor(c.Request.Context(), ids, userID)
	if err != nil {
		h.respondError(c, err)
		return
	}

	now := time.Now()
	result := pagination.PageResult[PostResponse]{
		Data:    postResponses(posts, attachmentsMap, mentionsMap, likesMap, now),
		HasMore: hasMore,
	}
	if hasMore && len(posts) > 0 {
		last := posts[len(posts)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	id := authzhttp.Access(c).TargetID()

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if _, err := h.post.UpdatePost(c.Request.Context(), announcements.UpdatePostInput{
		ID:            id,
		Body:          req.Body,
		PublishAt:     req.PublishAt,
		ExpiresAt:     req.ExpiresAt,
		ClearSchedule: req.ClearSchedule,
	}); err != nil {
		h.respondError(c, err)
		return
	}

	full, atts, ments, isLiked, err := h.post.GetPost(c.Request.Context(), id, middleware.GetUserID(c), true)
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.OK(c, postResponse(full, atts, ments, isLiked, time.Now()))
}

func (h *Handler) DeletePost(c *gin.Context) {
	if err := h.post.DeletePost(c.Request.Context(), authzhttp.Access(c).TargetID()); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) CreateComment(c *gin.Context) {
	parentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	comment, err := h.post.CreateComment(c.Request.Context(), announcements.CreateCommentInput{
		AuthorID: middleware.GetUserID(c),
		ParentID: parentID,
		Body:     req.Body,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.Created(c, PostResponse{
		ID:        comment.ID,
		ScopeType: comment.ScopeType,
		ScopeID:   comment.ScopeID,
		ParentID:  comment.ParentID,
		RootID:    comment.RootID,
		Body:      comment.Body,
		Status:    comment.Status(time.Now()),
		AuthorID:  comment.AuthorID,
		CreatedAt: comment.CreatedAt,
	})
}

func (h *Handler) ListComments(c *gin.Context) {
	rootID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	userID := middleware.GetUserID(c)
	rootPost, err := h.post.GetByID(c.Request.Context(), rootID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if rootPost == nil {
		response.NotFound(c, "post not found")
		return
	}

	if !authzhttp.Access(c).Authority() && !rootPost.Visible(time.Now()) {
		response.NotFound(c, "post not found")
		return
	}

	params := pagination.ParsePageParams(c)
	comments, hasMore, err := h.post.ListComments(c.Request.Context(), rootID, params)
	if err != nil {
		h.respondError(c, err)
		return
	}

	ids := make([]uuid.UUID, len(comments))
	for i := range comments {
		ids[i] = comments[i].ID
	}
	attachmentsMap, _ := h.post.AttachmentsFor(c.Request.Context(), ids)
	mentionsMap, _ := h.post.MentionsFor(c.Request.Context(), ids)
	likesMap, _ := h.post.UserLikesFor(c.Request.Context(), ids, userID)

	now := time.Now()
	result := pagination.PageResult[PostResponse]{
		Data:    postResponses(comments, attachmentsMap, mentionsMap, likesMap, now),
		HasMore: hasMore,
	}
	if hasMore && len(comments) > 0 {
		last := comments[len(comments)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) Like(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	if err := h.post.Like(c.Request.Context(), id, middleware.GetUserID(c)); err != nil {
		h.respondError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) Unlike(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	if err := h.post.Unlike(c.Request.Context(), id, middleware.GetUserID(c)); err != nil {
		h.respondError(c, err)
		return
	}

	response.NoContent(c)
}

// PostCustom dispatches POST /posts/:id — :pin, :attach.
func (h *Handler) PostCustom(c *gin.Context) {
	info := authzhttp.Access(c)
	switch info.Action() {
	case authz.ActionPin:
		h.pinPost(c, info.TargetID())
	case authz.ActionAttach:
		h.addPostAttachment(c, info.TargetID())
	default:
		response.NotFound(c, "unknown action")
	}
}

func (h *Handler) pinPost(c *gin.Context, id uuid.UUID) {
	var req PinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if err := h.post.Pin(c.Request.Context(), id, req.Pinned); err != nil {
		h.respondError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) addPostAttachment(c *gin.Context, id uuid.UUID) {
	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	att, err := h.post.AddAttachment(c.Request.Context(), announcements.AddPostAttachmentInput{
		PostID:      id,
		ActorID:     middleware.GetUserID(c),
		UploadID:    req.UploadID,
		DisplayName: req.DisplayName,
		FileType:    req.FileType,
		OrderIndex:  req.OrderIndex,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}

	response.Created(c, postAttachmentResponse(att))
}

func (h *Handler) RemovePostAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	att, err := h.post.AttachmentByID(c.Request.Context(), attachmentID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if att == nil {
		response.NotFound(c, "attachment not found")
		return
	}

	// Removing an attachment is editing the post; the post check decides.
	if !h.gates.CheckPost(c, authz.ActionUpdate, att.PostID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}
	if err := h.post.RemoveAttachment(c.Request.Context(), attachmentID); err != nil {
		h.respondError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *Handler) DownloadPostAttachment(c *gin.Context) {
	info := authzhttp.Access(c)
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.BadRequest(c, "invalid attachment id")
		return
	}

	url, err := h.post.PresignAttachment(c.Request.Context(), info.TargetID(), attachmentID, info.Authority())
	if err != nil {
		h.respondError(c, err)
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, url)
}
