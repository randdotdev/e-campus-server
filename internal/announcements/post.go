package announcements

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// maxUpdateRetries bounds the optimistic-concurrency retry loop shared by the
// post and activity update paths: a write that loses the version CAS re-reads
// fresh state and tries again, giving up as ErrConflict after this many rounds.
const maxUpdateRetries = 3

// ── Value objects ──────────────────────────────────────────────────────────

// ScopeType is the audience a post targets. Offering-scoped posts are the
// classroom posts; the other scopes are the institution-wide member feed.
type ScopeType string

// Post scopes, widest to narrowest.
const (
	ScopeUniversity ScopeType = "university"
	ScopeCollege    ScopeType = "college"
	ScopeDepartment ScopeType = "department"
	ScopeProgram    ScopeType = "program"
	ScopeOffering   ScopeType = "offering" // classroom posts
)

// ── Entities ───────────────────────────────────────────────────────────────

// Post is one feed entry. A post with ParentID/RootID set is a comment; a
// soft-deleted post keeps its comment thread readable.
type Post struct {
	ID           uuid.UUID  `db:"id"`
	ScopeType    ScopeType  `db:"scope_type"`
	ScopeID      *uuid.UUID `db:"scope_id"`
	ParentID     *uuid.UUID `db:"parent_id"`
	RootID       *uuid.UUID `db:"root_id"`
	Body         string     `db:"body"`
	IsPinned     bool       `db:"is_pinned"`
	PublishAt    *time.Time `db:"publish_at"`
	ExpiresAt    *time.Time `db:"expires_at"`
	AuthorID     uuid.UUID  `db:"author_id"`
	LikeCount    int        `db:"like_count"`
	CommentCount int        `db:"comment_count"`
	Version      int64      `db:"version"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

// PostAttachment is one file attached to a post.
type PostAttachment struct {
	ID          uuid.UUID `db:"id"`
	PostID      uuid.UUID `db:"post_id"`
	InodeID     uuid.UUID `db:"inode_id"`
	DisplayName string    `db:"display_name"`
	FileType    string    `db:"file_type"`
	OrderIndex  int       `db:"order_index"`
}

// MentionedUser is one @mentioned user's display projection.
type MentionedUser struct {
	UserID   uuid.UUID `db:"user_id"`
	Username string    `db:"username"`
	FullName string    `db:"full_name"`
}

// ── Derived read models ────────────────────────────────────────────────────

// PostWithAuthor is the post joined with the author's display columns
// (posts ⋈ users, the published identity columns).
type PostWithAuthor struct {
	Post
	AuthorName           string  `db:"author_name"`
	AuthorNameLocal      *string `db:"author_name_local"`
	AuthorAvatar         *string `db:"author_avatar"`
	AuthorRoleTitle      *string `db:"author_role_title"`
	AuthorRoleTitleLocal *string `db:"author_role_title_local"`
}

// PostView is a post with author and scope display columns, the shape the
// feed lists render.
type PostView struct {
	PostWithAuthor
	ScopeName      *string `db:"scope_name"`
	ScopeNameLocal *string `db:"scope_name_local"`
}

// ── Rules ──────────────────────────────────────────────────────────────────

// IsTopLevel reports whether the post starts a thread.
func (p *Post) IsTopLevel() bool { return p.ParentID == nil && p.RootID == nil }

// IsComment reports whether the post replies inside a thread.
func (p *Post) IsComment() bool { return p.ParentID != nil && p.RootID != nil }

// Visible reports whether the post is live for regular readers.
func (p *Post) Visible(now time.Time) bool {
	return visible(p.DeletedAt, p.PublishAt, p.ExpiresAt, now)
}

// CanView reports whether the reader may see the post; admins see scheduled
// and expired posts.
func (p *Post) CanView(revealHidden bool, now time.Time) bool {
	return canView(p.DeletedAt, p.PublishAt, p.ExpiresAt, revealHidden, now)
}

// Status returns the post's publish lifecycle state.
func (p *Post) Status(now time.Time) Status { return statusOf(p.PublishAt, p.ExpiresAt, now) }

// ValidScopeType reports whether s is a known scope.
func ValidScopeType(s ScopeType) bool {
	switch s {
	case ScopeUniversity, ScopeCollege, ScopeDepartment, ScopeProgram, ScopeOffering:
		return true
	}
	return false
}

// ValidScopeID reports whether the scope ID's presence matches the scope
// type: university-wide posts carry none, every narrower scope requires one.
func ValidScopeID(s ScopeType, scopeID *uuid.UUID) bool {
	if s == ScopeUniversity {
		return scopeID == nil
	}
	return scopeID != nil
}

var mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9._]+)`)

