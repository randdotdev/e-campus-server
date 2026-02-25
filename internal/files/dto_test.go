package files

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToFolderResponse(t *testing.T) {
	parentID := uuid.New()
	folder := &Folder{
		ID:        uuid.New(),
		ParentID:  &parentID,
		Name:      "Documents",
		CreatedAt: time.Now(),
	}

	resp := ToFolderResponse(folder)

	if resp.ID != folder.ID {
		t.Errorf("ID = %v, want %v", resp.ID, folder.ID)
	}
	if resp.Name != folder.Name {
		t.Errorf("Name = %v, want %v", resp.Name, folder.Name)
	}
	if resp.ParentID == nil || *resp.ParentID != parentID {
		t.Errorf("ParentID = %v, want %v", resp.ParentID, &parentID)
	}
}

func TestToFoldersResponse(t *testing.T) {
	folders := []Folder{
		{ID: uuid.New(), Name: "Folder 1"},
		{ID: uuid.New(), Name: "Folder 2"},
	}

	resp := ToFoldersResponse(folders)

	if len(resp) != 2 {
		t.Fatalf("len = %d, want 2", len(resp))
	}
	if resp[0].Name != "Folder 1" {
		t.Errorf("resp[0].Name = %v, want Folder 1", resp[0].Name)
	}
	if resp[1].Name != "Folder 2" {
		t.Errorf("resp[1].Name = %v, want Folder 2", resp[1].Name)
	}
}

func TestToFileResponse(t *testing.T) {
	folderID := uuid.New()
	storedFileID := uuid.New()
	file := &UserFileWithMeta{
		UserFile: UserFile{
			ID:           uuid.New(),
			FolderID:     &folderID,
			StoredFileID: storedFileID,
			Name:         "document.pdf",
			CreatedAt:    time.Now(),
		},
		SizeBytes: 1048576,
		MimeType:  "application/pdf",
	}

	resp := ToFileResponse(file)

	if resp.ID != file.ID {
		t.Errorf("ID = %v, want %v", resp.ID, file.ID)
	}
	if resp.Name != file.Name {
		t.Errorf("Name = %v, want %v", resp.Name, file.Name)
	}
	if resp.SizeBytes != file.SizeBytes {
		t.Errorf("SizeBytes = %v, want %v", resp.SizeBytes, file.SizeBytes)
	}
	if resp.SizeFormatted != "1 MB" {
		t.Errorf("SizeFormatted = %v, want 1 MB", resp.SizeFormatted)
	}
	if resp.MimeType != file.MimeType {
		t.Errorf("MimeType = %v, want %v", resp.MimeType, file.MimeType)
	}
	if resp.StoredFileID != storedFileID {
		t.Errorf("StoredFileID = %v, want %v", resp.StoredFileID, storedFileID)
	}
}

func TestToFileWithURLResponse(t *testing.T) {
	file := &UserFileWithMeta{
		UserFile: UserFile{
			ID:   uuid.New(),
			Name: "test.pdf",
		},
		SizeBytes: 1024,
	}

	resp := ToFileWithURLResponse(file, "https://example.com/file")

	if resp.URL != "https://example.com/file" {
		t.Errorf("URL = %v, want https://example.com/file", resp.URL)
	}
}

func TestToFilesResponse(t *testing.T) {
	files := []UserFileWithMeta{
		{UserFile: UserFile{ID: uuid.New(), Name: "file1.pdf"}, SizeBytes: 1024},
		{UserFile: UserFile{ID: uuid.New(), Name: "file2.pdf"}, SizeBytes: 2048},
	}

	resp := ToFilesResponse(files)

	if len(resp) != 2 {
		t.Fatalf("len = %d, want 2", len(resp))
	}
	if resp[0].Name != "file1.pdf" {
		t.Errorf("resp[0].Name = %v, want file1.pdf", resp[0].Name)
	}
	if resp[1].Name != "file2.pdf" {
		t.Errorf("resp[1].Name = %v, want file2.pdf", resp[1].Name)
	}
}
