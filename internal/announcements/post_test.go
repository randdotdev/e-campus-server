package announcements

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// postMockRepo is an in-memory PostRepository.
type postMockRepo struct {
	posts       map[uuid.UUID]*Post
	likes       map[string]bool
	attachments map[uuid.UUID]*PostAttachment
	mentions    map[uuid.UUID][]uuid.UUID
	// forceConflict makes every UpdatePost lose the version CAS, standing in
	// for a writer that keeps winning the race.
	forceConflict bool
}

func newPostMockRepo() *postMockRepo {
	return &postMockRepo{
		posts:       make(map[uuid.UUID]*Post),
		likes:       make(map[string]bool),
		attachments: make(map[uuid.UUID]*PostAttachment),
		mentions:    make(map[uuid.UUID][]uuid.UUID),
	}
}

func likeKey(p, u uuid.UUID) string { return p.String() + ":" + u.String() }

func (m *postMockRepo) CreatePost(ctx context.Context, p *Post, mentionIDs []uuid.UUID) error {
	m.posts[p.ID] = p
	if len(mentionIDs) > 0 {
		m.mentions[p.ID] = mentionIDs
	}
	return nil
}
func (m *postMockRepo) CreateComment(ctx context.Context, comment *Post, mentionIDs []uuid.UUID) error {
	m.posts[comment.ID] = comment
	if len(mentionIDs) > 0 {
		m.mentions[comment.ID] = mentionIDs
	}
	if root := m.posts[*comment.RootID]; root != nil {
		root.CommentCount++
	}
	return nil
}
func (m *postMockRepo) GetPostByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	return m.posts[id], nil
}
func (m *postMockRepo) GetPostByIDWithAuthor(ctx context.Context, id uuid.UUID) (*PostView, error) {
	p := m.posts[id]
	if p == nil {
		return nil, nil
	}
	return &PostView{PostWithAuthor: PostWithAuthor{Post: *p}}, nil
}
func (m *postMockRepo) UpdatePost(ctx context.Context, p *Post, expectedVersion int64, replaceMentions bool, mentionIDs []uuid.UUID) (int64, error) {
	if m.forceConflict {
		return 0, ErrConflict
	}
	existing := m.posts[p.ID]
	if existing != nil && existing.Version != expectedVersion {
		return 0, ErrConflict
	}
	p.Version = expectedVersion + 1
	m.posts[p.ID] = p
	if replaceMentions {
		if len(mentionIDs) > 0 {
			m.mentions[p.ID] = mentionIDs
		} else {
			delete(m.mentions, p.ID)
		}
	}
	return p.Version, nil
}
func (m *postMockRepo) SoftDeletePost(ctx context.Context, id uuid.UUID, t time.Time) error {
	if p := m.posts[id]; p != nil {
		p.DeletedAt = &t
	}
	return nil
}
func (m *postMockRepo) DeleteComment(ctx context.Context, id, rootID uuid.UUID) error {
	delete(m.posts, id)
	delete(m.mentions, id)
	if root := m.posts[rootID]; root != nil && root.CommentCount > 0 {
		root.CommentCount--
	}
	return nil
}
func (m *postMockRepo) ListPostsByScope(ctx context.Context, s ScopeType, id *uuid.UUID, isAdmin bool, p pagination.PageParams) ([]PostView, bool, error) {
	return nil, false, nil
}
func (m *postMockRepo) ListComments(ctx context.Context, rootID uuid.UUID, p pagination.PageParams) ([]PostView, bool, error) {
	return nil, false, nil
}
func (m *postMockRepo) Like(ctx context.Context, postID, userID uuid.UUID) error {
	key := likeKey(postID, userID)
	if m.likes[key] {
		return ErrAlreadyLiked
	}
	m.likes[key] = true
	if p := m.posts[postID]; p != nil {
		p.LikeCount++
	}
	return nil
}
func (m *postMockRepo) Unlike(ctx context.Context, postID, userID uuid.UUID) error {
	key := likeKey(postID, userID)
	if !m.likes[key] {
		return ErrNotLiked
	}
	delete(m.likes, key)
	if p := m.posts[postID]; p != nil && p.LikeCount > 0 {
		p.LikeCount--
	}
	return nil
}
func (m *postMockRepo) LikeExists(ctx context.Context, p, u uuid.UUID) (bool, error) {
	return m.likes[likeKey(p, u)], nil
}
func (m *postMockRepo) GetUserLikes(ctx context.Context, ids []uuid.UUID, u uuid.UUID) (map[uuid.UUID]bool, error) {
	return map[uuid.UUID]bool{}, nil
}
func (m *postMockRepo) CreateAttachment(ctx context.Context, a *PostAttachment) error {
	m.attachments[a.ID] = a
	return nil
}
func (m *postMockRepo) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	delete(m.attachments, id)
	return nil
}
func (m *postMockRepo) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*PostAttachment, error) {
	return m.attachments[id], nil
}
func (m *postMockRepo) ListAttachmentsByPostID(ctx context.Context, id uuid.UUID) ([]PostAttachment, error) {
	return nil, nil
}
func (m *postMockRepo) ListAttachmentsByPostIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]PostAttachment, error) {
	return map[uuid.UUID][]PostAttachment{}, nil
}
func (m *postMockRepo) ListMentionsByPostID(ctx context.Context, id uuid.UUID) ([]MentionedUser, error) {
	return nil, nil
}
func (m *postMockRepo) ListMentionsByPostIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]MentionedUser, error) {
	return map[uuid.UUID][]MentionedUser{}, nil
}