// ParseMentions extracts the distinct @mentioned usernames from a post body,
// lower-cased, in order of first appearance.
func ParseMentions(body string) []string {
	matches := mentionRegex.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool)
	var mentions []string
	for _, m := range matches {
		if len(m) >= 2 {
			u := strings.ToLower(m[1])
			if !seen[u] {
				seen[u] = true
				mentions = append(mentions, u)
			}
		}
	}
	return mentions
}

// BuildPost constructs a new top-level post from its input.
func BuildPost(in CreatePostInput) *Post {
	return &Post{
		ID:        uuid.New(),
		ScopeType: in.ScopeType,
		ScopeID:   in.ScopeID,
		Body:      in.Body,
		PublishAt: in.PublishAt,
		ExpiresAt: in.ExpiresAt,
		AuthorID:  in.AuthorID,
		CreatedAt: time.Now(),
	}
}

// BuildComment constructs a reply under parent, resolving the thread root.
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

// BuildPostAttachment constructs an attachment row for a post.
func BuildPostAttachment(postID, inodeID uuid.UUID, displayName, fileType string, orderIndex int) *PostAttachment {
	return &PostAttachment{
		ID:          uuid.New(),
		PostID:      postID,
		InodeID:     inodeID,
		DisplayName: displayName,
		FileType:    fileType,
		OrderIndex:  orderIndex,
	}
}

// ── Ports ──────────────────────────────────────────────────────────────────

// PostRepository persists posts, likes, attachments, and mentions. Every
// multi-row write is one repository transaction, named for the use case it
// serves.
//
// CreatePost inserts the post and its mention rows atomically. CreateComment
// inserts the comment, its mention rows, and the root post's comment-count
// increment atomically (the root is comment.RootID). UpdatePost is an
// optimistic compare-and-swap keyed on expectedVersion: it persists the post
// and, when replaceMentions is true, replaces the mention rows in the same
// transaction, returning the new version; a version mismatch is ErrConflict.
// DeleteComment removes the comment, its mentions, and decrements the root's
// comment count atomically. Like records the like and increments the count
// atomically — a duplicate is ErrAlreadyLiked, enforced by the primary key,
// never by a prior read. Unlike removes the like and decrements the count
// atomically — a missing like is ErrNotLiked. GetPostByID and
// GetPostByIDWithAuthor return nil (no error) when the post does not exist.
type PostRepository interface {
	CreatePost(ctx context.Context, p *Post, mentionIDs []uuid.UUID) error
	CreateComment(ctx context.Context, comment *Post, mentionIDs []uuid.UUID) error
	GetPostByID(ctx context.Context, id uuid.UUID) (*Post, error)
	GetPostByIDWithAuthor(ctx context.Context, id uuid.UUID) (*PostView, error)
	UpdatePost(ctx context.Context, p *Post, expectedVersion int64, replaceMentions bool, mentionIDs []uuid.UUID) (int64, error)
	SoftDeletePost(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
	DeleteComment(ctx context.Context, id, rootID uuid.UUID) error
	ListPostsByScope(ctx context.Context, scopeType ScopeType, scopeID *uuid.UUID, revealHidden bool, params pagination.PageParams) ([]PostView, bool, error)
	ListComments(ctx context.Context, rootID uuid.UUID, params pagination.PageParams) ([]PostView, bool, error)

	Like(ctx context.Context, postID, userID uuid.UUID) error
	Unlike(ctx context.Context, postID, userID uuid.UUID) error
	LikeExists(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	GetUserLikes(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error)

	CreateAttachment(ctx context.Context, a *PostAttachment) error
	DeleteAttachment(ctx context.Context, id uuid.UUID) error
	GetAttachmentByID(ctx context.Context, id uuid.UUID) (*PostAttachment, error)
	ListAttachmentsByPostID(ctx context.Context, postID uuid.UUID) ([]PostAttachment, error)
	ListAttachmentsByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostAttachment, error)

	ListMentionsByPostID(ctx context.Context, postID uuid.UUID) ([]MentionedUser, error)
	ListMentionsByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]MentionedUser, error)
}

