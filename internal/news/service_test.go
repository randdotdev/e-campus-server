package news

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type mockNewsRepo struct {
	news map[uuid.UUID]*News
}

func newMockNewsRepo() *mockNewsRepo {
	return &mockNewsRepo{news: make(map[uuid.UUID]*News)}
}

func (m *mockNewsRepo) Create(ctx context.Context, n *News) error {
	m.news[n.ID] = n
	return nil
}

func (m *mockNewsRepo) GetByID(ctx context.Context, id uuid.UUID) (*News, error) {
	return m.news[id], nil
}

func (m *mockNewsRepo) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*NewsWithAuthor, error) {
	n := m.news[id]
	if n == nil {
		return nil, nil
	}
	return &NewsWithAuthor{News: *n, AuthorName: "Test User"}, nil
}

func (m *mockNewsRepo) Update(ctx context.Context, n *News) error {
	m.news[n.ID] = n
	return nil
}

func (m *mockNewsRepo) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	if n := m.news[id]; n != nil {
		n.DeletedAt = &deletedAt
	}
	return nil
}

func (m *mockNewsRepo) ListByPublisher(ctx context.Context, publisherType string, publisherID *uuid.UUID, category string, isAdmin bool, params pagination.PageParams) ([]NewsWithAuthor, bool, error) {
	var result []NewsWithAuthor
	for _, n := range m.news {
		if n.PublisherType == publisherType && n.DeletedAt == nil {
			result = append(result, NewsWithAuthor{News: *n, AuthorName: "Test User"})
		}
	}
	return result, false, nil
}

type mockAttachmentRepo struct {
	attachments map[uuid.UUID]*NewsAttachment
}

func newMockAttachmentRepo() *mockAttachmentRepo {
	return &mockAttachmentRepo{attachments: make(map[uuid.UUID]*NewsAttachment)}
}

func (m *mockAttachmentRepo) Create(ctx context.Context, a *NewsAttachment) error {
	m.attachments[a.ID] = a
	return nil
}

func (m *mockAttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.attachments, id)
	return nil
}

func (m *mockAttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*NewsAttachment, error) {
	return m.attachments[id], nil
}

func (m *mockAttachmentRepo) ListByNewsID(ctx context.Context, newsID uuid.UUID) ([]NewsAttachment, error) {
	var result []NewsAttachment
	for _, a := range m.attachments {
		if a.NewsID == newsID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockAttachmentRepo) ListByNewsIDs(ctx context.Context, newsIDs []uuid.UUID) (map[uuid.UUID][]NewsAttachment, error) {
	result := make(map[uuid.UUID][]NewsAttachment)
	for _, a := range m.attachments {
		for _, newsID := range newsIDs {
			if a.NewsID == newsID {
				result[newsID] = append(result[newsID], *a)
			}
		}
	}
	return result, nil
}

type mockPublisherChecker struct {
	publishers map[string]bool
}

func newMockPublisherChecker() *mockPublisherChecker {
	return &mockPublisherChecker{publishers: make(map[string]bool)}
}

func (m *mockPublisherChecker) PublisherExists(ctx context.Context, publisherType string, publisherID uuid.UUID) (bool, error) {
	key := publisherType + ":" + publisherID.String()
	if exists, ok := m.publishers[key]; ok {
		return exists, nil
	}
	return true, nil
}

type mockSettingsProvider struct {
	defaultLang string
}

func newMockSettingsProvider() *mockSettingsProvider {
	return &mockSettingsProvider{defaultLang: LangEN}
}

func (m *mockSettingsProvider) GetDefaultLanguage(ctx context.Context) (string, error) {
	return m.defaultLang, nil
}

func newTestService() *Service {
	return NewService(
		newMockNewsRepo(),
		newMockAttachmentRepo(),
		newMockPublisherChecker(),
		newMockSettingsProvider(),
	)
}

func ptr[T any](v T) *T {
	return &v
}

func TestCreateNews(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	tests := []struct {
		name          string
		publisherType string
		publisherID   *uuid.UUID
		category      string
		wantErr       error
	}{
		{"university news", PublisherUniversity, nil, CategoryAnnouncement, nil},
		{"college news", PublisherCollege, ptr(uuid.New()), CategoryEvent, nil},
		{"department news", PublisherDepartment, ptr(uuid.New()), CategoryAchievement, nil},
		{"invalid publisher type", "invalid", nil, CategoryGeneral, ErrInvalidPublisher},
		{"college without id", PublisherCollege, nil, CategoryGeneral, ErrInvalidPublisher},
		{"university with id", PublisherUniversity, ptr(uuid.New()), CategoryGeneral, ErrInvalidPublisher},
		{"invalid category", PublisherUniversity, nil, "invalid", ErrInvalidCategory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			news, err := s.CreateNews(ctx, authorID, tt.publisherType, tt.publisherID, tt.category,
				"Title EN", nil, "Body EN", nil, nil, nil, nil)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("CreateNews() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("CreateNews() unexpected error: %v", err)
			}
			if news.PublisherType != tt.publisherType {
				t.Errorf("PublisherType = %v, want %v", news.PublisherType, tt.publisherType)
			}
			if news.Category != tt.category {
				t.Errorf("Category = %v, want %v", news.Category, tt.category)
			}
		})
	}
}

