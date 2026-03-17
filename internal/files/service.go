package files

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type FileRepository interface {
	CreateStoredFile(ctx context.Context, sf *StoredFile) error
	GetStoredFileByID(ctx context.Context, id uuid.UUID) (*StoredFile, error)
	GetStoredFileByHash(ctx context.Context, hash string) (*StoredFile, error)
	CountStoredFileReferences(ctx context.Context, id uuid.UUID) (int, error)
	DeleteStoredFile(ctx context.Context, id uuid.UUID) error
	CreateFolder(ctx context.Context, f *Folder) error
	GetFolderByID(ctx context.Context, id uuid.UUID) (*Folder, error)
	ListFolders(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, params pagination.PageParams) ([]Folder, bool, error)
	UpdateFolder(ctx context.Context, f *Folder) error
	DeleteFolder(ctx context.Context, id uuid.UUID) error
	IsFolderEmpty(ctx context.Context, id uuid.UUID) (bool, error)
	CreateUserFile(ctx context.Context, uf *UserFile) error
	GetUserFileByID(ctx context.Context, id uuid.UUID) (*UserFile, error)
	GetUserFileWithMeta(ctx context.Context, id uuid.UUID) (*UserFileWithMeta, error)
	ListUserFiles(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID, params pagination.PageParams) ([]UserFileWithMeta, bool, error)
	UpdateUserFile(ctx context.Context, uf *UserFile) error
	DeleteUserFile(ctx context.Context, id uuid.UUID) error
	GetStorageUsed(ctx context.Context, ownerID uuid.UUID) (int64, error)
}

type FileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
	PresignedGetURL(ctx context.Context, key string, duration time.Duration) (string, error)
}

type StorageLimits interface {
	GetFileSizeLimit(ctx context.Context, userID uuid.UUID) (int64, error)
	GetStorageLimit(ctx context.Context, userID uuid.UUID) (int64, error)
}

type Service struct {
	repo    FileRepository
	storage FileStorage
	limits  StorageLimits
}

func NewService(repo FileRepository, storage FileStorage, limits StorageLimits) *Service {
	return &Service{
		repo:    repo,
		storage: storage,
		limits:  limits,
	}
}

func (s *Service) CreateFolder(ctx context.Context, ownerID uuid.UUID, name string, parentID *uuid.UUID) (*Folder, error) {
	if parentID != nil {
		parent, err := s.repo.GetFolderByID(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrFolderNotFound
		}
		if err := ValidateOwnership(parent.OwnerID, ownerID); err != nil {
			return nil, err
		}
	}

	folder := BuildFolder(ownerID, name, parentID)

	if err := s.repo.CreateFolder(ctx, folder); err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateFolderName
		}
		return nil, err
	}

	return folder, nil
}

func (s *Service) GetFolder(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*Folder, error) {
	folder, err := s.repo.GetFolderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}
	if err := ValidateOwnership(folder.OwnerID, ownerID); err != nil {
		return nil, err
	}
	return folder, nil
}

func (s *Service) ListFolders(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, params pagination.PageParams) ([]Folder, bool, error) {
	return s.repo.ListFolders(ctx, ownerID, parentID, params)
}

func (s *Service) UpdateFolder(ctx context.Context, id uuid.UUID, ownerID uuid.UUID, name string, parentID *uuid.UUID) (*Folder, error) {
	folder, err := s.repo.GetFolderByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, ErrFolderNotFound
	}
	if err := ValidateOwnership(folder.OwnerID, ownerID); err != nil {
		return nil, err
	}

	folder.Name = name
	folder.ParentID = parentID

	if err := s.repo.UpdateFolder(ctx, folder); err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateFolderName
		}
		return nil, err
	}

	return folder, nil
}

func (s *Service) DeleteFolder(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	folder, err := s.repo.GetFolderByID(ctx, id)
	if err != nil {
		return err
	}
	if folder == nil {
		return ErrFolderNotFound
	}
	if err := ValidateOwnership(folder.OwnerID, ownerID); err != nil {
		return err
	}

	empty, err := s.repo.IsFolderEmpty(ctx, id)
	if err != nil {
		return err
	}
	if !empty {
		return ErrFolderNotEmpty
	}

	return s.repo.DeleteFolder(ctx, id)
}