// UserLookup resolves @mention usernames to user IDs (identity context).
type UserLookup interface {
	GetUserIDsByUsernames(ctx context.Context, usernames []string) (map[string]uuid.UUID, error)
}

// ScopeChecker validates and authorizes post scopes (management context
// data); the composition root adapts the management services to it.
type ScopeChecker interface {
	ScopeExists(ctx context.Context, scopeType ScopeType, scopeID uuid.UUID) (bool, error)
	CanAccessScope(ctx context.Context, userID uuid.UUID, scopeType ScopeType, scopeID *uuid.UUID) (bool, error)
}

// MuteChecker reports whether a user is muted (communication context).
type MuteChecker interface {
	IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error)
}

// Notifier sends bulk notifications (communication context). Failures are
// logged by the caller, never fatal to the use case.
type Notifier interface {
	SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

// ── Service input types ────────────────────────────────────────────────────

// CreatePostInput is the content of a new top-level post.
type CreatePostInput struct {
	AuthorID  uuid.UUID
	ScopeType ScopeType
	ScopeID   *uuid.UUID
	Body      string
	PublishAt *time.Time
	ExpiresAt *time.Time
}

// CreateCommentInput is the content of a new reply.
type CreateCommentInput struct {
	AuthorID uuid.UUID
	ParentID uuid.UUID
	Body     string
}

// UpdatePostInput is a partial edit of a post; nil fields are left unchanged
// and ClearSchedule drops both schedule bounds.
type UpdatePostInput struct {
	ID            uuid.UUID
	Body          *string
	PublishAt     *time.Time
	ExpiresAt     *time.Time
	ClearSchedule bool
}

// AddPostAttachmentInput attaches a file from the actor's own drive to a
// post; DisplayName empty means the drive name.
type AddPostAttachmentInput struct {
	PostID      uuid.UUID
	ActorID     uuid.UUID
	UploadID    uuid.UUID
	DisplayName string
	FileType    string
	OrderIndex  int
}

// ── Service (use cases) ────────────────────────────────────────────────────

// PostService manages the member feed and classroom posts: threads, likes,
// mentions, and attachments.
type PostService struct {
	repo     PostRepository
	users    UserLookup
	scopes   ScopeChecker
	mutes    MuteChecker
	notifier Notifier
	files    FileStore
	log      *slog.Logger
}

// NewPostService wires a post service. mutes and notifier may be nil.
func NewPostService(repo PostRepository, users UserLookup, scopes ScopeChecker, mutes MuteChecker, notifier Notifier, files FileStore, log *slog.Logger) *PostService {
	return &PostService{repo: repo, users: users, scopes: scopes, mutes: mutes, notifier: notifier, files: files, log: log}
}

// CreatePost publishes a top-level post after validating its scope.
func (s *PostService) CreatePost(ctx context.Context, in CreatePostInput) (*Post, error) {
	if !ValidScopeType(in.ScopeType) || !ValidScopeID(in.ScopeType, in.ScopeID) {
		return nil, ErrInvalidScope
	}
	if in.ScopeID != nil {
		exists, err := s.scopes.ScopeExists(ctx, in.ScopeType, *in.ScopeID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrInvalidScope
		}
	}

	post := BuildPost(in)
	mentionIDs, err := s.resolveMentions(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreatePost(ctx, post, mentionIDs); err != nil {
		return nil, err
	}
	s.notifyMentions(ctx, post.ID, mentionIDs)
	return post, nil
}

// CreateComment replies under a visible post; muted users cannot comment.
func (s *PostService) CreateComment(ctx context.Context, in CreateCommentInput) (*Post, error) {
	parent, err := s.repo.GetPostByID(ctx, in.ParentID)
	if err != nil {
		return nil, err
	}
	if parent == nil || !parent.Visible(time.Now()) {
		return nil, ErrPostNotFound
	}
	if err := s.checkMuted(ctx, in.AuthorID, parent.ScopeType, parent.ScopeID); err != nil {
		return nil, err
	}

	comment := BuildComment(in.AuthorID, parent, in.Body)
	mentionIDs, err := s.resolveMentions(ctx, in.Body)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateComment(ctx, comment, mentionIDs); err != nil {
		return nil, err
	}
	s.notifyMentions(ctx, comment.ID, mentionIDs)
	return comment, nil
}

// GetPost fetches one post with its attachments, mentions, and whether the
// reader liked it.
func (s *PostService) GetPost(ctx context.Context, id, userID uuid.UUID, revealHidden bool) (*PostView, []PostAttachment, []MentionedUser, bool, error) {
	post, err := s.repo.GetPostByIDWithAuthor(ctx, id)
	if err != nil {
		return nil, nil, nil, false, err
	}
	if post == nil {
		return nil, nil, nil, false, ErrPostNotFound
	}
	now := time.Now()
	if !post.CanView(revealHidden, now) {
		switch {
		case IsDeleted(post.DeletedAt):
			return nil, nil, nil, false, ErrPostDeleted
		case IsScheduled(post.PublishAt, now):
			return nil, nil, nil, false, ErrPostScheduled
		default:
			return nil, nil, nil, false, ErrPostExpired
		}
	}

	attachments, err := s.repo.ListAttachmentsByPostID(ctx, id)
	if err != nil {
		return nil, nil, nil, false, err
	}
	mentions, err := s.repo.ListMentionsByPostID(ctx, id)
	if err != nil {
		return nil, nil, nil, false, err
	}
	liked, err := s.repo.LikeExists(ctx, id, userID)
	if err != nil {
		return nil, nil, nil, false, err
	}
	return post, attachments, mentions, liked, nil
}

// GetByID fetches the bare post row.
func (s *PostService) GetByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	return s.repo.GetPostByID(ctx, id)
}

// ListPosts pages through a scope's feed.
func (s *PostService) ListPosts(ctx context.Context, scopeType ScopeType, scopeID *uuid.UUID, revealHidden bool, params pagination.PageParams) ([]PostView, bool, error) {
	return s.repo.ListPostsByScope(ctx, scopeType, scopeID, revealHidden, params)
}

// ListComments pages through a thread's replies.
func (s *PostService) ListComments(ctx context.Context, rootID uuid.UUID, params pagination.PageParams) ([]PostView, bool, error) {
	return s.repo.ListComments(ctx, rootID, params)
}

// UpdatePost applies the edit under optimistic concurrency; a changed body
// re-resolves its mentions. A concurrent edit that loses the version race is
// retried against fresh state up to maxUpdateRetries times.
func (s *PostService) UpdatePost(ctx context.Context, in UpdatePostInput) (*Post, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		post, err := s.repo.GetPostByID(ctx, in.ID)
		if err != nil {
			return nil, err
		}
		if post == nil {
			return nil, ErrPostNotFound
		}

		var mentionIDs []uuid.UUID
		bodyChanged := in.Body != nil
		if bodyChanged {
			post.Body = *in.Body
			if mentionIDs, err = s.resolveMentions(ctx, *in.Body); err != nil {
				return nil, err
			}
		}
		if in.ClearSchedule {
			post.PublishAt = nil
			post.ExpiresAt = nil
		} else {
			if in.PublishAt != nil {
				post.PublishAt = in.PublishAt
			}
			if in.ExpiresAt != nil {
				post.ExpiresAt = in.ExpiresAt
			}
		}
		now := time.Now()
		post.UpdatedAt = &now

		newVersion, err := s.repo.UpdatePost(ctx, post, post.Version, bodyChanged, mentionIDs)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		post.Version = newVersion
		if bodyChanged {
			s.notifyMentions(ctx, post.ID, mentionIDs)
		}
		return post, nil
	}
	return nil, ErrConflict
}

