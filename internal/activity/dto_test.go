package activity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToActivityResponse(t *testing.T) {
	now := time.Now()
	titleLocal := "عنوان"
	bodyLocal := "محتوى"
	authorNameLocal := "اسم المؤلف"
	avatar := "https://example.com/avatar.jpg"
	publisherID := uuid.New()

	a := &ActivityWithAuthor{
		Activity: Activity{
			ID:            uuid.New(),
			PublisherType: PublisherCollege,
			PublisherID:   &publisherID,
			Type:          TypeAnnouncement,
			TitleEN:       "Title EN",
			TitleLocal:    &titleLocal,
			BodyEN:        "Body EN",
			BodyLocal:     &bodyLocal,
			IsPinned:      true,
			CreatedAt:     now,
		},
		AuthorName:      "Author Name",
		AuthorNameLocal: &authorNameLocal,
		AuthorAvatar:    &avatar,
	}

	attachments := []ActivityAttachment{
		{ID: uuid.New(), ActivityID: a.ID, DisplayName: "file.pdf", FileType: FileTypeDocument},
	}

	resp := ToActivityResponse(a, attachments, now)

	if resp.ID != a.ID {
		t.Error("ID should match")
	}
	if resp.PublisherType != PublisherCollege {
		t.Error("PublisherType should match")
	}
	if resp.Type != TypeAnnouncement {
		t.Error("Type should match")
	}
	if resp.TitleEN != "Title EN" {
		t.Errorf("TitleEN should match, got %s", resp.TitleEN)
	}
	if resp.TitleLocal == nil || *resp.TitleLocal != titleLocal {
		t.Error("TitleLocal should match")
	}
	if resp.BodyEN != "Body EN" {
		t.Errorf("BodyEN should match, got %s", resp.BodyEN)
	}
	if resp.BodyLocal == nil || *resp.BodyLocal != bodyLocal {
		t.Error("BodyLocal should match")
	}
	if resp.AuthorName != "Author Name" {
		t.Error("AuthorName should match")
	}
	if resp.AuthorNameLocal == nil || *resp.AuthorNameLocal != authorNameLocal {
		t.Error("AuthorNameLocal should match")
	}
	if len(resp.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(resp.Attachments))
	}
	if resp.Status != StatusPublished {
		t.Errorf("Status should be published, got %s", resp.Status)
	}
}

func TestToActivityResponses(t *testing.T) {
	activities := []ActivityWithAuthor{
		{Activity: Activity{ID: uuid.New(), TitleEN: "First", BodyEN: "Body1", CreatedAt: time.Now()}, AuthorName: "A"},
		{Activity: Activity{ID: uuid.New(), TitleEN: "Second", BodyEN: "Body2", CreatedAt: time.Now()}, AuthorName: "B"},
	}

	attachmentsMap := make(map[uuid.UUID][]ActivityAttachment)
	attachmentsMap[activities[0].ID] = []ActivityAttachment{
		{ID: uuid.New(), ActivityID: activities[0].ID, DisplayName: "file.jpg", FileType: FileTypeImage},
	}

	result := ToActivityResponses(activities, attachmentsMap, time.Now())

	if len(result) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(result))
	}
	if len(result[0].Attachments) != 1 {
		t.Error("first activity should have 1 attachment")
	}
	if result[1].Attachments != nil {
		t.Error("second activity should have no attachments")
	}
}

func TestToAttachmentResponse(t *testing.T) {
	attachment := &ActivityAttachment{
		ID:           uuid.New(),
		ActivityID:   uuid.New(),
		StoredFileID: uuid.New(),
		DisplayName:  "document.pdf",
		FileType:     FileTypeDocument,
		OrderIndex:   2,
	}

	resp := ToAttachmentResponse(attachment)

	if resp.ID != attachment.ID {
		t.Error("ID should match")
	}
	if resp.StoredFileID != attachment.StoredFileID {
		t.Error("StoredFileID should match")
	}
	if resp.DisplayName != "document.pdf" {
		t.Error("DisplayName should match")
	}
	if resp.FileType != FileTypeDocument {
		t.Error("FileType should match")
	}
	if resp.OrderIndex != 2 {
		t.Error("OrderIndex should match")
	}
}

func TestToAttachmentResponses(t *testing.T) {
	if ToAttachmentResponses(nil) != nil {
		t.Error("nil input should return nil")
	}
	if ToAttachmentResponses([]ActivityAttachment{}) != nil {
		t.Error("empty input should return nil")
	}

	attachments := []ActivityAttachment{
		{ID: uuid.New(), DisplayName: "a.jpg", FileType: FileTypeImage},
		{ID: uuid.New(), DisplayName: "b.pdf", FileType: FileTypeDocument},
	}

	result := ToAttachmentResponses(attachments)

	if len(result) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(result))
	}
	if result[0].DisplayName != "a.jpg" {
		t.Error("first attachment name should match")
	}
	if result[1].DisplayName != "b.pdf" {
		t.Error("second attachment name should match")
	}
}
