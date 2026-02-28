package news

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToNewsResponse(t *testing.T) {
	now := time.Now()
	titleLocal := "عنوان"
	bodyLocal := "محتوى"
	authorNameLocal := "اسم المؤلف"
	avatar := "https://example.com/avatar.jpg"
	publisherID := uuid.New()

	news := &NewsWithAuthor{
		News: News{
			ID:            uuid.New(),
			PublisherType: PublisherCollege,
			PublisherID:   &publisherID,
			Category:      CategoryAnnouncement,
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

	attachments := []NewsAttachment{
		{ID: uuid.New(), NewsID: news.ID, DisplayName: "file.pdf", FileType: FileTypeDocument},
	}

	resp := ToNewsResponse(news, attachments, LangEN, LangEN, now)

	if resp.ID != news.ID {
		t.Error("ID should match")
	}
	if resp.PublisherType != PublisherCollege {
		t.Error("PublisherType should match")
	}
	if resp.Category != CategoryAnnouncement {
		t.Error("Category should match")
	}
	if resp.Title != "Title EN" {
		t.Errorf("Title should be EN, got %s", resp.Title)
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

func TestToNewsResponseWithLocalLang(t *testing.T) {
	titleLocal := "عنوان"
	bodyLocal := "محتوى"

	news := &NewsWithAuthor{
		News: News{
			ID:         uuid.New(),
			TitleEN:    "Title EN",
			TitleLocal: &titleLocal,
			BodyEN:     "Body EN",
			BodyLocal:  &bodyLocal,
			CreatedAt:  time.Now(),
		},
		AuthorName: "Author",
	}

	resp := ToNewsResponse(news, nil, LangLocal, LangEN, time.Now())

	if resp.Title != titleLocal {
		t.Errorf("Title should be local when preferring local, got %s", resp.Title)
	}
	if resp.Body != bodyLocal {
		t.Errorf("Body should be local when preferring local, got %s", resp.Body)
	}
}

func TestToNewsResponses(t *testing.T) {
	newsList := []NewsWithAuthor{
		{News: News{ID: uuid.New(), TitleEN: "First", BodyEN: "Body1", CreatedAt: time.Now()}, AuthorName: "A"},
		{News: News{ID: uuid.New(), TitleEN: "Second", BodyEN: "Body2", CreatedAt: time.Now()}, AuthorName: "B"},
	}

	attachmentsMap := make(map[uuid.UUID][]NewsAttachment)
	attachmentsMap[newsList[0].ID] = []NewsAttachment{
		{ID: uuid.New(), NewsID: newsList[0].ID, DisplayName: "file.jpg", FileType: FileTypeImage},
	}

	result := ToNewsResponses(newsList, attachmentsMap, LangEN, LangEN, time.Now())

	if len(result) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(result))
	}
	if len(result[0].Attachments) != 1 {
		t.Error("first news should have 1 attachment")
	}
	if result[1].Attachments != nil {
		t.Error("second news should have no attachments")
	}
}

func TestToAttachmentResponse(t *testing.T) {
	attachment := &NewsAttachment{
		ID:           uuid.New(),
		NewsID:       uuid.New(),
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
	if ToAttachmentResponses([]NewsAttachment{}) != nil {
		t.Error("empty input should return nil")
	}

	attachments := []NewsAttachment{
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
