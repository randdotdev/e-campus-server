package qa

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{"valid", "How do I solve problem 5?", nil},
		{"empty", "", ErrEmptyTitle},
		{"too long", strings.Repeat("a", MaxTitleLength+1), ErrTitleTooLong},
		{"max length", strings.Repeat("a", MaxTitleLength), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTitle(tt.title)
			if err != tt.wantErr {
				t.Errorf("ValidateTitle() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBody(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr error
	}{
		{"valid", "This is my question about the homework.", nil},
		{"empty", "", ErrEmptyBody},
		{"whitespace only counts", "   ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBody(tt.body)
			if err != tt.wantErr {
				t.Errorf("ValidateBody() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCanEditQuestion(t *testing.T) {
	author := uuid.New()
	other := uuid.New()

	q := &Question{CreatedBy: author}

	tests := []struct {
		name      string
		userID    uuid.UUID
		isTeacher bool
		want      bool
	}{
		{"author can edit", author, false, true},
		{"other cannot edit", other, false, false},
		{"teacher can edit", other, true, true},
		{"author teacher can edit", author, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanEditQuestion(q, tt.userID, tt.isTeacher)
			if got != tt.want {
				t.Errorf("CanEditQuestion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanDeleteQuestion(t *testing.T) {
	author := uuid.New()
	other := uuid.New()

	tests := []struct {
		name   string
		status string
		isFAQ  bool
		userID uuid.UUID
		want   bool
	}{
		{"author pending", StatusPending, false, author, true},
		{"author answered", StatusAnswered, false, author, false},
		{"author rejected", StatusRejected, false, author, true},
		{"author faq", StatusAnswered, true, author, true},
		{"other pending", StatusPending, false, other, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &Question{CreatedBy: author, Status: tt.status, IsFAQ: tt.isFAQ}
			got := CanDeleteQuestion(q, tt.userID)
			if got != tt.want {
				t.Errorf("CanDeleteQuestion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanViewQuestion(t *testing.T) {
	author := uuid.New()
	other := uuid.New()
	now := time.Now()
	deletedAt := &now

	tests := []struct {
		name      string
		status    string
		deletedAt *time.Time
		userID    uuid.UUID
		isTeacher bool
		want      bool
	}{
		{"answered visible to all", StatusAnswered, nil, other, false, true},
		{"pending visible to author", StatusPending, nil, author, false, true},
		{"pending not visible to other", StatusPending, nil, other, false, false},
		{"pending visible to teacher", StatusPending, nil, other, true, true},
		{"rejected visible to author", StatusRejected, nil, author, false, true},
		{"rejected not visible to other", StatusRejected, nil, other, false, false},
		{"deleted not visible", StatusAnswered, deletedAt, author, false, false},
		{"deleted not visible to teacher", StatusAnswered, deletedAt, author, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &Question{
				CreatedBy: author,
				Status:    tt.status,
				DeletedAt: tt.deletedAt,
			}
			got := CanViewQuestion(q, tt.userID, tt.isTeacher)
			if got != tt.want {
				t.Errorf("CanViewQuestion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildQuestion(t *testing.T) {
	offeringID := uuid.New()
	userID := uuid.New()

	q := BuildQuestion(offeringID, userID, "Test Title", "Test Body", true, false)

	if q.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if q.OfferingID != offeringID {
		t.Error("OfferingID mismatch")
	}
	if q.CreatedBy != userID {
		t.Error("CreatedBy mismatch")
	}
	if q.Title != "Test Title" {
		t.Error("Title mismatch")
	}
	if q.Body != "Test Body" {
		t.Error("Body mismatch")
	}
	if !q.IsAnonymous {
		t.Error("IsAnonymous should be true")
	}
	if q.IsFAQ {
		t.Error("IsFAQ should be false")
	}
	if q.Status != StatusPending {
		t.Error("Status should be pending")
	}
}

func TestBuildAnswer(t *testing.T) {
	questionID := uuid.New()
	teacherID := uuid.New()

	a := BuildAnswer(questionID, teacherID, "Test Answer")

	if a.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if a.QuestionID != questionID {
		t.Error("QuestionID mismatch")
	}
	if a.CreatedBy != teacherID {
		t.Error("CreatedBy mismatch")
	}
	if a.Body != "Test Answer" {
		t.Error("Body mismatch")
	}
}
