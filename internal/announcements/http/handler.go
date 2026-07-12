// Package http is the HTTP transport for the announcements domain.
package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/announcements"
	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// PinRequest toggles the pinned state of a post or an activity; both pin
// endpoints share this body.
type PinRequest struct {
	Pinned bool `json:"pinned"`
}

// Handler is the announcements context's HTTP surface: posts + activities.
// Posts and activities are authorised in the service by authorship and
// scope-visibility (like classroom teams — ownership is the law), so their
// routes carry no gate. Only creation consults the policy engine, and its
// scope lives in the request body, so the two create handlers authorise
// through the gates' in-handler checks (§18a).
type Handler struct {
	post  *announcements.PostService
	act   *announcements.ActivityService
	gates *authzhttp.Gates
	log   *zap.Logger
}

func NewHandler(post *announcements.PostService, act *announcements.ActivityService, gates *authzhttp.Gates, log *zap.Logger) *Handler {
	return &Handler{post: post, act: act, gates: gates, log: log}
}

// Routes maps every announcements route onto rg (already behind auth).
func (h *Handler) Routes(rg *gin.RouterGroup) {
	// Collection routes and member self-actions authorize in the handler or
	// the service (a comment or like is the caller's own act on a visible
	// post); everything addressed by the post's id goes through the gate.
	posts := rg.Group("/posts")
	posts.GET("", h.ListPosts)
	posts.POST("", h.CreatePost)
	posts.POST("/:id/comments", h.CreateComment)
	posts.POST("/:id/like", h.Like)
	posts.DELETE("/:id/like", h.Unlike)

	target := rg.Group("/posts/:id")
	h.gates.Post(target, authz.ResourcePost)
	target.GET("", h.GetPost)
	target.PUT("", h.UpdatePost)
	target.DELETE("", h.DeletePost)
	target.POST("", h.PostCustom) // :pin, :attach
	target.GET("/comments", h.ListComments)
	target.GET("/attachments/:attachmentId", h.DownloadPostAttachment)

	rg.DELETE("/post-attachments/:id", h.RemovePostAttachment)

	activities := rg.Group("/activities")
	activities.GET("", h.ListActivities)
	activities.POST("", h.CreateActivity)
	activities.GET("/:id", h.GetActivity)
	activities.GET("/:id/translation", h.GetTranslation)
	activities.PUT("/:id", h.UpdateActivity)
	activities.DELETE("/:id", h.DeleteActivity)
	activities.PUT("/:id/pin", h.PinActivity)
	activities.POST("/:id/attachments", h.AddActivityAttachment)
	activities.GET("/:id/attachments/:attachmentId", h.DownloadActivityAttachment)
	rg.DELETE("/activity-attachments/:id", h.RemoveActivityAttachment)
}

// respondError is the context's single error→status translation point (the
// second of the two sanctioned translation points): every endpoint funnels
// its service errors through here. Unknown errors are logged and surface as
// an opaque 500.
func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	// Not found. A deleted, expired, or not-yet-published post is hidden as
	// not-found so its existence never leaks to a reader who may not see it.
	case errors.Is(err, announcements.ErrPostNotFound),
		errors.Is(err, announcements.ErrPostDeleted),
		errors.Is(err, announcements.ErrPostExpired),
		errors.Is(err, announcements.ErrPostScheduled),
		errors.Is(err, announcements.ErrActivityNotFound),
		errors.Is(err, announcements.ErrAttachmentNotFound),
		errors.Is(err, announcements.ErrUploadNotFound),
		errors.Is(err, announcements.ErrTranslationMissing):
		response.NotFound(c, err.Error())

	// Conflicts: a lost optimistic-concurrency race, the like already exists,
	// or none exists to remove.
	case errors.Is(err, announcements.ErrConflict),
		errors.Is(err, announcements.ErrAlreadyLiked),
		errors.Is(err, announcements.ErrNotLiked):
		response.Conflict(c, err.Error())

	// Ownership and permission refusals.
	case errors.Is(err, announcements.ErrNotAuthorized),
		errors.Is(err, announcements.ErrUserMuted):
		response.Forbidden(c, err.Error())

	// Bad input or business-rule refusal.
	case errors.Is(err, announcements.ErrInvalidScope),
		errors.Is(err, announcements.ErrInvalidPublisher),
		errors.Is(err, announcements.ErrInvalidType),
		errors.Is(err, announcements.ErrInvalidLanguage),
		errors.Is(err, announcements.ErrCannotPinComment),
		errors.Is(err, announcements.ErrInvalidFileType),
		errors.Is(err, announcements.ErrFileTooLarge),
		errors.Is(err, pagination.ErrInvalidCursor):
		response.BadRequest(c, err.Error())

	default:
		h.log.Error("announcements handler error", zap.Error(err))
		response.InternalError(c)
	}
}
