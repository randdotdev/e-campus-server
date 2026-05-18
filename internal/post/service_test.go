package post

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

// Mock implementations

type mockPostRepo struct {
	posts map[uuid.UUID]*Post
}

func newMockPostRepo() *mockPostRepo {
	return &mockPostRepo{posts: make(map[uuid.UUID]*Post)}
}

func (m *mockPostRepo) Create(ctx context.Context, p *Post) error {
	m.posts[p.ID] = p
	return nil
}

func (m *mockPostRepo) GetByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	return m.posts[id], nil
}

func (m *mockPostRepo) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*postView, error) {
	p := m.posts[id]
	if p == nil {
		return nil, nil
	}
	return &postView{PostWithAuthor: PostWithAuthor{Post: *p, AuthorName: "Test User"}}, nil
}

func (m *mockPostRepo) Update(ctx context.Context, p *Post) error {
	m.posts[p.ID] = p
	return nil
}

func (m *mockPostRepo) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	if p := m.posts[id]; p != nil {
		p.DeletedAt = &deletedAt
	}
	return nil
}

func (m *mockPostRepo) HardDelete(ctx context.Context, id uuid.UUID) error {
	delete(m.posts, id)
	return nil
}

func (m *mockPostRepo) ListByScope(ctx context.Context, scopeType string, scopeID *uuid.UUID, isAdmin bool, params pagination.PageParams) ([]postView, bool, error) {
	var result []postView
	for _, p := range m.posts {
		if p.ScopeType == scopeType && p.ParentID == nil {
			result = append(result, postView{PostWithAuthor: PostWithAuthor{Post: *p, AuthorName: "Test User"}})
		}
	}
	return result, false, nil
}

func (m *mockPostRepo) ListComments(ctx context.Context, rootID uuid.UUID, params pagination.PageParams) ([]postView, bool, error) {
	var result []postView
	for _, p := range m.posts {
		if p.RootID != nil && *p.RootID == rootID {
			result = append(result, postView{PostWithAuthor: PostWithAuthor{Post: *p, AuthorName: "Test User"}})
		}
	}
	return result, false, nil
}

func (m *mockPostRepo) IncrementLikeCount(ctx context.Context, id uuid.UUID) error {
	if p := m.posts[id]; p != nil {
		p.LikeCount++
	}
	return nil
}

func (m *mockPostRepo) DecrementLikeCount(ctx context.Context, id uuid.UUID) error {
	if p := m.posts[id]; p != nil && p.LikeCount > 0 {
		p.LikeCount--
	}
	return nil
}

func (m *mockPostRepo) IncrementCommentCount(ctx context.Context, id uuid.UUID) error {
	if p := m.posts[id]; p != nil {
		p.CommentCount++
	}
	return nil
}

func (m *mockPostRepo) DecrementCommentCount(ctx context.Context, id uuid.UUID) error {
	if p := m.posts[id]; p != nil && p.CommentCount > 0 {
		p.CommentCount--
	}
	return nil
}

type mockLikeRepo struct {
	likes map[string]bool
}

func newMockLikeRepo() *mockLikeRepo {
	return &mockLikeRepo{likes: make(map[string]bool)}
}

func (m *mockLikeRepo) key(postID, userID uuid.UUID) string {
	return postID.String() + ":" + userID.String()
}

func (m *mockLikeRepo) Create(ctx context.Context, postID, userID uuid.UUID) error {
	m.likes[m.key(postID, userID)] = true
	return nil
}

func (m *mockLikeRepo) Delete(ctx context.Context, postID, userID uuid.UUID) error {
	delete(m.likes, m.key(postID, userID))
	return nil
}

func (m *mockLikeRepo) Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	return m.likes[m.key(postID, userID)], nil
}

func (m *mockLikeRepo) GetUserLikes(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	result := make(map[uuid.UUID]bool)
	for _, postID := range postIDs {
		if m.likes[m.key(postID, userID)] {
			result[postID] = true
		}
	}
	return result, nil
}

type mockAttachmentRepo struct {
	attachments map[uuid.UUID]*PostAttachment
}

func newMockAttachmentRepo() *mockAttachmentRepo {
	return &mockAttachmentRepo{attachments: make(map[uuid.UUID]*PostAttachment)}
}

func (m *mockAttachmentRepo) Create(ctx context.Context, a *PostAttachment) error {
	m.attachments[a.ID] = a
	return nil
}

func (m *mockAttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.attachments, id)
	return nil
}

func (m *mockAttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*PostAttachment, error) {
	return m.attachments[id], nil
}

