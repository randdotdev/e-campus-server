package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/pagination"
)

// mockRepo implements FileRepository for testing
type mockRepo struct {
	folders      map[uuid.UUID]*Folder
	userFiles    map[uuid.UUID]*UserFile
	storedFiles  map[uuid.UUID]*StoredFile
	storageUsed  int64
	folderEmpty  bool
	createErr    error
	duplicateErr bool
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		folders:     make(map[uuid.UUID]*Folder),
		userFiles:   make(map[uuid.UUID]*UserFile),
		storedFiles: make(map[uuid.UUID]*StoredFile),
		folderEmpty: true,
	}
}

func (m *mockRepo) CreateFolder(ctx context.Context, f *Folder) error {
	if m.duplicateErr {
		return errors.New("duplicate key")
	}
	if m.createErr != nil {
		return m.createErr
	}
	m.folders[f.ID] = f
	return nil
}

func (m *mockRepo) GetFolderByID(ctx context.Context, id uuid.UUID) (*Folder, error) {
	f, ok := m.folders[id]
	if !ok {
		return nil, nil
	}
	return f, nil
}

func (m *mockRepo) ListFolders(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, params pagination.PageParams) ([]Folder, bool, error) {
	var result []Folder
	for _, f := range m.folders {
		if f.OwnerID == ownerID {
			if parentID == nil && f.ParentID == nil {
				result = append(result, *f)
			} else if parentID != nil && f.ParentID != nil && *f.ParentID == *parentID {
				result = append(result, *f)
			}
		}
	}
	return result, false, nil
}

func (m *mockRepo) UpdateFolder(ctx context.Context, f *Folder) error {
	if m.duplicateErr {
		return errors.New("duplicate key")
	}
	m.folders[f.ID] = f
	return nil
}

func (m *mockRepo) DeleteFolder(ctx context.Context, id uuid.UUID) error {
	delete(m.folders, id)
	return nil
}

func (m *mockRepo) IsFolderEmpty(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.folderEmpty, nil
}

func (m *mockRepo) CreateStoredFile(ctx context.Context, sf *StoredFile) error {
	m.storedFiles[sf.ID] = sf
	return nil
}

func (m *mockRepo) GetStoredFileByID(ctx context.Context, id uuid.UUID) (*StoredFile, error) {
	sf, ok := m.storedFiles[id]
	if !ok {
		return nil, nil
	}
	return sf, nil
}

func (m *mockRepo) GetStoredFileByHash(ctx context.Context, hash string) (*StoredFile, error) {
	for _, sf := range m.storedFiles {
		if sf.ContentHash == hash {
			return sf, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) CountStoredFileReferences(ctx context.Context, id uuid.UUID) (int, error) {
	count := 0
	for _, uf := range m.userFiles {
		if uf.StoredFileID == id {
			count++
		}
	}
	return count, nil
}

func (m *mockRepo) DeleteStoredFile(ctx context.Context, id uuid.UUID) error {
	delete(m.storedFiles, id)
	return nil
}

func (m *mockRepo) CreateUserFile(ctx context.Context, uf *UserFile) error {
	m.userFiles[uf.ID] = uf
	return nil
}

func (m *mockRepo) GetUserFileByID(ctx context.Context, id uuid.UUID) (*UserFile, error) {
	uf, ok := m.userFiles[id]
	if !ok {
		return nil, nil
	}
	return uf, nil
}

func (m *mockRepo) GetUserFileWithMeta(ctx context.Context, id uuid.UUID) (*UserFileWithMeta, error) {
	uf, ok := m.userFiles[id]
	if !ok {
		return nil, nil
	}
	sf := m.storedFiles[uf.StoredFileID]
	return &UserFileWithMeta{
		UserFile:  *uf,
		SizeBytes: sf.SizeBytes,
		MimeType:  sf.MimeType,
	}, nil
}

func (m *mockRepo) ListUserFiles(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID, params pagination.PageParams) ([]UserFileWithMeta, bool, error) {
	var result []UserFileWithMeta
	for _, uf := range m.userFiles {
		if uf.OwnerID == ownerID {
			sf := m.storedFiles[uf.StoredFileID]
			result = append(result, UserFileWithMeta{
				UserFile:  *uf,
				SizeBytes: sf.SizeBytes,
				MimeType:  sf.MimeType,
			})
		}
	}
	return result, false, nil
}

func (m *mockRepo) UpdateUserFile(ctx context.Context, uf *UserFile) error {
	m.userFiles[uf.ID] = uf
	return nil
}

func (m *mockRepo) DeleteUserFile(ctx context.Context, id uuid.UUID) error {
	delete(m.userFiles, id)
	return nil
}

func (m *mockRepo) GetStorageUsed(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	return m.storageUsed, nil
}

// mockStorage implements FileStorage for testing
type mockStorage struct {
	files      map[string][]byte
	uploadErr  error
	deleteErr  error
	presignURL string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		files:      make(map[string][]byte),
		presignURL: "https://example.com/presigned",
	}
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	data, _ := io.ReadAll(reader)
	m.files[key] = data
	return nil
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.files, key)
	return nil
}

