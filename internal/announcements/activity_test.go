package announcements

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// activityMockRepo is an in-memory ActivityRepository.
type activityMockRepo struct {
	activities  map[uuid.UUID]*Activity
	attachments map[uuid.UUID]*ActivityAttachment
}

func newActivityMockRepo() *activityMockRepo {
	return &activityMockRepo{
		activities:  make(map[uuid.UUID]*Activity),
		attachments: make(map[uuid.UUID]*ActivityAttachment),
	}
}

func (m *activityMockRepo) CreateActivity(ctx context.Context, a *Activity) error {
	m.activities[a.ID] = a
	return nil
}

func (m *activityMockRepo) GetActivityByID(ctx context.Context, id uuid.UUID) (*Activity, error) {
	return m.activities[id], nil
}

func (m *activityMockRepo) GetActivityByIDWithAuthor(ctx context.Context, id uuid.UUID) (*ActivityWithAuthor, error) {
	a := m.activities[id]
	if a == nil {
		return nil, nil
	}
	return &ActivityWithAuthor{Activity: *a, AuthorName: "Test User"}, nil
}

func (m *activityMockRepo) UpdateActivity(ctx context.Context, a *Activity, expectedVersion int64) (int64, error) {
	existing := m.activities[a.ID]
	if existing != nil && existing.Version != expectedVersion {
		return 0, ErrConflict
	}
	a.Version = expectedVersion + 1
	m.activities[a.ID] = a
	return a.Version, nil
}

func (m *activityMockRepo) SoftDeleteActivity(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	if a := m.activities[id]; a != nil {
		a.DeletedAt = &deletedAt
	}
	return nil
}

func (m *activityMockRepo) ListActivitiesByPublisher(ctx context.Context, pt PublisherType, publisherID *uuid.UUID, activityType ActivityType, isAdmin bool, params pagination.PageParams) ([]ActivityWithAuthor, bool, error) {
	var result []ActivityWithAuthor
	for _, a := range m.activities {
		if a.PublisherType == pt && a.DeletedAt == nil {
			result = append(result, ActivityWithAuthor{Activity: *a})
		}
	}
	return result, false, nil
}

func (m *activityMockRepo) CreateAttachment(ctx context.Context, a *ActivityAttachment) error {
	m.attachments[a.ID] = a
	return nil
}

func (m *activityMockRepo) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	delete(m.attachments, id)
	return nil
}

func (m *activityMockRepo) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*ActivityAttachment, error) {
	return m.attachments[id], nil
}

func (m *activityMockRepo) ListAttachmentsByActivityID(ctx context.Context, activityID uuid.UUID) ([]ActivityAttachment, error) {
	var result []ActivityAttachment
	for _, a := range m.attachments {
		if a.ActivityID == activityID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *activityMockRepo) ListAttachmentsByActivityIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]ActivityAttachment, error) {
	result := make(map[uuid.UUID][]ActivityAttachment)
	for _, a := range m.attachments {
		for _, id := range ids {
			if a.ActivityID == id {
				result[id] = append(result[id], *a)
			}
		}
	}
	return result, nil
}

type okPublisherChecker struct{}

func (okPublisherChecker) PublisherExists(ctx context.Context, pt PublisherType, id uuid.UUID) (bool, error) {
	return true, nil
}

type enLangProvider struct{}

func (enLangProvider) GetDefaultLanguage(ctx context.Context) (string, error) { return "en", nil }

func newActivityService() *ActivityService {
	return NewActivityService(newActivityMockRepo(), okPublisherChecker{}, enLangProvider{}, stubFiles{}, slog.New(slog.DiscardHandler))
}

func ptr[T any](v T) *T { return &v }

func TestActivityCreate(t *testing.T) {
	s := newActivityService()
	ctx := context.Background()
	author := uuid.New()

	tests := []struct {
		name    string
		pt      PublisherType
		pid     *uuid.UUID
		typ     ActivityType
		wantErr error
	}{
		{"university news", PublisherUniversity, nil, TypeNews, nil},
		{"college webinar", PublisherCollege, ptr(uuid.New()), TypeWebinar, nil},
		{"invalid publisher", "invalid", nil, TypeNews, ErrInvalidPublisher},
		{"college without id", PublisherCollege, nil, TypeNews, ErrInvalidPublisher},
		{"university with id", PublisherUniversity, ptr(uuid.New()), TypeNews, ErrInvalidPublisher},
		{"invalid type", PublisherUniversity, nil, "invalid", ErrInvalidType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := s.Create(ctx, CreateActivityInput{
				AuthorID: author, PublisherType: tt.pt, PublisherID: tt.pid, Type: tt.typ,
				TitleEN: "Title", BodyEN: "Body",
			})
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if a.Type != tt.typ {
				t.Errorf("type = %v, want %v", a.Type, tt.typ)
			}
		})
	}
}