// DeletePost soft-deletes a top-level post (its thread stays readable) and
// hard-deletes a comment together with its thread bookkeeping.
func (s *PostService) DeletePost(ctx context.Context, id uuid.UUID) error {
	post, err := s.repo.GetPostByID(ctx, id)
	if err != nil {
		return err
	}
	if post == nil {
		return ErrPostNotFound
	}

	if !post.IsComment() {
		// Soft delete keeps the row, so its attachments stay counted.
		return s.repo.SoftDeletePost(ctx, id, time.Now())
	}
	attachments, err := s.repo.ListAttachmentsByPostID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteComment(ctx, id, *post.RootID); err != nil {
		return err
	}
	for _, att := range attachments {
		s.unlink(ctx, att.InodeID)
	}
	return nil
}

// Pin sets or clears a top-level post's pin under optimistic concurrency.
// The pin authority was decided at the gate (§21). A lost version race is
// retried against fresh state.
func (s *PostService) Pin(ctx context.Context, id uuid.UUID, pin bool) error {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		post, err := s.repo.GetPostByID(ctx, id)
		if err != nil {
			return err
		}
		if post == nil {
			return ErrPostNotFound
		}
		if post.IsComment() {
			return ErrCannotPinComment
		}
		post.IsPinned = pin
		now := time.Now()
		post.UpdatedAt = &now

		if _, err := s.repo.UpdatePost(ctx, post, post.Version, false, nil); errors.Is(err, ErrConflict) {
			continue
		} else if err != nil {
			return err
		}
		return nil
	}
	return ErrConflict
}