type openScopes struct{}

func (openScopes) ScopeExists(ctx context.Context, s ScopeType, id uuid.UUID) (bool, error) {
	return true, nil
}
func (openScopes) CanAccessScope(ctx context.Context, u uuid.UUID, s ScopeType, id *uuid.UUID) (bool, error) {
	return true, nil
}

type muteState struct{ muted bool }

func (m *muteState) IsMuted(ctx context.Context, u uuid.UUID, o *uuid.UUID) (bool, error) {
	return m.muted, nil
}

type noUsers struct{}

func (noUsers) GetUserIDsByUsernames(ctx context.Context, names []string) (map[string]uuid.UUID, error) {
	return map[string]uuid.UUID{}, nil
}

// stubFiles is an open FileStore: every file resolves to an image inode of
// the same id, counting always succeeds.
type stubFiles struct{}

func (stubFiles) ResolveUpload(ctx context.Context, ownerID, fileID uuid.UUID) (StoredFile, error) {
	return StoredFile{InodeID: fileID, Name: "file.png", MimeType: "image/png"}, nil
}

func (stubFiles) Link(ctx context.Context, inodeID uuid.UUID) error   { return nil }
func (stubFiles) Unlink(ctx context.Context, inodeID uuid.UUID) error { return nil }

func (stubFiles) Presign(ctx context.Context, inodeID uuid.UUID, filename string) (string, error) {
	return "https://files.test/" + filename, nil
}

func newPostService(mutes *muteState) (*PostService, *postMockRepo) {
	repo := newPostMockRepo()
	return NewPostService(repo, noUsers{}, openScopes{}, mutes, nil, stubFiles{}, slog.New(slog.DiscardHandler)), repo
}

func TestPostCreateScopeValidation(t *testing.T) {
	s, _ := newPostService(&muteState{})
	ctx := context.Background()
	author := uuid.New()

	if _, err := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: "invalid", Body: "hi"}); err != ErrInvalidScope {
		t.Errorf("invalid scope = %v, want ErrInvalidScope", err)
	}
	if _, err := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeCollege, Body: "hi"}); err != ErrInvalidScope {
		t.Errorf("college without id = %v, want ErrInvalidScope", err)
	}
	p, err := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "hi"})
	if err != nil {
		t.Fatalf("valid create = %v", err)
	}
	if p.ScopeType != ScopeUniversity {
		t.Errorf("scope = %v", p.ScopeType)
	}
}

func TestPostCommentMuted(t *testing.T) {
	mutes := &muteState{}
	s, _ := newPostService(mutes)
	ctx := context.Background()
	author := uuid.New()

	root, _ := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "root"})

	mutes.muted = true
	if _, err := s.CreateComment(ctx, CreateCommentInput{AuthorID: uuid.New(), ParentID: root.ID, Body: "c"}); err != ErrUserMuted {
		t.Errorf("muted comment = %v, want ErrUserMuted", err)
	}

	mutes.muted = false
	c, err := s.CreateComment(ctx, CreateCommentInput{AuthorID: uuid.New(), ParentID: root.ID, Body: "c"})
	if err != nil {
		t.Fatalf("comment = %v", err)
	}
	if !c.IsComment() {
		t.Error("expected comment")
	}
}

