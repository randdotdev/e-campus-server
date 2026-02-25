package files

import (
	"testing"

	"github.com/google/uuid"
)

func TestHashContent(t *testing.T) {
	content := []byte("hello world")
	hash := HashContent(content)

	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Same content should produce same hash
	hash2 := HashContent(content)
	if hash != hash2 {
		t.Error("same content should produce same hash")
	}

	// Different content should produce different hash
	hash3 := HashContent([]byte("different"))
	if hash == hash3 {
		t.Error("different content should produce different hash")
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{"plain text", []byte("hello world"), "text/plain; charset=utf-8"},
		{"html", []byte("<html><body>test</body></html>"), "text/html; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectContentType(tt.content)
			if got != tt.want {
				t.Errorf("DetectContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAllowedContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/pdf", true},
		{"image/jpeg", true},
		{"image/png", true},
		{"video/mp4", true},
		{"audio/mpeg", true},
		{"text/plain", true},
		{"application/octet-stream", false},
		{"application/x-executable", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			if got := IsAllowedContentType(tt.contentType); got != tt.want {
				t.Errorf("IsAllowedContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestGenerateStorageKey(t *testing.T) {
	userID := uuid.New()
	filename := "test.pdf"

	key := GenerateStorageKey(userID, filename)

	if key == "" {
		t.Error("storage key should not be empty")
	}

	// Should contain user ID
	if !contains(key, userID.String()) {
		t.Error("storage key should contain user ID")
	}

	// Should contain filename
	if !contains(key, filename) {
		t.Error("storage key should contain filename")
	}

	// Different calls should produce different keys (unique prefix)
	key2 := GenerateStorageKey(userID, filename)
	if key == key2 {
		t.Error("storage keys should be unique")
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1 KB"},
		{1536, "2 KB"},
		{1048576, "1 MB"},
		{5242880, "5 MB"},
		{1073741824, "1.0 GB"},
		{5368709120, "5.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := FormatFileSize(tt.bytes); got != tt.want {
				t.Errorf("FormatFileSize(%d) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestValidateOwnership(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name       string
		resourceID uuid.UUID
		requestID  uuid.UUID
		wantErr    error
	}{
		{"owner matches", ownerID, ownerID, nil},
		{"owner mismatch", ownerID, otherID, ErrNotOwner},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOwnership(tt.resourceID, tt.requestID)
			if err != tt.wantErr {
				t.Errorf("ValidateOwnership() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		limit   int64
		wantErr error
	}{
		{"under limit", 100, 200, nil},
		{"at limit", 200, 200, nil},
		{"over limit", 201, 200, ErrFileTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSize(tt.size, tt.limit)
			if err != tt.wantErr {
				t.Errorf("ValidateFileSize() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStorageQuota(t *testing.T) {
	tests := []struct {
		name       string
		used       int64
		additional int64
		quota      int64
		wantErr    error
	}{
		{"under quota", 100, 50, 200, nil},
		{"at quota", 100, 100, 200, nil},
		{"over quota", 100, 101, 200, ErrStorageQuotaExceeded},
		{"already over", 250, 10, 200, ErrStorageQuotaExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStorageQuota(tt.used, tt.additional, tt.quota)
			if err != tt.wantErr {
				t.Errorf("ValidateStorageQuota() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		wantErr bool
	}{
		{"plain text allowed", []byte("hello world"), false},
		{"html allowed", []byte("<html><body>test</body></html>"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mimeType, err := ValidateContentType(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContentType() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && mimeType == "" {
				t.Error("ValidateContentType() mimeType should not be empty")
			}
		})
	}
}

func TestCanDeleteStoredFile(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  bool
	}{
		{"no references", 0, true},
		{"one reference", 1, false},
		{"many references", 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanDeleteStoredFile(tt.count); got != tt.want {
				t.Errorf("CanDeleteStoredFile(%d) = %v, want %v", tt.count, got, tt.want)
			}
		})
	}
}

func TestBuildFolder(t *testing.T) {
	ownerID := uuid.New()
	parentID := uuid.New()

	t.Run("root folder", func(t *testing.T) {
		folder := BuildFolder(ownerID, "Documents", nil)
		if folder.OwnerID != ownerID {
			t.Errorf("OwnerID = %v, want %v", folder.OwnerID, ownerID)
		}
		if folder.Name != "Documents" {
			t.Errorf("Name = %v, want Documents", folder.Name)
		}
		if folder.ParentID != nil {
			t.Error("ParentID should be nil")
		}
		if folder.ID == uuid.Nil {
			t.Error("ID should be generated")
		}
	})

	t.Run("nested folder", func(t *testing.T) {
		folder := BuildFolder(ownerID, "Subfolder", &parentID)
		if folder.ParentID == nil || *folder.ParentID != parentID {
			t.Errorf("ParentID = %v, want %v", folder.ParentID, parentID)
		}
	})
}

func TestBuildUserFile(t *testing.T) {
	ownerID := uuid.New()
	folderID := uuid.New()
	storedFileID := uuid.New()

	file := BuildUserFile(ownerID, "test.pdf", &folderID, storedFileID)

	if file.OwnerID != ownerID {
		t.Errorf("OwnerID = %v, want %v", file.OwnerID, ownerID)
	}
	if file.Name != "test.pdf" {
		t.Errorf("Name = %v, want test.pdf", file.Name)
	}
	if file.FolderID == nil || *file.FolderID != folderID {
		t.Errorf("FolderID = %v, want %v", file.FolderID, folderID)
	}
	if file.StoredFileID != storedFileID {
		t.Errorf("StoredFileID = %v, want %v", file.StoredFileID, storedFileID)
	}
	if file.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
}

func TestBuildStoredFile(t *testing.T) {
	userID := uuid.New()

	sf := BuildStoredFile("files/key", "abc123", "application/pdf", 1024, userID)

	if sf.StorageKey != "files/key" {
		t.Errorf("StorageKey = %v, want files/key", sf.StorageKey)
	}
	if sf.ContentHash != "abc123" {
		t.Errorf("ContentHash = %v, want abc123", sf.ContentHash)
	}
	if sf.MimeType != "application/pdf" {
		t.Errorf("MimeType = %v, want application/pdf", sf.MimeType)
	}
	if sf.SizeBytes != 1024 {
		t.Errorf("SizeBytes = %v, want 1024", sf.SizeBytes)
	}
	if sf.UploadedBy == nil || *sf.UploadedBy != userID {
		t.Errorf("UploadedBy = %v, want %v", sf.UploadedBy, userID)
	}
}

func TestBuildStorageUsage(t *testing.T) {
	usage := BuildStorageUsage(512*1024*1024, 1024*1024*1024)

	if usage.UsedBytes != 512*1024*1024 {
		t.Errorf("UsedBytes = %v, want %v", usage.UsedBytes, 512*1024*1024)
	}
	if usage.LimitBytes != 1024*1024*1024 {
		t.Errorf("LimitBytes = %v, want %v", usage.LimitBytes, 1024*1024*1024)
	}
	if usage.UsedFormatted != "512 MB" {
		t.Errorf("UsedFormatted = %v, want 512 MB", usage.UsedFormatted)
	}
	if usage.LimitFormatted != "1.0 GB" {
		t.Errorf("LimitFormatted = %v, want 1.0 GB", usage.LimitFormatted)
	}
}

func TestPrepareUpload(t *testing.T) {
	content := []byte("hello world")

	t.Run("valid upload", func(t *testing.T) {
		hash, mimeType, err := PrepareUpload(content, int64(len(content)), 100, 0, 1000)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if hash == "" {
			t.Error("hash should not be empty")
		}
		if mimeType == "" {
			t.Error("mimeType should not be empty")
		}
	})

	t.Run("file too large", func(t *testing.T) {
		_, _, err := PrepareUpload(content, 100, 50, 0, 1000)
		if err != ErrFileTooLarge {
			t.Errorf("err = %v, want ErrFileTooLarge", err)
		}
	})

	t.Run("quota exceeded", func(t *testing.T) {
		_, _, err := PrepareUpload(content, int64(len(content)), 100, 990, 1000)
		if err != ErrStorageQuotaExceeded {
			t.Errorf("err = %v, want ErrStorageQuotaExceeded", err)
		}
	})
}
