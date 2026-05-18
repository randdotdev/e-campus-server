package activity

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type mockActivityRepo struct {
	activities map[uuid.UUID]*Activity
}

func newMockActivityRepo() *mockActivityRepo {
	return &mockActivityRepo{activities: make(map[uuid.UUID]*Activity)}
}

func (m *mockActivityRepo) Create(ctx context.Context, a *Activity) error {
	m.activities[a.ID] = a
	return nil
}

func (m *mockActivityRepo) GetByID(ctx context.Context, id uuid.UUID) (*Activity, error) {
	return m.activities[id], nil
}

func (m *mockActivityRepo) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*ActivityWithAuthor, error) {
	a := m.activities[id]
	if a == nil {
		return nil, nil
	}
	return &ActivityWithAuthor{Activity: *a, AuthorName: "Test User"}, nil
}

func (m *mockActivityRepo) Update(ctx context.Context, a *Activity) error {
	m.activities[a.ID] = a
	return nil
}

func (m *mockActivityRepo) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	if a := m.activities[id]; a != nil {
		a.DeletedAt = &deletedAt
	}
	return nil
}

func (m *mockActivityRepo) ListByPublisher(ctx context.Context, publisherType string, publisherID *uuid.UUID, activityType string, isAdmin bool, params pagination.PageParams) ([]ActivityWithAuthor, bool, error) {
	var result []ActivityWithAuthor
	for _, a := range m.activities {
		if a.PublisherType == publisherType && a.DeletedAt == nil {
			result = append(result, ActivityWithAuthor{Activity: *a, AuthorName: "Test User"})
		}
	}
	return result, false, nil
}

type mockAttachmentRepo struct {
	attachments map[uuid.UUID]*ActivityAttachment
}

func newMockAttachmentRepo() *mockAttachmentRepo {
	return &mockAttachmentRepo{attachments: make(map[uuid.UUID]*ActivityAttachment)}
}

func (m *mockAttachmentRepo) Create(ctx context.Context, a *ActivityAttachment) error {
	m.attachments[a.ID] = a
	return nil
}

func (m *mockAttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.attachments, id)
	return nil
}

func (m *mockAttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*ActivityAttachment, error) {
	return m.attachments[id], nil
}

func (m *mockAttachmentRepo) ListByActivityID(ctx context.Context, activityID uuid.UUID) ([]ActivityAttachment, error) {
	var result []ActivityAttachment
	for _, a := range m.attachments {
		if a.ActivityID == activityID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockAttachmentRepo) ListByActivityIDs(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID][]ActivityAttachment, error) {
	result := make(map[uuid.UUID][]ActivityAttachment)
	for _, a := range m.attachments {
		for _, id := range activityIDs {
			if a.ActivityID == id {
				result[id] = append(result[id], *a)
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
		newMockActivityRepo(),
		newMockAttachmentRepo(),
		newMockPublisherChecker(),
		newMockSettingsProvider(),
	)
}

func ptr[T any](v T) *T {
	return &v
}

func TestCreateActivity(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	tests := []struct {
		name          string
		publisherType string
		publisherID   *uuid.UUID
		activityType  string
		wantErr       error
	}{
		{"university news", PublisherUniversity, nil, TypeNews, nil},
		{"college webinar", PublisherCollege, ptr(uuid.New()), TypeWebinar, nil},
		{"department workshop", PublisherDepartment, ptr(uuid.New()), TypeWorkshop, nil},
		{"invalid publisher type", "invalid", nil, TypeNews, ErrInvalidPublisher},
		{"college without id", PublisherCollege, nil, TypeNews, ErrInvalidPublisher},
		{"university with id", PublisherUniversity, ptr(uuid.New()), TypeNews, ErrInvalidPublisher},
		{"invalid type", PublisherUniversity, nil, "invalid", ErrInvalidType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := s.CreateActivity(ctx, authorID, tt.publisherType, tt.publisherID, tt.activityType,
				"Title EN", nil, "Body EN", nil, nil, nil, nil)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("CreateActivity() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("CreateActivity() unexpected error: %v", err)
			}
			if a.PublisherType != tt.publisherType {
				t.Errorf("PublisherType = %v, want %v", a.PublisherType, tt.publisherType)
			}
			if a.Type != tt.activityType {
				t.Errorf("Type = %v, want %v", a.Type, tt.activityType)
			}
		})
	}
}

func TestGetActivity(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	a, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
		"Title", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name    string
		id      uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"existing activity", a.ID, false, nil},
		{"non-existing activity", uuid.New(), false, ErrActivityNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := s.GetActivity(ctx, tt.id, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetActivity() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetActivity() unexpected error: %v", err)
			}
			if result.ID != tt.id {
				t.Errorf("ID = %v, want %v", result.ID, tt.id)
			}
		})
	}
}

func TestGetActivityScheduledExpired(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	scheduled, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
		"Scheduled", nil, "Body", nil, nil, &future, nil)
	expired, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
		"Expired", nil, "Body", nil, nil, nil, &past)

	tests := []struct {
		name    string
		id      uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"scheduled non-admin", scheduled.ID, false, ErrActivityNotFound},
		{"scheduled admin", scheduled.ID, true, nil},
		{"expired non-admin", expired.ID, false, ErrActivityNotFound},
		{"expired admin", expired.ID, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := s.GetActivity(ctx, tt.id, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("GetActivity() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("GetActivity() unexpected error: %v", err)
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

	withLocal, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
		"EN Title", &titleLocal, "EN Body", &bodyLocal, nil, nil, nil)
	withoutLocal, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
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

func TestUpdateActivity(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	a, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
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
			_, err := s.UpdateActivity(ctx, a.ID, tt.userID, tt.isAdmin, &newTitle, nil, nil, nil, nil, nil, nil, nil)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("UpdateActivity() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateActivity() unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateActivityType(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	a, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
		"Title", nil, "Body", nil, nil, nil, nil)

	tests := []struct {
		name         string
		activityType string
		wantErr      error
	}{
		{"valid type", TypeConference, nil},
		{"invalid type", "invalid", ErrInvalidType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.UpdateActivity(ctx, a.ID, authorID, false, nil, nil, nil, nil, &tt.activityType, nil, nil, nil)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("UpdateActivity() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateActivity() unexpected error: %v", err)
			}
		})
	}
}

func TestDeleteActivity(t *testing.T) {
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
			a, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
				"To delete", nil, "Body", nil, nil, nil, nil)
			err := s.DeleteActivity(ctx, a.ID, tt.userID, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("DeleteActivity() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("DeleteActivity() unexpected error: %v", err)
			}
		})
	}
}

func TestPinActivity(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	a, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
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
			err := s.PinActivity(ctx, a.ID, tt.isAdmin, true)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("PinActivity() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("PinActivity() unexpected error: %v", err)
			}
		})
	}
}

func TestAddAttachment(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	a, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
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
			_, err := s.AddAttachment(ctx, a.ID, tt.userID, tt.isAdmin, uuid.New(), "test.jpg", tt.fileType, 0)

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

	a, _ := s.CreateActivity(ctx, authorID, PublisherUniversity, nil, TypeAnnouncement,
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
			attachment, _ := s.AddAttachment(ctx, a.ID, authorID, false, uuid.New(), "test.jpg", FileTypeImage, 0)
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