func (m *mockStorage) PresignedGetURL(ctx context.Context, key string, duration time.Duration) (string, error) {
	return m.presignURL, nil
}

// mockLimits implements StorageLimits for testing
type mockLimits struct {
	fileSizeLimit int64
	storageLimit  int64
}

func (m *mockLimits) GetFileSizeLimit(ctx context.Context, userID uuid.UUID) (int64, error) {
	return m.fileSizeLimit, nil
}

func (m *mockLimits) GetStorageLimit(ctx context.Context, userID uuid.UUID) (int64, error) {
	return m.storageLimit, nil
}

// newTestService creates a Service with mocks for testing
func newTestService() (*Service, *mockRepo, *mockStorage) {
	repo := newMockRepo()
	storage := newMockStorage()
	limits := &mockLimits{
		fileSizeLimit: 100 * 1024 * 1024,  // 100MB
		storageLimit:  1024 * 1024 * 1024, // 1GB
	}
	service := NewService(repo, storage, limits)
	return service, repo, storage
}

func TestServiceCreateFolder(t *testing.T) {
	t.Run("creates folder at root", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		folder, err := service.CreateFolder(context.Background(), ownerID, "Documents", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if folder.Name != "Documents" {
			t.Errorf("Name = %v, want Documents", folder.Name)
		}
		if folder.OwnerID != ownerID {
			t.Errorf("OwnerID = %v, want %v", folder.OwnerID, ownerID)
		}
		if folder.ParentID != nil {
			t.Error("ParentID should be nil for root folder")
		}
	})

	t.Run("creates nested folder", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		parent, _ := service.CreateFolder(context.Background(), ownerID, "Parent", nil)
		child, err := service.CreateFolder(context.Background(), ownerID, "Child", &parent.ID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if child.ParentID == nil || *child.ParentID != parent.ID {
			t.Errorf("ParentID = %v, want %v", child.ParentID, parent.ID)
		}
	})

	t.Run("fails if parent not found", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		nonExistent := uuid.New()
		_, err := service.CreateFolder(context.Background(), ownerID, "Child", &nonExistent)

		if err != ErrFolderNotFound {
			t.Errorf("err = %v, want ErrFolderNotFound", err)
		}
	})

	t.Run("fails if parent not owned", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerA := uuid.New()
		ownerB := uuid.New()
		parent, _ := service.CreateFolder(context.Background(), ownerA, "Parent", nil)
		_, err := service.CreateFolder(context.Background(), ownerB, "Child", &parent.ID)

		if err != ErrNotOwner {
			t.Errorf("err = %v, want ErrNotOwner", err)
		}
	})

	t.Run("fails on duplicate name", func(t *testing.T) {
		service, repo, _ := newTestService()
		repo.duplicateErr = true

		ownerID := uuid.New()
		_, err := service.CreateFolder(context.Background(), ownerID, "Documents", nil)

		if err != ErrDuplicateFolderName {
			t.Errorf("err = %v, want ErrDuplicateFolderName", err)
		}
	})
}

func TestServiceGetFolder(t *testing.T) {
	t.Run("returns folder", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		created, _ := service.CreateFolder(context.Background(), ownerID, "Documents", nil)

		folder, err := service.GetFolder(context.Background(), created.ID, ownerID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if folder.ID != created.ID {
			t.Errorf("ID = %v, want %v", folder.ID, created.ID)
		}
	})

	t.Run("fails if not found", func(t *testing.T) {
		service, _, _ := newTestService()

		_, err := service.GetFolder(context.Background(), uuid.New(), uuid.New())

		if err != ErrFolderNotFound {
			t.Errorf("err = %v, want ErrFolderNotFound", err)
		}
	})

	t.Run("fails if not owner", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerA := uuid.New()
		ownerB := uuid.New()
		created, _ := service.CreateFolder(context.Background(), ownerA, "Documents", nil)

		_, err := service.GetFolder(context.Background(), created.ID, ownerB)

		if err != ErrNotOwner {
			t.Errorf("err = %v, want ErrNotOwner", err)
		}
	})
}

