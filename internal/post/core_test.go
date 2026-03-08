package post

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsTopLevelPost(t *testing.T) {
	tests := []struct {
		name     string
		parentID *uuid.UUID
		rootID   *uuid.UUID
		want     bool
	}{
		{"top level", nil, nil, true},
		{"comment", ptr(uuid.New()), ptr(uuid.New()), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Post{ParentID: tt.parentID, RootID: tt.rootID}
			if got := IsTopLevelPost(p); got != tt.want {
				t.Errorf("IsTopLevelPost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsComment(t *testing.T) {
	tests := []struct {
		name     string
		parentID *uuid.UUID
		rootID   *uuid.UUID
		want     bool
	}{
		{"top level", nil, nil, false},
		{"comment", ptr(uuid.New()), ptr(uuid.New()), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Post{ParentID: tt.parentID, RootID: tt.rootID}
			if got := IsComment(p); got != tt.want {
				t.Errorf("IsComment() = %v, want %v", got, tt.want)
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
		{"past publish time", &past, false},
		{"future publish time", &future, true},
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

func TestIsDeleted(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		deletedAt *time.Time
		want      bool
	}{
		{"not deleted", nil, false},
		{"deleted", &now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDeleted(tt.deletedAt); got != tt.want {
				t.Errorf("IsDeleted() = %v, want %v", got, tt.want)
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
		{"normal post", nil, nil, nil, false, true},
		{"deleted post member", &now, nil, nil, false, false},
		{"deleted post admin", &now, nil, nil, true, true},
		{"scheduled post member", nil, &future, nil, false, false},
		{"scheduled post admin", nil, &future, nil, true, true},
		{"published scheduled post", nil, &past, nil, false, true},
		{"expired post member", nil, nil, &past, false, false},
		{"expired post admin", nil, nil, &past, true, true},
		{"not expired", nil, nil, &future, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Post{DeletedAt: tt.deletedAt, PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := CanView(p, tt.isAdmin, now); got != tt.want {
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
		{"published no times", nil, nil, StatusPublished},
		{"scheduled", &future, nil, StatusScheduled},
		{"published past schedule", &past, nil, StatusPublished},
		{"expired", nil, &past, StatusExpired},
		{"not expired", nil, &future, StatusPublished},
		{"scheduled takes priority over expired", &future, &past, StatusScheduled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Post{PublishAt: tt.publishAt, ExpiresAt: tt.expiresAt}
			if got := GetStatus(p, now); got != tt.want {
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
		{"other user", otherID, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Post{AuthorID: authorID}
			if got := CanEdit(p, tt.userID, tt.isAdmin); got != tt.want {
				t.Errorf("CanEdit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateScopeType(t *testing.T) {
	tests := []struct {
		name      string
		scopeType string
		want      bool
	}{
		{"university", ScopeUniversity, true},
		{"college", ScopeCollege, true},
		{"department", ScopeDepartment, true},
		{"program", ScopeProgram, true},
		{"course", ScopeCourse, true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateScopeType(tt.scopeType); got != tt.want {
				t.Errorf("ValidateScopeType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateScopeID(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name      string
		scopeType string
		scopeID   *uuid.UUID
		want      bool
	}{
		{"university nil", ScopeUniversity, nil, true},
		{"university with id", ScopeUniversity, &id, false},
		{"college with id", ScopeCollege, &id, true},
		{"college nil", ScopeCollege, nil, false},
		{"department with id", ScopeDepartment, &id, true},
		{"program with id", ScopeProgram, &id, true},
		{"course with id", ScopeCourse, &id, true},
		{"course nil", ScopeCourse, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateScopeID(tt.scopeType, tt.scopeID); got != tt.want {
				t.Errorf("ValidateScopeID() = %v, want %v", got, tt.want)
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
		{"voice", FileTypeVoice, true},
		{"video", FileTypeVideo, true},
		{"invalid", "invalid", false},
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

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
		size     int64
		want     bool
	}{
		{"image ok", FileTypeImage, 5 * 1024 * 1024, true},
		{"image too large", FileTypeImage, 15 * 1024 * 1024, false},
		{"video ok", FileTypeVideo, 40 * 1024 * 1024, true},
		{"video too large", FileTypeVideo, 60 * 1024 * 1024, false},
		{"voice ok", FileTypeVoice, 5 * 1024 * 1024, true},
		{"document ok", FileTypeDocument, 15 * 1024 * 1024, true},
		{"document too large", FileTypeDocument, 25 * 1024 * 1024, false},
		{"unknown type", "unknown", 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateFileSize(tt.fileType, tt.size); got != tt.want {
				t.Errorf("ValidateFileSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMentions(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"no mentions", "Hello world", nil},
		{"single mention", "Hello @john", []string{"john"}},
		{"multiple mentions", "Hello @john and @jane", []string{"john", "jane"}},
		{"duplicate mentions", "Hey @john check @john", []string{"john"}},
		{"with dots", "Hi @john.doe", []string{"john.doe"}},
		{"with underscore", "Hi @john_doe", []string{"john_doe"}},
		{"mixed case", "Hi @John.Doe", []string{"john.doe"}},
		{"at start", "@admin please check", []string{"admin"}},
		{"multiple words", "@admin @mod @user test", []string{"admin", "mod", "user"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMentions(tt.body)
			if len(got) != len(tt.want) {
				t.Errorf("ParseMentions() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseMentions()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildPost(t *testing.T) {
	authorID := uuid.New()
	scopeID := uuid.New()
	publishAt := time.Now().Add(time.Hour)
	expiresAt := time.Now().Add(24 * time.Hour)

	p := BuildPost(authorID, ScopeCollege, &scopeID, "Test body", &publishAt, &expiresAt)

	if p.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
	if p.AuthorID != authorID {
		t.Errorf("AuthorID = %v, want %v", p.AuthorID, authorID)
	}
	if p.ScopeType != ScopeCollege {
		t.Errorf("ScopeType = %v, want %v", p.ScopeType, ScopeCollege)
	}
	if p.ScopeID == nil || *p.ScopeID != scopeID {
		t.Errorf("ScopeID = %v, want %v", p.ScopeID, scopeID)
	}
	if p.Body != "Test body" {
		t.Errorf("Body = %v, want %v", p.Body, "Test body")
	}
	if p.ParentID != nil || p.RootID != nil {
		t.Error("ParentID and RootID should be nil for top-level post")
	}
}

func TestBuildComment(t *testing.T) {
	authorID := uuid.New()
	parentID := uuid.New()
	scopeID := uuid.New()

	parent := &Post{
		ID:        parentID,
		ScopeType: ScopeCollege,
		ScopeID:   &scopeID,
	}

	c := BuildComment(authorID, parent, "Comment body")

	if c.ID == uuid.Nil {
		t.Error("ID should be generated")
	}
	if c.ParentID == nil || *c.ParentID != parentID {
		t.Errorf("ParentID = %v, want %v", c.ParentID, parentID)
	}
	if c.RootID == nil || *c.RootID != parentID {
		t.Errorf("RootID = %v, want %v", c.RootID, parentID)
	}
	if c.ScopeType != parent.ScopeType {
		t.Errorf("ScopeType = %v, want %v", c.ScopeType, parent.ScopeType)
	}

	// Test nested comment (reply to comment)
	nestedParent := &Post{
		ID:        uuid.New(),
		ScopeType: ScopeCollege,
		ScopeID:   &scopeID,
		RootID:    &parentID,
	}

	nested := BuildComment(authorID, nestedParent, "Nested reply")
	if nested.RootID == nil || *nested.RootID != parentID {
		t.Errorf("Nested RootID = %v, want %v", nested.RootID, parentID)
	}
}

func ptr[T any](v T) *T {
	return &v
}