func (s *Service) UploadFile(ctx context.Context, ownerID uuid.UUID, name string, folderID *uuid.UUID, reader io.Reader, size int64) (*UserFileWithMeta, error) {
	// Fetch limits
	sizeLimit, err := s.limits.GetFileSizeLimit(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	used, err := s.repo.GetStorageUsed(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	quota, err := s.limits.GetStorageLimit(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	// Verify folder ownership if specified
	if folderID != nil {
		folder, err := s.repo.GetFolderByID(ctx, *folderID)
		if err != nil {
			return nil, err
		}
		if folder == nil {
			return nil, ErrFolderNotFound
		}
		if err := ValidateOwnership(folder.OwnerID, ownerID); err != nil {
			return nil, err
		}
	}

	// Read content
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// Pure validation and preparation
	hash, mimeType, err := PrepareUpload(content, size, sizeLimit, used, quota)
	if err != nil {
		return nil, err
	}

	// Check for existing file with same hash (deduplication)
	existing, err := s.repo.GetStoredFileByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	var storedFile *StoredFile
	if existing != nil {
		storedFile = existing
	} else {
		storageKey := GenerateStorageKey(ownerID, name)
		if err := s.storage.Upload(ctx, storageKey, bytes.NewReader(content), int64(len(content)), mimeType); err != nil {
			return nil, err
		}

		storedFile = BuildStoredFile(storageKey, hash, mimeType, size, ownerID)
		if err := s.repo.CreateStoredFile(ctx, storedFile); err != nil {
			return nil, err
		}
	}

	userFile := BuildUserFile(ownerID, name, folderID, storedFile.ID)
	if err := s.repo.CreateUserFile(ctx, userFile); err != nil {
		return nil, err
	}

	return BuildUserFileWithMeta(userFile, storedFile.SizeBytes, storedFile.MimeType), nil
}

func (s *Service) GetFile(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*UserFileWithMeta, error) {
	file, err := s.repo.GetUserFileWithMeta(ctx, id)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}
	if err := ValidateOwnership(file.OwnerID, ownerID); err != nil {
		return nil, err
	}
	return file, nil
}

func (s *Service) ListFiles(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID, params pagination.PageParams) ([]UserFileWithMeta, bool, error) {
	return s.repo.ListUserFiles(ctx, ownerID, folderID, params)
}

func (s *Service) UpdateFile(ctx context.Context, id uuid.UUID, ownerID uuid.UUID, name string, folderID *uuid.UUID) (*UserFileWithMeta, error) {
	file, err := s.repo.GetUserFileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, ErrFileNotFound
	}
	if err := ValidateOwnership(file.OwnerID, ownerID); err != nil {
		return nil, err
	}

	if folderID != nil {
		folder, err := s.repo.GetFolderByID(ctx, *folderID)
		if err != nil {
			return nil, err
		}
		if folder == nil {
			return nil, ErrFolderNotFound
		}
		if err := ValidateOwnership(folder.OwnerID, ownerID); err != nil {
			return nil, err
		}
	}

	file.Name = name
	file.FolderID = folderID

	if err := s.repo.UpdateUserFile(ctx, file); err != nil {
		return nil, err
	}

	return s.repo.GetUserFileWithMeta(ctx, id)
}

func (s *Service) DeleteFile(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) error {
	file, err := s.repo.GetUserFileByID(ctx, id)
	if err != nil {
		return err
	}
	if file == nil {
		return ErrFileNotFound
	}
	if err := ValidateOwnership(file.OwnerID, ownerID); err != nil {
		return err
	}

	storedFileID := file.StoredFileID

	if err := s.repo.DeleteUserFile(ctx, id); err != nil {
		return err
	}

	// Check if stored file is still referenced
	count, err := s.repo.CountStoredFileReferences(ctx, storedFileID)
	if err != nil {
		return err
	}

	// TODO: Also check lesson_attachments when content package is ready

	if CanDeleteStoredFile(count) {
		storedFile, err := s.repo.GetStoredFileByID(ctx, storedFileID)
		if err != nil {
			return err
		}
		if storedFile != nil {
			if err := s.storage.Delete(ctx, storedFile.StorageKey); err != nil {
				return err
			}
			if err := s.repo.DeleteStoredFile(ctx, storedFileID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) CopyToStorage(ctx context.Context, storedFileID uuid.UUID, ownerID uuid.UUID, name string, folderID *uuid.UUID) (*UserFileWithMeta, error) {
	storedFile, err := s.repo.GetStoredFileByID(ctx, storedFileID)
	if err != nil {
		return nil, err
	}
	if storedFile == nil {
		return nil, ErrStoredFileNotFound
	}

	// Fetch quota info
	used, err := s.repo.GetStorageUsed(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	quota, err := s.limits.GetStorageLimit(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	// Pure validation
	if err := ValidateStorageQuota(used, storedFile.SizeBytes, quota); err != nil {
		return nil, err
	}

	// Verify folder ownership if specified
	if folderID != nil {
		folder, err := s.repo.GetFolderByID(ctx, *folderID)
		if err != nil {
			return nil, err
		}
		if folder == nil {
			return nil, ErrFolderNotFound
		}
		if err := ValidateOwnership(folder.OwnerID, ownerID); err != nil {
			return nil, err
		}
	}

	userFile := BuildUserFile(ownerID, name, folderID, storedFileID)
	if err := s.repo.CreateUserFile(ctx, userFile); err != nil {
		return nil, err
	}

	return BuildUserFileWithMeta(userFile, storedFile.SizeBytes, storedFile.MimeType), nil
}

func (s *Service) GetFileURL(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (string, error) {
	file, err := s.repo.GetUserFileByID(ctx, id)
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", ErrFileNotFound
	}
	if err := ValidateOwnership(file.OwnerID, ownerID); err != nil {
		return "", err
	}

	storedFile, err := s.repo.GetStoredFileByID(ctx, file.StoredFileID)
	if err != nil {
		return "", err
	}
	if storedFile == nil {
		return "", ErrStoredFileNotFound
	}

	return s.storage.PresignedGetURL(ctx, storedFile.StorageKey, time.Hour)
}

func (s *Service) GetStorageUsage(ctx context.Context, ownerID uuid.UUID) (*StorageUsage, error) {
	used, err := s.repo.GetStorageUsed(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	limit, err := s.limits.GetStorageLimit(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	return BuildStorageUsage(used, limit), nil
}

func isDuplicateKeyError(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") || contains(err.Error(), "unique constraint"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
