package news

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

func TestValidateCategory(t *testing.T) {
	tests := []struct {
		name     string
		category string
		want     bool
	}{
		{"announcement", CategoryAnnouncement, true},
		{"event", CategoryEvent, true},
		{"achievement", CategoryAchievement, true},
		{"academic", CategoryAcademic, true},
		{"general", CategoryGeneral, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateCategory(tt.category); got != tt.want {
				t.Errorf("ValidateCategory() = %v, want %v", got, tt.want)
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
			n := &News{DeletedAt: tt.deletedAt, PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := IsVisible(n, now); got != tt.want {
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
			n := &News{DeletedAt: tt.deletedAt, PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := CanView(n, tt.isAdmin, now); got != tt.want {
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
			n := &News{PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := GetStatus(n, now); got != tt.want {
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
			n := &News{AuthorID: authorID}
			if got := CanEdit(n, tt.userID, tt.isAdmin); got != tt.want {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &News{TitleEN: tt.titleEN, TitleLocal: tt.titleLocal}
			if got := ResolveTitle(n, tt.prefLang, tt.defaultLang); got != tt.want {
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
		titleLocal *string
		bodyLocal  *string
		lang       string
		wantOK     bool
	}{
		{"en always ok", nil, nil, LangEN, true},
		{"local exists", &titleLocal, &bodyLocal, LangLocal, true},
		{"local missing title", nil, &bodyLocal, LangLocal, false},
		{"local missing body", &titleLocal, nil, LangLocal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &News{TitleEN: "EN", BodyEN: "EN Body", TitleLocal: tt.titleLocal, BodyLocal: tt.bodyLocal}
			_, _, ok := GetTranslation(n, tt.lang)
			if ok != tt.wantOK {
				t.Errorf("GetTranslation() ok = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestBuildNews(t *testing.T) {
	authorID := uuid.New()
	publisherID := uuid.New()
	titleLocal := "Local"
	bodyLocal := "Body Local"

	n := BuildNews(authorID, PublisherCollege, &publisherID, CategoryEvent, "Title", &titleLocal, "Body", &bodyLocal, nil, nil, nil)

	if n.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
	if n.AuthorID != authorID {
		t.Errorf("AuthorID = %v, want %v", n.AuthorID, authorID)
	}
	if n.PublisherType != PublisherCollege {
		t.Errorf("PublisherType = %v, want %v", n.PublisherType, PublisherCollege)
	}
	if n.Category != CategoryEvent {
		t.Errorf("Category = %v, want %v", n.Category, CategoryEvent)
	}
}

func TestBuildAttachment(t *testing.T) {
	newsID := uuid.New()
	fileID := uuid.New()

	a := BuildAttachment(newsID, fileID, "file.pdf", FileTypeDocument, 0)

	if a.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
	if a.NewsID != newsID {
		t.Errorf("NewsID = %v, want %v", a.NewsID, newsID)
	}
	if a.FileType != FileTypeDocument {
		t.Errorf("FileType = %v, want %v", a.FileType, FileTypeDocument)
	}
}
