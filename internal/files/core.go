package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

var allowedMimeTypes = map[string]bool{
	// Documents
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/vnd.ms-excel":      true,
	"application/vnd.ms-powerpoint": true,
	"text/plain":                    true,
	"text/csv":                      true,
	"text/html":                     true,
	"application/rtf":               true,

	// Images
	"image/jpeg":    true,
	"image/png":     true,
	"image/gif":     true,
	"image/webp":    true,
	"image/svg+xml": true,

	// Audio
	"audio/mpeg": true,
	"audio/wav":  true,
	"audio/ogg":  true,
	"audio/webm": true,
	"audio/mp4":  true,

	// Video
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"video/quicktime": true,

	// Archives
	"application/zip":              true,
	"application/x-rar-compressed": true,
	"application/x-7z-compressed":  true,
	"application/gzip":             true,

	// Code/Data
	"application/json": true,
	"application/xml":  true,
	"text/xml":         true,
}

func HashContent(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func DetectContentType(data []byte) string {
	return http.DetectContentType(data)
}

func IsAllowedContentType(contentType string) bool {
	ct := strings.Split(contentType, ";")[0]
	ct = strings.TrimSpace(ct)
	return allowedMimeTypes[ct]
}

func GenerateStorageKey(userID uuid.UUID, filename string) string {
	unique := uuid.New().String()[:8]
	return fmt.Sprintf("files/%s/%s_%s", userID.String(), unique, filename)
}

func FormatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func ValidateOwnership(resourceOwnerID, requestingUserID uuid.UUID) error {
	if resourceOwnerID != requestingUserID {
		return ErrNotOwner
	}
	return nil
}

func ValidateFileSize(size, limit int64) error {
	if size > limit {
		return ErrFileTooLarge
	}
	return nil
}

func ValidateStorageQuota(currentUsed, additionalBytes, quota int64) error {
	if currentUsed+additionalBytes > quota {
		return ErrStorageQuotaExceeded
	}
	return nil
}

func ValidateContentType(content []byte) (string, error) {
	mimeType := DetectContentType(content)
	if !IsAllowedContentType(mimeType) {
		return "", ErrInvalidFileType
	}
	return mimeType, nil
}

func CanDeleteStoredFile(referenceCount int) bool {
	return referenceCount == 0
}

func BuildFolder(ownerID uuid.UUID, name string, parentID *uuid.UUID) *Folder {
	return &Folder{
		ID:       uuid.New(),
		OwnerID:  ownerID,
		ParentID: parentID,
		Name:     name,
	}
}

func BuildUserFile(ownerID uuid.UUID, name string, folderID *uuid.UUID, storedFileID uuid.UUID) *UserFile {
	return &UserFile{
		ID:           uuid.New(),
		OwnerID:      ownerID,
		FolderID:     folderID,
		StoredFileID: storedFileID,
		Name:         name,
	}
}

func BuildStoredFile(storageKey, contentHash, mimeType string, sizeBytes int64, uploadedBy uuid.UUID) *StoredFile {
	return &StoredFile{
		ID:          uuid.New(),
		StorageKey:  storageKey,
		ContentHash: contentHash,
		SizeBytes:   sizeBytes,
		MimeType:    mimeType,
		UploadedBy:  &uploadedBy,
	}
}

func BuildStorageUsage(used, limit int64) *StorageUsage {
	return &StorageUsage{
		UsedBytes:      used,
		LimitBytes:     limit,
		UsedFormatted:  FormatFileSize(used),
		LimitFormatted: FormatFileSize(limit),
	}
}

func BuildUserFileWithMeta(userFile *UserFile, sizeBytes int64, mimeType string) *UserFileWithMeta {
	return &UserFileWithMeta{
		UserFile:  *userFile,
		SizeBytes: sizeBytes,
		MimeType:  mimeType,
	}
}

func PrepareUpload(content []byte, size, sizeLimit, usedStorage, storageQuota int64) (hash string, mimeType string, err error) {
	if err = ValidateFileSize(size, sizeLimit); err != nil {
		return "", "", err
	}

	if err = ValidateStorageQuota(usedStorage, size, storageQuota); err != nil {
		return "", "", err
	}

	mimeType, err = ValidateContentType(content)
	if err != nil {
		return "", "", err
	}

	hash = HashContent(content)
	return hash, mimeType, nil
}