// Like records the reader's like. The duplicate guard is the likes table's
// primary key; a race surfaces as ErrAlreadyLiked.
func (s *PostService) Like(ctx context.Context, postID, userID uuid.UUID) error {
	post, err := s.repo.GetPostByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil || !post.Visible(time.Now()) {
		return ErrPostNotFound
	}
	if err := s.checkMuted(ctx, userID, post.ScopeType, post.ScopeID); err != nil {
		return err
	}
	return s.repo.Like(ctx, postID, userID)
}

// Unlike removes the reader's like; a missing like is ErrNotLiked.
func (s *PostService) Unlike(ctx context.Context, postID, userID uuid.UUID) error {
	return s.repo.Unlike(ctx, postID, userID)
}

// UserLikesFor reports which of the given posts the user liked.
func (s *PostService) UserLikesFor(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	return s.repo.GetUserLikes(ctx, postIDs, userID)
}

// CanAccessScope reports whether the user belongs to the scope's audience.
func (s *PostService) CanAccessScope(ctx context.Context, userID uuid.UUID, scopeType ScopeType, scopeID *uuid.UUID) (bool, error) {
	return s.scopes.CanAccessScope(ctx, userID, scopeType, scopeID)
}

// AddAttachment references a file from the actor's own drive on their
// post. The reference is counted before the row exists; type and size are
// judged from the stored facts, never the request's claims.
func (s *PostService) AddAttachment(ctx context.Context, in AddPostAttachmentInput) (*PostAttachment, error) {
	post, err := s.repo.GetPostByID(ctx, in.PostID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}
	if !ValidFileType(in.FileType) {
		return nil, ErrInvalidFileType
	}
	file, err := s.files.ResolveUpload(ctx, in.ActorID, in.UploadID)
	if err != nil {
		return nil, err
	}
	if !ValidFileSize(in.FileType, file.SizeBytes) {
		return nil, ErrFileTooLarge
	}
	name := in.DisplayName
	if name == "" {
		name = file.Name
	}
	if err := s.files.Link(ctx, file.InodeID); err != nil {
		return nil, err
	}
	att := BuildPostAttachment(in.PostID, file.InodeID, name, in.FileType, in.OrderIndex)
	if err := s.repo.CreateAttachment(ctx, att); err != nil {
		s.unlink(ctx, file.InodeID)
		return nil, err
	}
	return att, nil
}