func TestServiceDeleteFolder(t *testing.T) {
	t.Run("deletes empty folder", func(t *testing.T) {
		service, repo, _ := newTestService()
		repo.folderEmpty = true

		ownerID := uuid.New()
		folder, _ := service.CreateFolder(context.Background(), ownerID, "ToDelete", nil)

		err := service.DeleteFolder(context.Background(), folder.ID, ownerID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deleted
		_, err = service.GetFolder(context.Background(), folder.ID, ownerID)
		if err != ErrFolderNotFound {
			t.Error("folder should be deleted")
		}
	})

	t.Run("fails if not empty", func(t *testing.T) {
		service, repo, _ := newTestService()
		repo.folderEmpty = false

		ownerID := uuid.New()
		folder, _ := service.CreateFolder(context.Background(), ownerID, "NotEmpty", nil)

		err := service.DeleteFolder(context.Background(), folder.ID, ownerID)

		if err != ErrFolderNotEmpty {
			t.Errorf("err = %v, want ErrFolderNotEmpty", err)
		}
	})

	t.Run("fails if not owner", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerA := uuid.New()
		ownerB := uuid.New()
		folder, _ := service.CreateFolder(context.Background(), ownerA, "Folder", nil)

		err := service.DeleteFolder(context.Background(), folder.ID, ownerB)

		if err != ErrNotOwner {
			t.Errorf("err = %v, want ErrNotOwner", err)
		}
	})
}

func TestServiceUploadFile(t *testing.T) {
	t.Run("uploads file successfully", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		content := []byte("hello world")

		file, err := service.UploadFile(context.Background(), ownerID, "test.txt", nil, bytes.NewReader(content), int64(len(content)))

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if file.Name != "test.txt" {
			t.Errorf("Name = %v, want test.txt", file.Name)
		}
		if file.OwnerID != ownerID {
			t.Errorf("OwnerID = %v, want %v", file.OwnerID, ownerID)
		}
	})

	t.Run("fails if file too large", func(t *testing.T) {
		repo := newMockRepo()
		storage := newMockStorage()
		limits := &mockLimits{fileSizeLimit: 10, storageLimit: 1024 * 1024 * 1024}
		service := NewService(repo, storage, limits)

		ownerID := uuid.New()
		content := []byte("hello world - this is too large")

		_, err := service.UploadFile(context.Background(), ownerID, "test.txt", nil, bytes.NewReader(content), int64(len(content)))

		if err != ErrFileTooLarge {
			t.Errorf("err = %v, want ErrFileTooLarge", err)
		}
	})

	t.Run("fails if quota exceeded", func(t *testing.T) {
		repo := newMockRepo()
		repo.storageUsed = 1024 * 1024 * 1024 // Already at limit
		storage := newMockStorage()
		limits := &mockLimits{fileSizeLimit: 100 * 1024 * 1024, storageLimit: 1024 * 1024 * 1024}
		service := NewService(repo, storage, limits)

		ownerID := uuid.New()
		content := []byte("hello world")

		_, err := service.UploadFile(context.Background(), ownerID, "test.txt", nil, bytes.NewReader(content), int64(len(content)))

		if err != ErrStorageQuotaExceeded {
			t.Errorf("err = %v, want ErrStorageQuotaExceeded", err)
		}
	})

	t.Run("deduplicates identical content", func(t *testing.T) {
		service, repo, _ := newTestService()

		ownerID := uuid.New()
		content := []byte("hello world")

		file1, _ := service.UploadFile(context.Background(), ownerID, "file1.txt", nil, bytes.NewReader(content), int64(len(content)))
		file2, _ := service.UploadFile(context.Background(), ownerID, "file2.txt", nil, bytes.NewReader(content), int64(len(content)))

		// Should share same stored file
		if file1.StoredFileID != file2.StoredFileID {
			t.Error("identical content should share stored file")
		}

		// Should only have one stored file
		if len(repo.storedFiles) != 1 {
			t.Errorf("storedFiles count = %d, want 1", len(repo.storedFiles))
		}
	})

	t.Run("uploads to folder", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		folder, _ := service.CreateFolder(context.Background(), ownerID, "Documents", nil)
		content := []byte("hello world")

		file, err := service.UploadFile(context.Background(), ownerID, "test.txt", &folder.ID, bytes.NewReader(content), int64(len(content)))

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if file.FolderID == nil || *file.FolderID != folder.ID {
			t.Errorf("FolderID = %v, want %v", file.FolderID, folder.ID)
		}
	})
}