func TestActivityGetVisibility(t *testing.T) {
	s := newActivityService()
	ctx := context.Background()
	author := uuid.New()
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	scheduled, _ := s.Create(ctx, CreateActivityInput{AuthorID: author, PublisherType: PublisherUniversity, Type: TypeNews, TitleEN: "s", BodyEN: "b", PublishAt: &future})
	expired, _ := s.Create(ctx, CreateActivityInput{AuthorID: author, PublisherType: PublisherUniversity, Type: TypeNews, TitleEN: "e", BodyEN: "b", ExpiresAt: &past})

	cases := []struct {
		id           uuid.UUID
		revealHidden bool
		wantErr      error
	}{
		{scheduled.ID, false, ErrActivityNotFound},
		{scheduled.ID, true, nil},
		{expired.ID, false, ErrActivityNotFound},
		{expired.ID, true, nil},
	}
	for _, tc := range cases {
		if _, _, err := s.Get(ctx, tc.id, tc.revealHidden); err != tc.wantErr {
			t.Errorf("Get(revealHidden=%v) err = %v, want %v", tc.revealHidden, err, tc.wantErr)
		}
	}
}

func TestActivityTranslate(t *testing.T) {
	s := newActivityService()
	ctx := context.Background()
	author := uuid.New()

	withLocal, _ := s.Create(ctx, CreateActivityInput{AuthorID: author, PublisherType: PublisherUniversity, Type: TypeNews, TitleEN: "EN", TitleLocal: ptr("LO"), BodyEN: "ENB", BodyLocal: ptr("LOB")})
	withoutLocal, _ := s.Create(ctx, CreateActivityInput{AuthorID: author, PublisherType: PublisherUniversity, Type: TypeNews, TitleEN: "EN", BodyEN: "ENB"})

	cases := []struct {
		id      uuid.UUID
		lang    Lang
		wantErr error
	}{
		{withLocal.ID, LangEN, nil},
		{withLocal.ID, LangLocal, nil},
		{withoutLocal.ID, LangLocal, ErrTranslationMissing},
		{withLocal.ID, "ar", ErrInvalidLanguage},
	}
	for _, tc := range cases {
		if _, _, err := s.Translate(ctx, tc.id, tc.lang, false); err != tc.wantErr {
			t.Errorf("Translate(%v) err = %v, want %v", tc.lang, err, tc.wantErr)
		}
	}
}

// Update/Delete/Pin authorization moved to the edge (the gate and handler
// checks); the service executes for an already-entitled caller (§21).
func TestActivityUpdateAndDelete(t *testing.T) {
	s := newActivityService()
	ctx := context.Background()
	author := uuid.New()
	a, _ := s.Create(ctx, CreateActivityInput{AuthorID: author, PublisherType: PublisherUniversity, Type: TypeNews, TitleEN: "T", BodyEN: "B"})

	if _, err := s.Update(ctx, UpdateActivityInput{ID: a.ID, ActorID: author, TitleEN: ptr("New")}); err != nil {
		t.Errorf("Update = %v", err)
	}
	if err := s.Pin(ctx, a.ID, true); err != nil {
		t.Errorf("Pin = %v", err)
	}
	if err := s.Delete(ctx, a.ID); err != nil {
		t.Errorf("Delete = %v", err)
	}
}

func TestActivityAttachments(t *testing.T) {
	s := newActivityService()
	ctx := context.Background()
	author := uuid.New()
	a, _ := s.Create(ctx, CreateActivityInput{AuthorID: author, PublisherType: PublisherUniversity, Type: TypeNews, TitleEN: "T", BodyEN: "B"})

	// Attach authority moved to the edge; the service only validates the file.
	if _, err := s.AddAttachment(ctx, AddActivityAttachmentInput{ActivityID: a.ID, ActorID: author, FileType: "invalid", UploadID: uuid.New()}); err != ErrInvalidFileType {
		t.Errorf("AddAttachment(bad type) = %v, want ErrInvalidFileType", err)
	}
	att, err := s.AddAttachment(ctx, AddActivityAttachmentInput{ActivityID: a.ID, ActorID: author, FileType: FileTypeImage, UploadID: uuid.New()})
	if err != nil {
		t.Fatalf("AddAttachment(author) = %v", err)
	}
	if err := s.RemoveAttachment(ctx, att.ID); err != nil {
		t.Errorf("RemoveAttachment(author) = %v", err)
	}
}