func TestGetNews(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	news, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"Title", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name    string
		id      uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"existing news", news.ID, false, nil},
		{"non-existing news", uuid.New(), false, ErrNewsNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := s.GetNews(ctx, tt.id, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetNews() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetNews() unexpected error: %v", err)
			}
			if result.ID != tt.id {
				t.Errorf("ID = %v, want %v", result.ID, tt.id)
			}
		})
	}
}

func TestGetNewsScheduledExpired(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	scheduled, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"Scheduled", nil, "Body", nil, nil, &future, nil)
	expired, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"Expired", nil, "Body", nil, nil, nil, &past)

	tests := []struct {
		name    string
		id      uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"scheduled non-admin", scheduled.ID, false, ErrNewsNotFound},
		{"scheduled admin", scheduled.ID, true, nil},
		{"expired non-admin", expired.ID, false, ErrNewsNotFound},
		{"expired admin", expired.ID, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := s.GetNews(ctx, tt.id, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetNews() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetNews() unexpected error: %v", err)
			}
		})
	}
}

func TestServiceGetTranslation(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	titleLocal := "Local Title"
	bodyLocal := "Local Body"

	withLocal, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"EN Title", &titleLocal, "EN Body", &bodyLocal, nil, nil, nil)
	withoutLocal, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"EN Title", nil, "EN Body", nil, nil, nil, nil)

	tests := []struct {
		name    string
		id      uuid.UUID
		lang    string
		wantErr error
	}{
		{"en translation", withLocal.ID, LangEN, nil},
		{"local translation exists", withLocal.ID, LangLocal, nil},
		{"local translation missing", withoutLocal.ID, LangLocal, ErrTranslationMissing},
		{"invalid language", withLocal.ID, "ar", ErrInvalidLanguage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := s.GetTranslation(ctx, tt.id, tt.lang, false)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetTranslation() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetTranslation() unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateNews(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	news, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"Original", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name    string
		userID  uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"author can edit", authorID, false, nil},
		{"admin can edit", otherID, true, nil},
		{"other cannot edit", otherID, false, ErrNotAuthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newTitle := "Updated"
			_, err := s.UpdateNews(ctx, news.ID, tt.userID, tt.isAdmin, &newTitle, nil, nil, nil, nil, nil, nil, nil)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("UpdateNews() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateNews() unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateNewsCategory(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	news, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"Title", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name     string
		category string
		wantErr  error
	}{
		{"valid category", CategoryEvent, nil},
		{"invalid category", "invalid", ErrInvalidCategory},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.UpdateNews(ctx, news.ID, authorID, false, nil, nil, nil, nil, &tt.category, nil, nil, nil)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("UpdateNews() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateNews() unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteNews(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name    string
		userID  uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"author can delete", authorID, false, nil},
		{"admin can delete", otherID, true, nil},
		{"other cannot delete", otherID, false, ErrNotAuthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			news, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
				"To delete", nil, "Body", nil, nil, nil, nil)
			err := s.DeleteNews(ctx, news.ID, tt.userID, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("DeleteNews() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("DeleteNews() unexpected error: %v", err)
			}
		})
	}
}

func TestPinNews(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	news, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"Pinnable", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name    string
		isAdmin bool
		wantErr error
	}{
		{"admin can pin", true, nil},
		{"non-admin cannot pin", false, ErrNotAuthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.PinNews(ctx, news.ID, tt.isAdmin, true)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("PinNews() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("PinNews() unexpected error: %v", err)
			}
		})
	}
}

func TestAddAttachment(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	news, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"With attachment", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name     string
		userID   uuid.UUID
		isAdmin  bool
		fileType string
		wantErr  error
	}{
		{"author can add image", authorID, false, FileTypeImage, nil},
		{"admin can add document", otherID, true, FileTypeDocument, nil},
		{"other cannot add", otherID, false, FileTypeImage, ErrNotAuthorized},
		{"invalid file type", authorID, false, "invalid", ErrInvalidFileType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.AddAttachment(ctx, news.ID, tt.userID, tt.isAdmin, uuid.New(), "test.jpg", tt.fileType, 0)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("AddAttachment() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("AddAttachment() unexpected error: %v", err)
			}
		})
	}
}

func TestRemoveAttachment(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	news, _ := s.CreateNews(ctx, authorID, PublisherUniversity, nil, CategoryAnnouncement,
		"With attachment", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name    string
		userID  uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"author can remove", authorID, false, nil},
		{"admin can remove", otherID, true, nil},
		{"other cannot remove", otherID, false, ErrNotAuthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attachment, _ := s.AddAttachment(ctx, news.ID, authorID, false, uuid.New(), "test.jpg", FileTypeImage, 0)
			err := s.RemoveAttachment(ctx, attachment.ID, tt.userID, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("RemoveAttachment() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("RemoveAttachment() unexpected error: %v", err)
			}
		})
	}
}
