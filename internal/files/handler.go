package files

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// Folder handlers

func (h *Handler) CreateFolder(c *gin.Context) {
	var req CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	folder, err := h.service.CreateFolder(c.Request.Context(), userID, req.Name, req.ParentID)
	if errors.Is(err, ErrFolderNotFound) {
		response.NotFound(c, "parent folder not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "not owner of parent folder")
	} else if errors.Is(err, ErrDuplicateFolderName) {
		response.Conflict(c, "folder name already exists")
	} else if err != nil {
		h.log.Error("create folder failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToFolderResponse(folder))
	}
}

func (h *Handler) GetFolder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid folder id")
		return
	}

	userID := middleware.GetUserID(c)
	folder, err := h.service.GetFolder(c.Request.Context(), id, userID)
	if errors.Is(err, ErrFolderNotFound) {
		response.NotFound(c, "folder not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "access denied")
	} else if err != nil {
		h.log.Error("get folder failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToFolderResponse(folder))
	}
}

func (h *Handler) ListFolders(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	userID := middleware.GetUserID(c)

	var parentID *uuid.UUID
	if pid := c.Query("parent_id"); pid != "" {
		id, err := uuid.Parse(pid)
		if err != nil {
			response.BadRequest(c, "invalid parent_id")
			return
		}
		parentID = &id
	}

	folders, hasMore, err := h.service.ListFolders(c.Request.Context(), userID, parentID, params)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list folders failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[FolderResponse]{
		Data:    ToFoldersResponse(folders),
		HasMore: hasMore,
	}
	if hasMore && len(folders) > 0 {
		last := folders[len(folders)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateFolder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid folder id")
		return
	}

	var req UpdateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	folder, err := h.service.UpdateFolder(c.Request.Context(), id, userID, req.Name, req.ParentID)
	if errors.Is(err, ErrFolderNotFound) {
		response.NotFound(c, "folder not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "access denied")
	} else if errors.Is(err, ErrDuplicateFolderName) {
		response.Conflict(c, "folder name already exists")
	} else if err != nil {
		h.log.Error("update folder failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToFolderResponse(folder))
	}
}

func (h *Handler) DeleteFolder(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid folder id")
		return
	}

	userID := middleware.GetUserID(c)
	err = h.service.DeleteFolder(c.Request.Context(), id, userID)
	if errors.Is(err, ErrFolderNotFound) {
		response.NotFound(c, "folder not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "access denied")
	} else if errors.Is(err, ErrFolderNotEmpty) {
		response.BadRequest(c, "folder is not empty")
	} else if err != nil {
		h.log.Error("delete folder failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

// File handlers

func (h *Handler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "file is required")
		return
	}
	defer func() { _ = file.Close() }()

	var folderID *uuid.UUID
	if fid := c.PostForm("folder_id"); fid != "" {
		id, err := uuid.Parse(fid)
		if err != nil {
			response.BadRequest(c, "invalid folder_id")
			return
		}
		folderID = &id
	}

	userID := middleware.GetUserID(c)
	f, err := h.service.UploadFile(c.Request.Context(), userID, header.Filename, folderID, file, header.Size)
	if errors.Is(err, ErrFileTooLarge) {
		response.BadRequest(c, "file exceeds size limit")
	} else if errors.Is(err, ErrStorageQuotaExceeded) {
		response.BadRequest(c, "storage quota exceeded")
	} else if errors.Is(err, ErrInvalidFileType) {
		response.BadRequest(c, "file type not allowed")
	} else if errors.Is(err, ErrFolderNotFound) {
		response.NotFound(c, "folder not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "not owner of folder")
	} else if err != nil {
		h.log.Error("upload file failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToFileResponse(f))
	}
}

func (h *Handler) GetFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	userID := middleware.GetUserID(c)
	file, err := h.service.GetFile(c.Request.Context(), id, userID)
	if errors.Is(err, ErrFileNotFound) {
		response.NotFound(c, "file not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "access denied")
	} else if err != nil {
		h.log.Error("get file failed", zap.Error(err))
		response.InternalError(c)
	} else {
		url, _ := h.service.GetFileURL(c.Request.Context(), id, userID)
		response.OK(c, ToFileWithURLResponse(file, url))
	}
}

func (h *Handler) ListFiles(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	userID := middleware.GetUserID(c)

	var folderID *uuid.UUID
	if fid := c.Query("folder_id"); fid != "" {
		id, err := uuid.Parse(fid)
		if err != nil {
			response.BadRequest(c, "invalid folder_id")
			return
		}
		folderID = &id
	}

	files, hasMore, err := h.service.ListFiles(c.Request.Context(), userID, folderID, params)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list files failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[FileResponse]{
		Data:    ToFilesResponse(files),
		HasMore: hasMore,
	}
	if hasMore && len(files) > 0 {
		last := files[len(files)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UpdateFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	var req UpdateFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	file, err := h.service.UpdateFile(c.Request.Context(), id, userID, req.Name, req.FolderID)
	if errors.Is(err, ErrFileNotFound) {
		response.NotFound(c, "file not found")
	} else if errors.Is(err, ErrFolderNotFound) {
		response.NotFound(c, "folder not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "access denied")
	} else if err != nil {
		h.log.Error("update file failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, ToFileResponse(file))
	}
}

func (h *Handler) DeleteFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	userID := middleware.GetUserID(c)
	err = h.service.DeleteFile(c.Request.Context(), id, userID)
	if errors.Is(err, ErrFileNotFound) {
		response.NotFound(c, "file not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "access denied")
	} else if err != nil {
		h.log.Error("delete file failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) CopyFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	// Get stored_file_id from the source file
	userID := middleware.GetUserID(c)
	sourceFile, err := h.service.repo.GetUserFileByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("get source file failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if sourceFile == nil {
		response.NotFound(c, "file not found")
		return
	}

	var req CopyFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	file, err := h.service.CopyToStorage(c.Request.Context(), sourceFile.StoredFileID, userID, req.Name, req.FolderID)
	if errors.Is(err, ErrStoredFileNotFound) {
		response.NotFound(c, "file not found")
	} else if errors.Is(err, ErrStorageQuotaExceeded) {
		response.BadRequest(c, "storage quota exceeded")
	} else if errors.Is(err, ErrFolderNotFound) {
		response.NotFound(c, "folder not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "not owner of folder")
	} else if err != nil {
		h.log.Error("copy file failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToFileResponse(file))
	}
}

func (h *Handler) GetStorageUsage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	usage, err := h.service.GetStorageUsage(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get storage usage failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, usage)
}

// ServeFile redirects to a presigned MinIO URL for a stored_files.id.
// No ownership check — any authenticated user can serve a file by its stored ID.
func (h *Handler) ServeFile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid file id")
		return
	}

	url, err := h.service.ServeStoredFile(c.Request.Context(), id)
	if errors.Is(err, ErrStoredFileNotFound) {
		response.NotFound(c, "file not found")
		return
	}
	if err != nil {
		h.log.Error("serve file failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	c.Redirect(302, url)
}