func (m *mockAttachmentRepo) ListByPostID(ctx context.Context, postID uuid.UUID) ([]PostAttachment, error) {
	var result []PostAttachment
	for _, a := range m.attachments {
		if a.PostID == postID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockAttachmentRepo) ListByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostAttachment, error) {
	result := make(map[uuid.UUID][]PostAttachment)
	for _, a := range m.attachments {
		for _, postID := range postIDs {
			if a.PostID == postID {
				result[postID] = append(result[postID], *a)
			}
		}
	}
	return result, nil
}

type mockMentionRepo struct {
	mentions map[uuid.UUID][]uuid.UUID
}

func newMockMentionRepo() *mockMentionRepo {
	return &mockMentionRepo{mentions: make(map[uuid.UUID][]uuid.UUID)}
}

func (m *mockMentionRepo) CreateBatch(ctx context.Context, postID uuid.UUID, userIDs []uuid.UUID) error {
	m.mentions[postID] = userIDs
	return nil
}

func (m *mockMentionRepo) DeleteByPostID(ctx context.Context, postID uuid.UUID) error {
	delete(m.mentions, postID)
	return nil
}

func (m *mockMentionRepo) ListByPostID(ctx context.Context, postID uuid.UUID) ([]MentionedUser, error) {
	var result []MentionedUser
	for _, userID := range m.mentions[postID] {
		result = append(result, MentionedUser{UserID: userID, Username: "user", FullName: "User"})
	}
	return result, nil
}

func (m *mockMentionRepo) ListByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]MentionedUser, error) {
	result := make(map[uuid.UUID][]MentionedUser)
	for _, postID := range postIDs {
		for _, userID := range m.mentions[postID] {
			result[postID] = append(result[postID], MentionedUser{UserID: userID})
		}
	}
	return result, nil
}

type mockUserLookup struct {
	users map[string]uuid.UUID
}

func newMockUserLookup() *mockUserLookup {
	return &mockUserLookup{users: make(map[string]uuid.UUID)}
}

func (m *mockUserLookup) GetUserIDsByUsernames(ctx context.Context, usernames []string) (map[string]uuid.UUID, error) {
	result := make(map[string]uuid.UUID)
	for _, username := range usernames {
		if id, ok := m.users[username]; ok {
			result[username] = id
		}
	}
	return result, nil
}

type mockScopeChecker struct {
	scopes map[string]bool
}

func newMockScopeChecker() *mockScopeChecker {
	return &mockScopeChecker{scopes: make(map[string]bool)}
}

func (m *mockScopeChecker) CanAccessScope(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) (bool, error) {
	return true, nil
}

func (m *mockScopeChecker) ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error) {
	key := scopeType + ":" + scopeID.String()
	if exists, ok := m.scopes[key]; ok {
		return exists, nil
	}
	return true, nil
}

type mockMuteChecker struct {
	muted map[string]bool
}

func newMockMuteChecker() *mockMuteChecker {
	return &mockMuteChecker{muted: make(map[string]bool)}
}

func (m *mockMuteChecker) IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	key := userID.String()
	if offeringID != nil {
		key += ":" + offeringID.String()
	}
	return m.muted[key], nil
}

func newTestService() *Service {
	return NewService(
		newMockPostRepo(),
		newMockLikeRepo(),
		newMockAttachmentRepo(),
		newMockMentionRepo(),
		newMockUserLookup(),
		newMockScopeChecker(),
		newMockMuteChecker(),
		nil,
	)
}

// Tests

func TestCreatePost(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	tests := []struct {
		name      string
		scopeType string
		scopeID   *uuid.UUID
		body      string
		wantErr   error
	}{
		{"university scope", ScopeUniversity, nil, "Hello world", nil},
		{"college scope", ScopeCollege, ptr(uuid.New()), "College post", nil},
		{"invalid scope type", "invalid", nil, "Test", ErrInvalidScope},
		{"college without scope id", ScopeCollege, nil, "Test", ErrInvalidScope},
		{"university with scope id", ScopeUniversity, ptr(uuid.New()), "Test", ErrInvalidScope},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			post, err := s.CreatePost(ctx, authorID, tt.scopeType, tt.scopeID, tt.body, nil, nil)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("CreatePost() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("CreatePost() unexpected error: %v", err)
			}
			if post.Body != tt.body {
				t.Errorf("Body = %v, want %v", post.Body, tt.body)
			}
			if post.AuthorID != authorID {
				t.Errorf("AuthorID = %v, want %v", post.AuthorID, authorID)
			}
		})
	}
}

func TestCreateComment(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	// Create parent post
	parent, err := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "Parent post", nil, nil)
	if err != nil {
		t.Fatalf("Failed to create parent: %v", err)
	}

	// Create comment
	comment, err := s.CreateComment(ctx, authorID, parent.ID, "This is a comment")
	if err != nil {
		t.Fatalf("CreateComment() error: %v", err)
	}

	if comment.ParentID == nil || *comment.ParentID != parent.ID {
		t.Errorf("ParentID = %v, want %v", comment.ParentID, parent.ID)
	}
	if comment.RootID == nil || *comment.RootID != parent.ID {
		t.Errorf("RootID = %v, want %v", comment.RootID, parent.ID)
	}
}