// RemoveAttachment detaches a file from a post and drops its reference
// count. The post authority was decided at the edge (§21).
func (s *PostService) RemoveAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	att, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if att == nil {
		return ErrAttachmentNotFound
	}
	if err := s.repo.DeleteAttachment(ctx, attachmentID); err != nil {
		return err
	}
	s.unlink(ctx, att.InodeID)
	return nil
}

// PresignAttachment mints a download URL for one attachment of a post the
// reader may see — whoever can read the post can save its files, and the
// author keeps access to their own scheduled or expired post.
func (s *PostService) PresignAttachment(ctx context.Context, postID, attachmentID uuid.UUID, revealHidden bool) (string, error) {
	post, err := s.repo.GetPostByID(ctx, postID)
	if err != nil {
		return "", err
	}
	if post == nil || !post.CanView(revealHidden, time.Now()) {
		return "", ErrPostNotFound
	}
	att, err := s.repo.GetAttachmentByID(ctx, attachmentID)
	if err != nil {
		return "", err
	}
	if att == nil || att.PostID != postID {
		return "", ErrAttachmentNotFound
	}
	return s.files.Presign(ctx, att.InodeID, att.DisplayName)
}

// unlink drops one reference count. A failed Unlink over-counts — a leaked
// blob the sweeper cannot reclaim, never a lost one — so it is logged, not
// fatal.
func (s *PostService) unlink(ctx context.Context, inodeID uuid.UUID) {
	if err := s.files.Unlink(ctx, inodeID); err != nil {
		s.log.WarnContext(ctx, "announcements: unlink failed; blob over-counted", "inode", inodeID, "error", err)
	}
}

// AttachmentByID fetches one attachment.
func (s *PostService) AttachmentByID(ctx context.Context, id uuid.UUID) (*PostAttachment, error) {
	return s.repo.GetAttachmentByID(ctx, id)
}

// AttachmentsFor returns the attachments of many posts keyed by post ID.
func (s *PostService) AttachmentsFor(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostAttachment, error) {
	return s.repo.ListAttachmentsByPostIDs(ctx, postIDs)
}

// MentionsFor returns the mentions of many posts keyed by post ID.
func (s *PostService) MentionsFor(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]MentionedUser, error) {
	return s.repo.ListMentionsByPostIDs(ctx, postIDs)
}

// resolveMentions parses @usernames and resolves them to user IDs (read-only,
// runs outside the transaction).
func (s *PostService) resolveMentions(ctx context.Context, body string) ([]uuid.UUID, error) {
	usernames := ParseMentions(body)
	if len(usernames) == 0 {
		return nil, nil
	}
	idMap, err := s.users.GetUserIDsByUsernames(ctx, usernames)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(idMap))
	for _, id := range idMap {
		ids = append(ids, id)
	}
	return ids, nil
}

// notifyMentions fires after the write commits — never notify on a write that
// rolled back. Advisory: failures are logged, never returned.
func (s *PostService) notifyMentions(ctx context.Context, postID uuid.UUID, userIDs []uuid.UUID) {
	if s.notifier == nil || len(userIDs) == 0 {
		return
	}
	if err := s.notifier.SendBulk(ctx, userIDs, "mentioned", "You were mentioned", nil, map[string]any{"post_id": postID}); err != nil {
		s.log.WarnContext(ctx, "mention notification failed", "post_id", postID, "error", err)
	}
}

func (s *PostService) checkMuted(ctx context.Context, userID uuid.UUID, scopeType ScopeType, scopeID *uuid.UUID) error {
	if s.mutes == nil {
		return nil
	}
	var offeringID *uuid.UUID
	if scopeType == ScopeOffering {
		offeringID = scopeID
	}
	muted, err := s.mutes.IsMuted(ctx, userID, offeringID)
	if err != nil {
		return err
	}
	if muted {
		return ErrUserMuted
	}
	return nil
}
