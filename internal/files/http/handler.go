// Package http is the files context's HTTP surface: the upload endpoint
// and the single error→status translation point.
package http

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/files"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type Handler struct {
	inodes *files.InodeService
	log    *zap.Logger
}

func NewHandler(inodes *files.InodeService, log *zap.Logger) *Handler {
	return &Handler{inodes: inodes, log: log}
}

// Routes maps the one files endpoint. Uploading needs no gate: any
// authenticated user may bring bytes in (size-limited); what the bytes may
// be attached to is each consumer context's decision.
func (h *Handler) Routes(protected *gin.RouterGroup) {
	protected.POST("/uploads", h.Upload)
}

// UploadResponse is the receipt the client attaches with.
type UploadResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
}

// Upload accepts one multipart file and returns its receipt.
func (h *Handler) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "missing file")
		return
	}
	src, err := fh.Open()
	if err != nil {
		response.BadRequest(c, "unreadable file")
		return
	}
	defer func() { _ = src.Close() }()

	mimeType := fh.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	up, err := h.inodes.Upload(c.Request.Context(), middleware.GetUserID(c), fh.Filename, fh.Size, mimeType, src)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, UploadResponse{
		ID: up.ID, Name: up.Name, SizeBytes: fh.Size, MimeType: mimeType, CreatedAt: up.CreatedAt,
	})
}

// respondError is the context's single error→status translation point.
func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, files.ErrUploadNotFound),
		errors.Is(err, files.ErrInodeNotFound):
		response.NotFound(c, err.Error())

	case errors.Is(err, files.ErrNameInvalid):
		response.BadRequest(c, err.Error())

	case errors.Is(err, files.ErrFileTooLarge):
		response.Err(c, 413, "FILE_TOO_LARGE", err.Error())

	case errors.Is(err, files.ErrFileGone):
		response.Conflict(c, err.Error())

	default:
		h.log.Error("files: unhandled error", zap.Error(err))
		response.InternalError(c)
	}
}