func TestCreateNestedComment(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	// Create parent post
	parent, _ := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "Parent post", nil, nil)

	// Create first-level comment
	comment1, _ := s.CreateComment(ctx, authorID, parent.ID, "First comment")

	// Create nested reply
	reply, err := s.CreateComment(ctx, authorID, comment1.ID, "Nested reply")
	if err != nil {
		t.Fatalf("CreateComment() nested error: %v", err)
	}

	// Nested reply should have parent as comment1 but root as original post
	if reply.ParentID == nil || *reply.ParentID != comment1.ID {
		t.Errorf("ParentID = %v, want %v", reply.ParentID, comment1.ID)
	}
	if reply.RootID == nil || *reply.RootID != parent.ID {
		t.Errorf("RootID = %v, want %v (original post)", reply.RootID, parent.ID)
	}
}

func TestUpdatePost(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	post, _ := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "Original", nil, nil)

	tests := []struct {
		name    string
		userID  uuid.UUID
		isAdmin bool
		body    string
		wantErr error
	}{
		{"author can edit", authorID, false, "Updated by author", nil},
		{"admin can edit", otherID, true, "Updated by admin", nil},
		{"other cannot edit", otherID, false, "Should fail", ErrNotAuthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newBody := tt.body
			_, err := s.UpdatePost(ctx, post.ID, tt.userID, tt.isAdmin, &newBody, nil, nil, false)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("UpdatePost() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdatePost() unexpected error: %v", err)
			}
		})
	}
}

func TestDeletePost(t *testing.T) {
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
			post, _ := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "To delete", nil, nil)
			err := s.DeletePost(ctx, post.ID, tt.userID, tt.isAdmin)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("DeletePost() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("DeletePost() unexpected error: %v", err)
			}
		})
	}
}

func TestLikePost(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	userID := uuid.New()

	post, _ := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "Likeable", nil, nil)

	// First like should succeed
	if err := s.LikePost(ctx, post.ID, userID); err != nil {
		t.Fatalf("LikePost() first error: %v", err)
	}

	// Second like should fail
	if err := s.LikePost(ctx, post.ID, userID); err != ErrAlreadyLiked {
		t.Errorf("LikePost() second error = %v, want %v", err, ErrAlreadyLiked)
	}
}

func TestUnlikePost(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	userID := uuid.New()

	post, _ := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "Unlikeable", nil, nil)

	// Unlike without like should fail
	if err := s.UnlikePost(ctx, post.ID, userID); err != ErrNotLiked {
		t.Errorf("UnlikePost() without like error = %v, want %v", err, ErrNotLiked)
	}

	// Like then unlike should succeed
	_ = s.LikePost(ctx, post.ID, userID)
	if err := s.UnlikePost(ctx, post.ID, userID); err != nil {
		t.Errorf("UnlikePost() after like error: %v", err)
	}
}

func TestPinPost(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()

	post, _ := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "Pinnable", nil, nil)
	comment, _ := s.CreateComment(ctx, authorID, post.ID, "Comment")

	tests := []struct {
		name    string
		postID  uuid.UUID
		isAdmin bool
		wantErr error
	}{
		{"admin can pin post", post.ID, true, nil},
		{"non-admin cannot pin", post.ID, false, ErrNotAuthorized},
		{"cannot pin comment", comment.ID, true, ErrCannotPinComment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.PinPost(ctx, tt.postID, tt.isAdmin, true)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("PinPost() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("PinPost() unexpected error: %v", err)
			}
		})
	}
}

func TestAddAttachment(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	authorID := uuid.New()
	otherID := uuid.New()

	post, _ := s.CreatePost(ctx, authorID, ScopeUniversity, nil, "With attachment", nil, nil)

	tests := []struct {
		name      string
		userID    uuid.UUID
		isAdmin   bool
		fileType  string
		sizeBytes int64
		wantErr   error
	}{
		{"author can add image", authorID, false, FileTypeImage, 5 * 1024 * 1024, nil},
		{"admin can add", otherID, true, FileTypeDocument, 10 * 1024 * 1024, nil},
		{"other cannot add", otherID, false, FileTypeImage, 1024, ErrNotAuthorized},
		{"invalid file type", authorID, false, "invalid", 1024, ErrInvalidFileType},
		{"file too large", authorID, false, FileTypeImage, 15 * 1024 * 1024, ErrFileTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.AddAttachment(ctx, post.ID, tt.userID, tt.isAdmin, uuid.New(), "test.jpg", tt.fileType, tt.sizeBytes, 0)

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

// ptr is defined in core_test.go
