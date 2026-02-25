package files

import (
	"github.com/google/uuid"
)

// Folder requests/responses

type CreateFolderRequest struct {
	Name     string     `json:"name" binding:"required,min=1,max=255"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type UpdateFolderRequest struct {
	Name     string     `json:"name" binding:"required,min=1,max=255"`
	ParentID *uuid.UUID `json:"parent_id"`
}

type FolderResponse struct {
	ID        uuid.UUID  `json:"id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	CreatedAt string     `json:"created_at"`
}

func ToFolderResponse(f *Folder) FolderResponse {
	return FolderResponse{
		ID:        f.ID,
		ParentID:  f.ParentID,
		Name:      f.Name,
		CreatedAt: f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ToFoldersResponse(folders []Folder) []FolderResponse {
	resp := make([]FolderResponse, len(folders))
	for i, f := range folders {
		resp[i] = ToFolderResponse(&f)
	}
	return resp
}

// File requests/responses

type UpdateFileRequest struct {
	Name     string     `json:"name" binding:"required,min=1,max=255"`
	FolderID *uuid.UUID `json:"folder_id"`
}

type CopyFileRequest struct {
	Name     string     `json:"name" binding:"required,min=1,max=255"`
	FolderID *uuid.UUID `json:"folder_id"`
}

type FileResponse struct {
	ID            uuid.UUID  `json:"id"`
	FolderID      *uuid.UUID `json:"folder_id,omitempty"`
	StoredFileID  uuid.UUID  `json:"stored_file_id"`
	Name          string     `json:"name"`
	SizeBytes     int64      `json:"size_bytes"`
	SizeFormatted string     `json:"size_formatted"`
	MimeType      string     `json:"mime_type"`
	CreatedAt     string     `json:"created_at"`
}

type FileWithURLResponse struct {
	FileResponse
	URL string `json:"url"`
}

func ToFileResponse(f *UserFileWithMeta) FileResponse {
	return FileResponse{
		ID:            f.ID,
		FolderID:      f.FolderID,
		StoredFileID:  f.StoredFileID,
		Name:          f.Name,
		SizeBytes:     f.SizeBytes,
		SizeFormatted: FormatFileSize(f.SizeBytes),
		MimeType:      f.MimeType,
		CreatedAt:     f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ToFileWithURLResponse(f *UserFileWithMeta, url string) FileWithURLResponse {
	return FileWithURLResponse{
		FileResponse: ToFileResponse(f),
		URL:          url,
	}
}

func ToFilesResponse(files []UserFileWithMeta) []FileResponse {
	resp := make([]FileResponse, len(files))
	for i, f := range files {
		resp[i] = ToFileResponse(&f)
	}
	return resp
}