func TestPostCommentIncrementsCountInTx(t *testing.T) {
	s, repo := newPostService(&muteState{})
	ctx := context.Background()
	author := uuid.New()
	root, _ := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "root"})

	if _, err := s.CreateComment(ctx, CreateCommentInput{AuthorID: author, ParentID: root.ID, Body: "c"}); err != nil {
		t.Fatalf("comment = %v", err)
	}
	if repo.posts[root.ID].CommentCount != 1 {
		t.Errorf("comment count = %d, want 1", repo.posts[root.ID].CommentCount)
	}
}

func TestPostLikeUnlike(t *testing.T) {
	s, repo := newPostService(&muteState{})
	ctx := context.Background()
	author := uuid.New()
	liker := uuid.New()
	p, _ := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "x"})

	if err := s.Like(ctx, p.ID, liker); err != nil {
		t.Fatalf("like = %v", err)
	}
	if repo.posts[p.ID].LikeCount != 1 {
		t.Errorf("like count = %d, want 1", repo.posts[p.ID].LikeCount)
	}
	if err := s.Like(ctx, p.ID, liker); err != ErrAlreadyLiked {
		t.Errorf("double like = %v, want ErrAlreadyLiked", err)
	}
	if err := s.Unlike(ctx, p.ID, liker); err != nil {
		t.Fatalf("unlike = %v", err)
	}
	if repo.posts[p.ID].LikeCount != 0 {
		t.Errorf("like count = %d, want 0", repo.posts[p.ID].LikeCount)
	}
	if err := s.Unlike(ctx, p.ID, liker); err != ErrNotLiked {
		t.Errorf("double unlike = %v, want ErrNotLiked", err)
	}
}

func TestPostPinComment(t *testing.T) {
	s, _ := newPostService(&muteState{})
	ctx := context.Background()
	author := uuid.New()
	root, _ := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "root"})
	c, _ := s.CreateComment(ctx, CreateCommentInput{AuthorID: author, ParentID: root.ID, Body: "c"})

	if err := s.Pin(ctx, c.ID, true); err != ErrCannotPinComment {
		t.Errorf("pin comment = %v, want ErrCannotPinComment", err)
	}
	if err := s.Pin(ctx, root.ID, true); err != nil {
		t.Errorf("pin = %v", err)
	}
}

func TestPostDeleteCommentDecrementsInTx(t *testing.T) {
	s, repo := newPostService(&muteState{})
	ctx := context.Background()
	author := uuid.New()
	root, _ := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "root"})
	c, _ := s.CreateComment(ctx, CreateCommentInput{AuthorID: author, ParentID: root.ID, Body: "c"})

	if err := s.DeletePost(ctx, c.ID); err != nil {
		t.Fatalf("delete comment = %v", err)
	}
	if repo.posts[root.ID].CommentCount != 0 {
		t.Errorf("comment count = %d, want 0", repo.posts[root.ID].CommentCount)
	}
	if _, ok := repo.posts[c.ID]; ok {
		t.Error("comment should be hard-deleted")
	}
}

func TestPostUpdateBumpsVersion(t *testing.T) {
	s, _ := newPostService(&muteState{})
	ctx := context.Background()
	author := uuid.New()
	p, _ := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "v0"})
	origVersion := p.Version

	body := "v1"
	updated, err := s.UpdatePost(ctx, UpdatePostInput{ID: p.ID, Body: &body})
	if err != nil {
		t.Fatalf("update = %v", err)
	}
	if updated.Version != origVersion+1 {
		t.Errorf("version = %d, want %d", updated.Version, origVersion+1)
	}
}

func TestPostUpdateConflict(t *testing.T) {
	s, repo := newPostService(&muteState{})
	ctx := context.Background()
	author := uuid.New()
	p, _ := s.CreatePost(ctx, CreatePostInput{AuthorID: author, ScopeType: ScopeUniversity, Body: "v0"})

	repo.forceConflict = true
	body := "v1"
	if _, err := s.UpdatePost(ctx, UpdatePostInput{ID: p.ID, Body: &body}); err != ErrConflict {
		t.Errorf("update under permanent conflict = %v, want ErrConflict", err)
	}
}