func TestServiceGetFile(t *testing.T) {
	t.Run("returns file with metadata", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		content := []byte("hello world")
		created, _ := service.UploadFile(context.Background(), ownerID, "test.txt", nil, bytes.NewReader(content), int64(len(content)))

		file, err := service.GetFile(context.Background(), created.ID, ownerID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if file.Name != "test.txt" {
			t.Errorf("Name = %v, want test.txt", file.Name)
		}
		if file.SizeBytes != int64(len(content)) {
			t.Errorf("SizeBytes = %v, want %v", file.SizeBytes, len(content))
		}
	})

	t.Run("fails if not owner", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerA := uuid.New()
		ownerB := uuid.New()
		content := []byte("hello world")
		created, _ := service.UploadFile(context.Background(), ownerA, "test.txt", nil, bytes.NewReader(content), int64(len(content)))

		_, err := service.GetFile(context.Background(), created.ID, ownerB)

		if err != ErrNotOwner {
			t.Errorf("err = %v, want ErrNotOwner", err)
		}
	})
}

func TestServiceDeleteFile(t *testing.T) {
	t.Run("deletes file and orphaned stored file", func(t *testing.T) {
		service, repo, storage := newTestService()

		ownerID := uuid.New()
		content := []byte("hello world")

		file, _ := service.UploadFile(context.Background(), ownerID, "test.txt", nil, bytes.NewReader(content), int64(len(content)))
		storedFileID := file.StoredFileID

		// Verify S3 file exists
		if len(storage.files) != 1 {
			t.Errorf("storage files = %d, want 1", len(storage.files))
		}

		err := service.DeleteFile(context.Background(), file.ID, ownerID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Stored file should be deleted too
		if _, exists := repo.storedFiles[storedFileID]; exists {
			t.Error("orphaned stored file should be deleted")
		}

		// S3 file should be deleted
		if len(storage.files) != 0 {
			t.Errorf("storage files = %d, want 0", len(storage.files))
		}
	})

	t.Run("keeps shared stored file", func(t *testing.T) {
		service, repo, storage := newTestService()

		ownerID := uuid.New()
		content := []byte("hello world")

		file1, _ := service.UploadFile(context.Background(), ownerID, "file1.txt", nil, bytes.NewReader(content), int64(len(content)))
		file2, _ := service.UploadFile(context.Background(), ownerID, "file2.txt", nil, bytes.NewReader(content), int64(len(content)))
		storedFileID := file1.StoredFileID

		// Delete first file
		err := service.DeleteFile(context.Background(), file1.ID, ownerID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Stored file should still exist (referenced by file2)
		if _, exists := repo.storedFiles[storedFileID]; !exists {
			t.Error("shared stored file should not be deleted")
		}

		// S3 file should still exist
		if len(storage.files) != 1 {
			t.Errorf("storage files = %d, want 1", len(storage.files))
		}

		// Delete second file
		_ = service.DeleteFile(context.Background(), file2.ID, ownerID)

		// Now stored file should be deleted
		if _, exists := repo.storedFiles[storedFileID]; exists {
			t.Error("orphaned stored file should be deleted")
		}

		// S3 file should be deleted
		if len(storage.files) != 0 {
			t.Errorf("storage files = %d, want 0", len(storage.files))
		}
	})
}

func TestServiceGetFileURL(t *testing.T) {
	t.Run("returns presigned URL", func(t *testing.T) {
		service, _, storage := newTestService()
		storage.presignURL = "https://bucket.s3.amazonaws.com/file?signed=true"

		ownerID := uuid.New()
		content := []byte("hello world")
		file, _ := service.UploadFile(context.Background(), ownerID, "test.txt", nil, bytes.NewReader(content), int64(len(content)))

		url, err := service.GetFileURL(context.Background(), file.ID, ownerID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != "https://bucket.s3.amazonaws.com/file?signed=true" {
			t.Errorf("URL = %v, want presigned URL", url)
		}
	})
}

func TestServiceCopyToStorage(t *testing.T) {
	t.Run("copies stored file to user storage", func(t *testing.T) {
		service, repo, _ := newTestService()

		ownerID := uuid.New()
		content := []byte("hello world")

		// Upload a file first
		original, _ := service.UploadFile(context.Background(), ownerID, "original.txt", nil, bytes.NewReader(content), int64(len(content)))

		// Copy it
		copy, err := service.CopyToStorage(context.Background(), original.StoredFileID, ownerID, "copy.txt", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if copy.Name != "copy.txt" {
			t.Errorf("Name = %v, want copy.txt", copy.Name)
		}
		if copy.StoredFileID != original.StoredFileID {
			t.Error("copy should reference same stored file")
		}

		// Should have 2 user files, 1 stored file
		if len(repo.userFiles) != 2 {
			t.Errorf("userFiles = %d, want 2", len(repo.userFiles))
		}
		if len(repo.storedFiles) != 1 {
			t.Errorf("storedFiles = %d, want 1", len(repo.storedFiles))
		}
	})
}

func TestServiceGetStorageUsage(t *testing.T) {
	t.Run("returns usage stats", func(t *testing.T) {
		repo := newMockRepo()
		repo.storageUsed = 512 * 1024 * 1024
		storage := newMockStorage()
		limits := &mockLimits{fileSizeLimit: 100 * 1024 * 1024, storageLimit: 1024 * 1024 * 1024}
		service := NewService(repo, storage, limits)

		usage, err := service.GetStorageUsage(context.Background(), uuid.New())

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if usage.UsedBytes != 512*1024*1024 {
			t.Errorf("UsedBytes = %v, want %v", usage.UsedBytes, 512*1024*1024)
		}
		if usage.LimitBytes != 1024*1024*1024 {
			t.Errorf("LimitBytes = %v, want %v", usage.LimitBytes, 1024*1024*1024)
		}
		if usage.UsedFormatted != "512 MB" {
			t.Errorf("UsedFormatted = %v, want 512 MB", usage.UsedFormatted)
		}
	})
}

func TestServiceListFolders(t *testing.T) {
	t.Run("lists root folders", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		_, _ = service.CreateFolder(context.Background(), ownerID, "Folder1", nil)
		_, _ = service.CreateFolder(context.Background(), ownerID, "Folder2", nil)

		folders, hasMore, err := service.ListFolders(context.Background(), ownerID, nil, pagination.PageParams{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(folders) != 2 {
			t.Errorf("folders count = %d, want 2", len(folders))
		}
		if hasMore {
			t.Error("hasMore should be false")
		}
	})
}

func TestServiceListFiles(t *testing.T) {
	t.Run("lists files in root", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		content := []byte("hello")
		_, _ = service.UploadFile(context.Background(), ownerID, "file1.txt", nil, bytes.NewReader(content), int64(len(content)))
		_, _ = service.UploadFile(context.Background(), ownerID, "file2.txt", nil, bytes.NewReader(content), int64(len(content)))

		files, hasMore, err := service.ListFiles(context.Background(), ownerID, nil, pagination.PageParams{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 2 {
			t.Errorf("files count = %d, want 2", len(files))
		}
		if hasMore {
			t.Error("hasMore should be false")
		}
	})
}

func TestServiceUpdateFolder(t *testing.T) {
	t.Run("updates folder name", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		folder, _ := service.CreateFolder(context.Background(), ownerID, "Original", nil)

		updated, err := service.UpdateFolder(context.Background(), folder.ID, ownerID, "Renamed", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "Renamed" {
			t.Errorf("Name = %v, want Renamed", updated.Name)
		}
	})
}

func TestServiceUpdateFile(t *testing.T) {
	t.Run("updates file name", func(t *testing.T) {
		service, _, _ := newTestService()

		ownerID := uuid.New()
		content := []byte("hello")
		file, _ := service.UploadFile(context.Background(), ownerID, "original.txt", nil, bytes.NewReader(content), int64(len(content)))

		updated, err := service.UpdateFile(context.Background(), file.ID, ownerID, "renamed.txt", nil)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "renamed.txt" {
			t.Errorf("Name = %v, want renamed.txt", updated.Name)
		}
	})
}
