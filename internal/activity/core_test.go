package activity

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidatePublisherType(t *testing.T) {
	tests := []struct {
		name          string
		publisherType string
		want          bool
	}{
		{"university", PublisherUniversity, true},
		{"college", PublisherCollege, true},
		{"department", PublisherDepartment, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidatePublisherType(tt.publisherType); got != tt.want {
				t.Errorf("ValidatePublisherType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePublisherID(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name          string
		publisherType string
		publisherID   *uuid.UUID
		want          bool
	}{
		{"university nil", PublisherUniversity, nil, true},
		{"university with id", PublisherUniversity, &id, false},
		{"college with id", PublisherCollege, &id, true},
		{"college nil", PublisherCollege, nil, false},
		{"department with id", PublisherDepartment, &id, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidatePublisherID(tt.publisherType, tt.publisherID); got != tt.want {
				t.Errorf("ValidatePublisherID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		name         string
		activityType string
		want         bool
	}{
		{"news", TypeNews, true},
		{"announcement", TypeAnnouncement, true},
		{"webinar", TypeWebinar, true},
		{"workshop", TypeWorkshop, true},
		{"conference", TypeConference, true},
		{"symposium", TypeSymposium, true},
		{"training_course", TypeTrainingCourse, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateType(tt.activityType); got != tt.want {
				t.Errorf("ValidateType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateFileType(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		want     bool
	}{
		{"image", FileTypeImage, true},
		{"document", FileTypeDocument, true},
		{"video", FileTypeVideo, true},
		{"invalid", "voice", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateFileType(tt.fileType); got != tt.want {
				t.Errorf("ValidateFileType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateLanguage(t *testing.T) {
	tests := []struct {
		name string
		lang string
		want bool
	}{
		{"en", LangEN, true},
		{"local", LangLocal, true},
		{"invalid", "ar", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateLanguage(tt.lang); got != tt.want {
				t.Errorf("ValidateLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsScheduled(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		publishAt *time.Time
		want      bool
	}{
		{"no publish time", nil, false},
		{"past", &past, false},
		{"future", &future, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsScheduled(tt.publishAt, now); got != tt.want {
				t.Errorf("IsScheduled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsExpired(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{"no expiry", nil, false},
		{"expired", &past, true},
		{"not expired", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsExpired(tt.expiresAt, now); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVisible(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		deletedAt *time.Time
		publishAt *time.Time
		expiresAt *time.Time
		want      bool
	}{
		{"normal", nil, nil, nil, true},
		{"deleted", &now, nil, nil, false},
		{"scheduled", nil, &future, nil, false},
		{"published", nil, &past, nil, true},
		{"expired", nil, nil, &past, false},
		{"not expired", nil, nil, &future, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Activity{DeletedAt: tt.deletedAt, PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := IsVisible(a, now); got != tt.want {
				t.Errorf("IsVisible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanView(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		deletedAt *time.Time
		publishAt *time.Time
		expiresAt *time.Time
		isAdmin   bool
		want      bool
	}{
		{"normal", nil, nil, nil, false, true},
		{"deleted non-admin", &now, nil, nil, false, false},
		{"deleted admin", &now, nil, nil, true, true},
		{"scheduled non-admin", nil, &future, nil, false, false},
		{"scheduled admin", nil, &future, nil, true, true},
		{"expired non-admin", nil, nil, &past, false, false},
		{"expired admin", nil, nil, &past, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Activity{DeletedAt: tt.deletedAt, PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := CanView(a, tt.isAdmin, now); got != tt.want {
				t.Errorf("CanView() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStatus(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		publishAt *time.Time
		expiresAt *time.Time
		want      string
	}{
		{"published", nil, nil, StatusPublished},
		{"scheduled", &future, nil, StatusScheduled},
		{"published past schedule", &past, nil, StatusPublished},
		{"expired", nil, &past, StatusExpired},
		{"not expired", nil, &future, StatusPublished},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Activity{PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := GetStatus(a, now); got != tt.want {
				t.Errorf("GetStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanEdit(t *testing.T) {
	authorID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name    string
		userID  uuid.UUID
		isAdmin bool
		want    bool
	}{
		{"author", authorID, false, true},
		{"admin", otherID, true, true},
		{"other", otherID, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Activity{AuthorID: authorID}
			if got := CanEdit(a, tt.userID, tt.isAdmin); got != tt.want {
				t.Errorf("CanEdit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveTitle(t *testing.T) {
	titleLocal := "Local Title"

	tests := []struct {
		name        string
		titleEN     string
		titleLocal  *string
		prefLang    string
		defaultLang string
		want        string
	}{
		{"prefer en", "English", &titleLocal, LangEN, LangEN, "English"},
		{"prefer local exists", "English", &titleLocal, LangLocal, LangEN, "Local Title"},
		{"prefer local missing", "English", nil, LangLocal, LangEN, "English"},
		{"en empty prefer en falls back to local", "", &titleLocal, LangEN, LangEN, "Local Title"},
		{"en empty prefer local", "", &titleLocal, LangLocal, LangEN, "Local Title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Activity{TitleEN: tt.titleEN, TitleLocal: tt.titleLocal}
			if got := ResolveTitle(a, tt.prefLang, tt.defaultLang); got != tt.want {
				t.Errorf("ResolveTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetTranslation(t *testing.T) {
	titleLocal := "Local Title"
	bodyLocal := "Local Body"

	tests := []struct {
		name       string
		titleEN    string
		bodyEN     string
		titleLocal *string
		bodyLocal  *string
		lang       string
		wantOK     bool
	}{
		{"en with content ok", "EN", "EN Body", nil, nil, LangEN, true},
		{"en empty not ok", "", "", nil, nil, LangEN, false},
		{"local exists", "EN", "EN Body", &titleLocal, &bodyLocal, LangLocal, true},
		{"local missing title", "EN", "EN Body", nil, &bodyLocal, LangLocal, false},
		{"local missing body", "EN", "EN Body", &titleLocal, nil, LangLocal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Activity{TitleEN: tt.titleEN, BodyEN: tt.bodyEN, TitleLocal: tt.titleLocal, BodyLocal: tt.bodyLocal}
			_, _, ok := GetTranslation(a, tt.lang)
			if ok != tt.wantOK {
				t.Errorf("GetTranslation() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestBuildActivity(t *testing.T) {
	authorID := uuid.New()
	publisherID := uuid.New()
	titleLocal := "Local"
	bodyLocal := "Body Local"

	a := BuildActivity(authorID, PublisherCollege, &publisherID, TypeWebinar, "Title", &titleLocal, "Body", &bodyLocal, nil, nil, nil)

	if a.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
	if a.AuthorID != authorID {
		t.Errorf("AuthorID = %v, want %v", a.AuthorID, authorID)
	}
	if a.PublisherType != PublisherCollege {
		t.Errorf("PublisherType = %v, want %v", a.PublisherType, PublisherCollege)
	}
	if a.Type != TypeWebinar {
		t.Errorf("Type = %v, want %v", a.Type, TypeWebinar)
	}
}

func TestBuildActivityAttachment(t *testing.T) {
	activityID := uuid.New()
	fileID := uuid.New()

	a := BuildActivityAttachment(activityID, fileID, "file.pdf", FileTypeDocument, 0)

	if a.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
	if a.ActivityID != activityID {
		t.Errorf("ActivityID = %v, want %v", a.ActivityID, activityID)
	}
	if a.FileType != FileTypeDocument {
		t.Errorf("FileType = %v, want %v", a.FileType, FileTypeDocument)
	}
}
