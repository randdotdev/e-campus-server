package post

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
	posts := rg.Group("/posts")
	posts.Use(auth)
	{
		posts.GET("", h.ListPosts)
		posts.POST("", h.CreatePost)
		posts.GET("/:id", h.GetPost)
		posts.PUT("/:id", h.UpdatePost)
		posts.DELETE("/:id", h.DeletePost)

		posts.GET("/:id/comments", h.ListComments)
		posts.POST("/:id/comments", h.CreateComment)

		posts.POST("/:id/like", h.LikePost)
		posts.DELETE("/:id/like", h.UnlikePost)

		posts.POST("/:id/attachments", h.AddAttachment)
		posts.PUT("/:id/pin", h.PinPost)
	}

	attachments := rg.Group("/post-attachments")
	attachments.Use(auth)
	{
		attachments.DELETE("/:id", h.RemoveAttachment)
	}
}

func (h *Handler) CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	isAdmin := h.isAdminForScope(c, req.ScopeType, req.ScopeID)

	if req.ScopeType == ScopeCourse {
		if req.ScopeID == nil {
			response.BadRequest(c, "scope_id required for course scope")
			return
		}
		if !isAdmin {
			response.Forbidden(c, "only teaching staff can post in a course")
			return
		}
	} else if !isAdmin {
		canAccess, err := h.service.CanAccessScope(c.Request.Context(), userID, req.ScopeType, req.ScopeID)
		if err != nil {
			h.log.Error("check scope access failed", zap.Error(err))
			response.InternalError(c)
			return
		}
		if !canAccess {
			response.Forbidden(c, "no access to this scope")
			return
		}
	}

	post, err := h.service.CreatePost(c.Request.Context(), userID, req.ScopeType, req.ScopeID, req.Body, req.PublishAt, req.ExpiresAt)
	if err != nil {
		if errors.Is(err, ErrInvalidScope) {
			response.BadRequest(c, err.Error())
			return
		}
		h.log.Error("create post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	// Fetch the full post with author details so the response mirrors what ListPosts returns.
	full, atts, ments, _, err := h.service.GetPost(c.Request.Context(), post.ID, userID, true)
	if err != nil {
		h.log.Error("fetch created post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToPostResponse(full, atts, ments, false, time.Now()))
}

func (h *Handler) GetPost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	userID := middleware.GetUserID(c)

	// Get post first to check scope
	post, attachments, mentions, isLiked, err := h.service.GetPost(c.Request.Context(), postID, userID, true)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.NotFound(c, "post not found")
			return
		}
		if errors.Is(err, ErrPostDeleted) || errors.Is(err, ErrPostExpired) || errors.Is(err, ErrPostScheduled) {
			response.NotFound(c, "post not found")
			return
		}
		h.log.Error("get post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	isAdmin := h.isAdminForScope(c, post.ScopeType, post.ScopeID)
	isAuthor := post.AuthorID == userID

	// Re-check visibility if not admin or author
	if !isAdmin && !isAuthor && !CanView(&post.Post, false, time.Now()) {
		response.NotFound(c, "post not found")
		return
	}

	resp := ToPostResponse(post, attachments, mentions, isLiked, time.Now())
	response.OK(c, resp)
}

func (h *Handler) ListPosts(c *gin.Context) {
	scopeType := c.Query("scope_type")
	if scopeType == "" {
		scopeType = ScopeUniversity
	}

	var scopeID *uuid.UUID
	if scopeIDStr := c.Query("scope_id"); scopeIDStr != "" {
		id, err := uuid.Parse(scopeIDStr)
		if err != nil {
			response.BadRequest(c, "invalid scope_id")
			return
		}
		scopeID = &id
	}

	userID := middleware.GetUserID(c)
	isAdmin := h.isAdminForScope(c, scopeType, scopeID)

	// Check access
	if !isAdmin {
		canAccess, err := h.service.CanAccessScope(c.Request.Context(), userID, scopeType, scopeID)
		if err != nil {
			h.log.Error("check scope access failed", zap.Error(err))
			response.InternalError(c)
			return
		}
		if !canAccess {
			response.Forbidden(c, "no access to this scope")
			return
		}
	}

	params := pagination.ParsePageParams(c)
	posts, hasMore, err := h.service.ListPosts(c.Request.Context(), scopeType, scopeID, isAdmin, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list posts failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	// Collect post IDs
	postIDs := make([]uuid.UUID, len(posts))
	for i := range posts {
		postIDs[i] = posts[i].ID
	}

	// Batch fetch attachments, mentions, and likes
	attachmentsMap, err := h.service.GetAttachmentsForPosts(c.Request.Context(), postIDs)
	if err != nil {
		h.log.Error("get attachments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	mentionsMap, err := h.service.GetMentionsForPosts(c.Request.Context(), postIDs)
	if err != nil {
		h.log.Error("get mentions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	likesMap, err := h.service.GetUserLikesForPosts(c.Request.Context(), postIDs, userID)
	if err != nil {
		h.log.Error("get likes failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	now := time.Now()
	result := pagination.PageResult[PostResponse]{
		Data:    ToPostResponses(posts, attachmentsMap, mentionsMap, likesMap, now),
		HasMore: hasMore,
	}
	if hasMore && len(posts) > 0 {
		last := posts[len(posts)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdatePost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	// Get post to check scope for admin status
	existingPost, err := h.service.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		h.log.Error("get post failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingPost == nil {
		response.NotFound(c, "post not found")
		return
	}

	isAdmin := h.isAdminForScope(c, existingPost.ScopeType, existingPost.ScopeID)

	_, err = h.service.UpdatePost(c.Request.Context(), postID, userID, isAdmin, req.Body, req.PublishAt, req.ExpiresAt, req.ClearSchedule)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.NotFound(c, "post not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to update this post")
			return
		}
		h.log.Error("update post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	// Fetch the full post with author details so the response mirrors what ListPosts returns.
	full, atts, ments, isLiked, err := h.service.GetPost(c.Request.Context(), postID, userID, isAdmin)
	if err != nil {
		h.log.Error("fetch updated post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToPostResponse(full, atts, ments, isLiked, time.Now()))
}

func (h *Handler) DeletePost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	userID := middleware.GetUserID(c)

	// Get post to check scope for admin status
	existingPost, err := h.service.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		h.log.Error("get post failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingPost == nil {
		response.NotFound(c, "post not found")
		return
	}

	isAdmin := h.isAdminForScope(c, existingPost.ScopeType, existingPost.ScopeID)

	if err := h.service.DeletePost(c.Request.Context(), postID, userID, isAdmin); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.NotFound(c, "post not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "not authorized to delete this post")
			return
		}
		h.log.Error("delete post failed", zap.Error(err))
		response.InternalError(c)
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

	userID := middleware.GetUserID(c)

	comment, err := h.service.CreateComment(c.Request.Context(), userID, parentID, req.Body)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.NotFound(c, "post not found")
			return
		}
		if errors.Is(err, ErrCannotCommentOnComment) {
			response.BadRequest(c, "cannot comment on deleted post")
			return
		}
		if errors.Is(err, ErrUserMuted) {
			response.Forbidden(c, "user is muted")
			return
		}
		h.log.Error("create comment failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	resp := PostResponse{
		ID:        comment.ID,
		ScopeType: comment.ScopeType,
		ScopeID:   comment.ScopeID,
		ParentID:  comment.ParentID,
		RootID:    comment.RootID,
		Body:      comment.Body,
		AuthorID:  comment.AuthorID,
		CreatedAt: comment.CreatedAt,
	}
	response.Created(c, resp)
}

func (h *Handler) ListComments(c *gin.Context) {
	rootID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	userID := middleware.GetUserID(c)

	rootPost, err := h.service.GetPostByID(c.Request.Context(), rootID)
	if err != nil {
		h.log.Error("get root post failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if rootPost == nil {
		response.NotFound(c, "post not found")
		return
	}

	isAdmin := h.isAdminForScope(c, rootPost.ScopeType, rootPost.ScopeID)
	isAuthor := rootPost.AuthorID == userID

	if !isAdmin && !isAuthor && !IsVisible(rootPost, time.Now()) {
		response.NotFound(c, "post not found")
		return
	}

	params := pagination.ParsePageParams(c)

	comments, hasMore, err := h.service.ListComments(c.Request.Context(), rootID, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list comments failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	// Collect comment IDs
	commentIDs := make([]uuid.UUID, len(comments))
	for i := range comments {
		commentIDs[i] = comments[i].ID
	}

	// Batch fetch
	attachmentsMap, _ := h.service.GetAttachmentsForPosts(c.Request.Context(), commentIDs)
	mentionsMap, _ := h.service.GetMentionsForPosts(c.Request.Context(), commentIDs)
	likesMap, _ := h.service.GetUserLikesForPosts(c.Request.Context(), commentIDs, userID)

	now := time.Now()
	result := pagination.PageResult[PostResponse]{
		Data:    ToPostResponses(comments, attachmentsMap, mentionsMap, likesMap, now),
		HasMore: hasMore,
	}
	if hasMore && len(comments) > 0 {
		last := comments[len(comments)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) LikePost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.service.LikePost(c.Request.Context(), postID, userID); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.NotFound(c, "post not found")
			return
		}
		if errors.Is(err, ErrAlreadyLiked) {
			response.Conflict(c, "already liked")
			return
		}
		if errors.Is(err, ErrUserMuted) {
			response.Forbidden(c, "user is muted")
			return
		}
		h.log.Error("like post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) UnlikePost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.service.UnlikePost(c.Request.Context(), postID, userID); err != nil {
		if errors.Is(err, ErrNotLiked) {
			response.BadRequest(c, "not liked")
			return
		}
		h.log.Error("unlike post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) AddAttachment(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	var req AddAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)

	// Get post to check scope for admin status
	existingPost, err := h.service.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		h.log.Error("get post failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingPost == nil {
		response.NotFound(c, "post not found")
		return
	}

	isAdmin := h.isAdminForScope(c, existingPost.ScopeType, existingPost.ScopeID)

	attachment, err := h.service.AddAttachment(
		c.Request.Context(),
		postID,
		userID,
		isAdmin,
		req.StoredFileID,
		req.DisplayName,
		req.FileType,
		req.SizeBytes,
		req.OrderIndex,
	)
	if err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.NotFound(c, "post not found")
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

	// Get attachment to get post
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

	// Get post to check scope for admin status
	existingPost, err := h.service.GetPostByID(c.Request.Context(), attachment.PostID)
	if err != nil {
		h.log.Error("get post failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingPost == nil {
		response.NotFound(c, "post not found")
		return
	}

	isAdmin := h.isAdminForScope(c, existingPost.ScopeType, existingPost.ScopeID)

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

func (h *Handler) PinPost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid post id")
		return
	}

	var req PinPostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	// Get post to check scope for admin status
	existingPost, err := h.service.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		h.log.Error("get post failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if existingPost == nil {
		response.NotFound(c, "post not found")
		return
	}

	isAdmin := h.isAdminForScope(c, existingPost.ScopeType, existingPost.ScopeID)

	if err := h.service.PinPost(c.Request.Context(), postID, isAdmin, req.Pinned); err != nil {
		if errors.Is(err, ErrPostNotFound) {
			response.NotFound(c, "post not found")
			return
		}
		if errors.Is(err, ErrNotAuthorized) {
			response.Forbidden(c, "only admins can pin posts")
			return
		}
		if errors.Is(err, ErrCannotPinComment) {
			response.BadRequest(c, "cannot pin comments")
			return
		}
		h.log.Error("pin post failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

// isAdminForScope checks if current user is admin for the given scope.
func (h *Handler) isAdminForScope(c *gin.Context, scopeType string, scopeID *uuid.UUID) bool {
	switch scopeType {
	case ScopeUniversity:
		return authz.Check(c, authz.ResourcePost, authz.ActionCreate)
	case ScopeCollege:
		if scopeID == nil {
			return false
		}
		return authz.Check(c, authz.ResourcePost, authz.ActionCreate, *scopeID)
	case ScopeDepartment:
		if scopeID == nil {
			return false
		}
		return authz.Check(c, authz.ResourcePost, authz.ActionCreate, *scopeID)
	case ScopeProgram:
		if scopeID == nil {
			return false
		}
		return authz.Check(c, authz.ResourcePost, authz.ActionCreate, *scopeID)
	case ScopeCourse:
		if scopeID == nil {
			return false
		}
		role := authz.CourseRole(c, *scopeID)
		return role == authz.CourseRoleTeacher ||
			role == authz.CourseRoleAssistant
	}
	return false
}
