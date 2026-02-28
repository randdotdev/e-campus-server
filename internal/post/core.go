package post

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9._]+)`)

func IsTopLevelPost(p *Post) bool {
	return p.ParentID == nil && p.RootID == nil
}

func IsComment(p *Post) bool {
	return p.ParentID != nil && p.RootID != nil
}

func IsScheduled(publishAt *time.Time, now time.Time) bool {
	return publishAt != nil && publishAt.After(now)
}

func IsExpired(expiresAt *time.Time, now time.Time) bool {
	return expiresAt != nil && expiresAt.Before(now)
}

func IsDeleted(deletedAt *time.Time) bool {
	return deletedAt != nil
}

func IsVisible(p *Post, now time.Time) bool {
	if IsDeleted(p.DeletedAt) {
		return false
	}
	if IsScheduled(p.PublishAt, now) {
		return false
	}
	if IsExpired(p.ExpiresAt, now) {
		return false
	}
	return true
}

func CanView(p *Post, isAdmin bool, now time.Time) bool {
	if IsDeleted(p.DeletedAt) {
		return isAdmin
	}
	if IsScheduled(p.PublishAt, now) {
		return isAdmin
	}
	if IsExpired(p.ExpiresAt, now) {
		return isAdmin
	}
	return true
}

func GetStatus(p *Post, now time.Time) string {
	if IsScheduled(p.PublishAt, now) {
		return StatusScheduled
	}
	if IsExpired(p.ExpiresAt, now) {
		return StatusExpired
	}
	return StatusPublished
}

func CanEdit(p *Post, userID uuid.UUID, isAdmin bool) bool {
	if p.AuthorID == userID {
		return true
	}
	return isAdmin
}

func CanDelete(p *Post, userID uuid.UUID, isAdmin bool) bool {
	return CanEdit(p, userID, isAdmin)
}

func CanPin(isAdmin bool) bool {
	return isAdmin
}

func ValidateScopeType(scopeType string) bool {
	switch scopeType {
	case ScopeUniversity, ScopeCollege, ScopeDepartment, ScopeProgram:
		return true
	}
	return false
}

func ValidateScopeID(scopeType string, scopeID *uuid.UUID) bool {
	if scopeType == ScopeUniversity {
		return scopeID == nil
	}
	return scopeID != nil
}

func ValidateFileType(fileType string) bool {
	switch fileType {
	case FileTypeImage, FileTypeDocument, FileTypeVoice, FileTypeVideo:
		return true
	}
	return false
}

func ValidateFileSize(fileType string, sizeBytes int64) bool {
	switch fileType {
	case FileTypeImage:
		return sizeBytes <= MaxImageSize
	case FileTypeVideo:
		return sizeBytes <= MaxVideoSize
	case FileTypeVoice:
		return sizeBytes <= MaxVoiceSize
	case FileTypeDocument:
		return sizeBytes <= MaxDocumentSize
	}
	return false
}

func ParseMentions(body string) []string {
	matches := mentionRegex.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	var mentions []string

	for _, match := range matches {
		if len(match) >= 2 {
			username := strings.ToLower(match[1])
			if !seen[username] {
				seen[username] = true
				mentions = append(mentions, username)
			}
		}
	}
	return mentions
}

func BuildPost(authorID uuid.UUID, scopeType string, scopeID *uuid.UUID, body string, publishAt, expiresAt *time.Time) *Post {
	return &Post{
		ID:        uuid.New(),
		ScopeType: scopeType,
		ScopeID:   scopeID,
		Body:      body,
		PublishAt: publishAt,
		ExpiresAt: expiresAt,
		AuthorID:  authorID,
		CreatedAt: time.Now(),
	}
}

func BuildComment(authorID uuid.UUID, parent *Post, body string) *Post {
	rootID := parent.RootID
	if rootID == nil {
		rootID = &parent.ID
	}
	return &Post{
		ID:        uuid.New(),
		ScopeType: parent.ScopeType,
		ScopeID:   parent.ScopeID,
		ParentID:  &parent.ID,
		RootID:    rootID,
		Body:      body,
		AuthorID:  authorID,
		CreatedAt: time.Now(),
	}
}

func BuildAttachment(postID, storedFileID uuid.UUID, displayName, fileType string, orderIndex int) *PostAttachment {
	return &PostAttachment{
		ID:           uuid.New(),
		PostID:       postID,
		StoredFileID: storedFileID,
		DisplayName:  displayName,
		FileType:     fileType,
		OrderIndex:   orderIndex,
	}
}
