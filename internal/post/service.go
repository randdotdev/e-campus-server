package post

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type PostRepository interface {
	Create(ctx context.Context, p *Post) error
	GetByID(ctx context.Context, id uuid.UUID) (*Post, error)
	GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*postView, error)
	Update(ctx context.Context, p *Post) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
	HardDelete(ctx context.Context, id uuid.UUID) error

	ListByScope(ctx context.Context, scopeType string, scopeID *uuid.UUID, isAdmin bool, params pagination.PageParams) ([]postView, bool, error)
	ListComments(ctx context.Context, rootID uuid.UUID, params pagination.PageParams) ([]postView, bool, error)

	IncrementLikeCount(ctx context.Context, id uuid.UUID) error
	DecrementLikeCount(ctx context.Context, id uuid.UUID) error
	IncrementCommentCount(ctx context.Context, id uuid.UUID) error
	DecrementCommentCount(ctx context.Context, id uuid.UUID) error
}

type LikeRepository interface {
	Create(ctx context.Context, postID, userID uuid.UUID) error
	Delete(ctx context.Context, postID, userID uuid.UUID) error
	Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error)
	GetUserLikes(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error)
}

type AttachmentRepository interface {
	Create(ctx context.Context, a *PostAttachment) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*PostAttachment, error)
	ListByPostID(ctx context.Context, postID uuid.UUID) ([]PostAttachment, error)
	ListByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostAttachment, error)
}

type MentionRepository interface {
	CreateBatch(ctx context.Context, postID uuid.UUID, userIDs []uuid.UUID) error
	DeleteByPostID(ctx context.Context, postID uuid.UUID) error
	ListByPostID(ctx context.Context, postID uuid.UUID) ([]MentionedUser, error)
	ListByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]MentionedUser, error)
}

type UserLookup interface {
	GetUserIDsByUsernames(ctx context.Context, usernames []string) (map[string]uuid.UUID, error)
}

type ScopeChecker interface {
	CanAccessScope(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) (bool, error)
	ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error)
}

type MuteChecker interface {
	IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error)
}

type Notifier interface {
	SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

type Service struct {
	posts       PostRepository
	likes       LikeRepository
	attachments AttachmentRepository
	mentions    MentionRepository
	users       UserLookup
	scopes      ScopeChecker
	mutes       MuteChecker
	notifier    Notifier
}

func NewService(
	posts PostRepository,
	likes LikeRepository,
	attachments AttachmentRepository,
	mentions MentionRepository,
	users UserLookup,
	scopes ScopeChecker,
	mutes MuteChecker,
	notifier Notifier,
) *Service {
	return &Service{
		posts:       posts,
		likes:       likes,
		attachments: attachments,
		mentions:    mentions,
		users:       users,
		scopes:      scopes,
		mutes:       mutes,
		notifier:    notifier,
	}
}

func (s *Service) CreatePost(ctx context.Context, authorID uuid.UUID, scopeType string, scopeID *uuid.UUID, body string, publishAt, expiresAt *time.Time) (*Post, error) {
	if !ValidateScopeType(scopeType) {
		return nil, ErrInvalidScope
	}
	if !ValidateScopeID(scopeType, scopeID) {
		return nil, ErrInvalidScope
	}

	if scopeID != nil {
		exists, err := s.scopes.ScopeExists(ctx, scopeType, *scopeID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrInvalidScope
		}
	}

	post := BuildPost(authorID, scopeType, scopeID, body, publishAt, expiresAt)

	if err := s.posts.Create(ctx, post); err != nil {
		return nil, err
	}

	if err := s.processMentions(ctx, post.ID, body); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *Service) CreateComment(ctx context.Context, authorID, parentID uuid.UUID, body string) (*Post, error) {
	parent, err := s.posts.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrPostNotFound
	}

	now := time.Now()
	if !IsVisible(parent, now) {
		return nil, ErrPostNotFound
	}

	if err := s.checkMuted(ctx, authorID, parent.ScopeType, parent.ScopeID); err != nil {
		return nil, err
	}

	comment := BuildComment(authorID, parent, body)

	if err := s.posts.Create(ctx, comment); err != nil {
		return nil, err
	}

	// Increment on root post, not immediate parent
	rootID := parent.ID
	if parent.RootID != nil {
		rootID = *parent.RootID
	}
	if err := s.posts.IncrementCommentCount(ctx, rootID); err != nil {
		return nil, err
	}

	if err := s.processMentions(ctx, comment.ID, body); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *Service) GetPost(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) (*postView, []PostAttachment, []MentionedUser, bool, error) {
	post, err := s.posts.GetByIDWithAuthor(ctx, id)
	if err != nil {
		return nil, nil, nil, false, err
	}
	if post == nil {
		return nil, nil, nil, false, ErrPostNotFound
	}

	now := time.Now()
	if !CanView(&post.Post, isAdmin, now) {
		if IsDeleted(post.DeletedAt) {
			return nil, nil, nil, false, ErrPostDeleted
		}
		if IsScheduled(post.PublishAt, now) {
			return nil, nil, nil, false, ErrPostScheduled
		}
		return nil, nil, nil, false, ErrPostExpired
	}

	attachments, err := s.attachments.ListByPostID(ctx, id)
	if err != nil {
		return nil, nil, nil, false, err
	}

	mentions, err := s.mentions.ListByPostID(ctx, id)
	if err != nil {
		return nil, nil, nil, false, err
	}

	liked, err := s.likes.Exists(ctx, id, userID)
	if err != nil {
		return nil, nil, nil, false, err
	}

	return post, attachments, mentions, liked, nil
}

func (s *Service) ListPosts(ctx context.Context, scopeType string, scopeID *uuid.UUID, isAdmin bool, params pagination.PageParams) ([]postView, bool, error) {
	return s.posts.ListByScope(ctx, scopeType, scopeID, isAdmin, params)
}

func (s *Service) ListComments(ctx context.Context, rootID uuid.UUID, params pagination.PageParams) ([]postView, bool, error) {
	return s.posts.ListComments(ctx, rootID, params)
}

func (s *Service) UpdatePost(ctx context.Context, id, userID uuid.UUID, isAdmin bool, body *string, publishAt, expiresAt *time.Time, clearSchedule bool) (*Post, error) {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}

	if !CanEdit(post, userID, isAdmin) {
		return nil, ErrNotAuthorized
	}

	if body != nil {
		post.Body = *body
		if err := s.mentions.DeleteByPostID(ctx, id); err != nil {
			return nil, err
		}
		if err := s.processMentions(ctx, id, *body); err != nil {
			return nil, err
		}
	}
	if clearSchedule {
		post.PublishAt = nil
		post.ExpiresAt = nil
	} else {
		if publishAt != nil {
			post.PublishAt = publishAt
		}
		if expiresAt != nil {
			post.ExpiresAt = expiresAt
		}
	}

	now := time.Now()
	post.UpdatedAt = &now

	if err := s.posts.Update(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *Service) DeletePost(ctx context.Context, id, userID uuid.UUID, isAdmin bool) error {
	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post == nil {
		return ErrPostNotFound
	}

	if !CanDelete(post, userID, isAdmin) {
		return ErrNotAuthorized
	}

	if IsComment(post) {
		if err := s.mentions.DeleteByPostID(ctx, id); err != nil {
			return err
		}
		if err := s.posts.HardDelete(ctx, id); err != nil {
			return err
		}
		if err := s.posts.DecrementCommentCount(ctx, *post.RootID); err != nil {
			return err
		}
	} else {
		now := time.Now()
		if err := s.posts.SoftDelete(ctx, id, now); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) PinPost(ctx context.Context, id uuid.UUID, isAdmin bool, pin bool) error {
	if !CanPin(isAdmin) {
		return ErrNotAuthorized
	}

	post, err := s.posts.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post == nil {
		return ErrPostNotFound
	}

	if IsComment(post) {
		return ErrCannotPinComment
	}

	post.IsPinned = pin
	now := time.Now()
	post.UpdatedAt = &now

	return s.posts.Update(ctx, post)
}

func (s *Service) LikePost(ctx context.Context, postID, userID uuid.UUID) error {
	post, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return err
	}
	if post == nil {
		return ErrPostNotFound
	}

	now := time.Now()
	if !IsVisible(post, now) {
		return ErrPostNotFound
	}

	if err := s.checkMuted(ctx, userID, post.ScopeType, post.ScopeID); err != nil {
		return err
	}

	exists, err := s.likes.Exists(ctx, postID, userID)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyLiked
	}

	if err := s.likes.Create(ctx, postID, userID); err != nil {
		return err
	}

	return s.posts.IncrementLikeCount(ctx, postID)
}

func (s *Service) UnlikePost(ctx context.Context, postID, userID uuid.UUID) error {
	exists, err := s.likes.Exists(ctx, postID, userID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotLiked
	}

	if err := s.likes.Delete(ctx, postID, userID); err != nil {
		return err
	}

	return s.posts.DecrementLikeCount(ctx, postID)
}

func (s *Service) AddAttachment(ctx context.Context, postID, userID uuid.UUID, isAdmin bool, storedFileID uuid.UUID, displayName, fileType string, sizeBytes int64, orderIndex int) (*PostAttachment, error) {
	post, err := s.posts.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}

	if !CanEdit(post, userID, isAdmin) {
		return nil, ErrNotAuthorized
	}

	if !ValidateFileType(fileType) {
		return nil, ErrInvalidFileType
	}

	if !ValidateFileSize(fileType, sizeBytes) {
		return nil, ErrFileTooLarge
	}

	attachment := BuildAttachment(postID, storedFileID, displayName, fileType, orderIndex)

	if err := s.attachments.Create(ctx, attachment); err != nil {
		return nil, err
	}

	return attachment, nil
}

func (s *Service) RemoveAttachment(ctx context.Context, attachmentID, userID uuid.UUID, isAdmin bool) error {
	attachment, err := s.attachments.GetByID(ctx, attachmentID)
	if err != nil {
		return err
	}
	if attachment == nil {
		return ErrAttachmentNotFound
	}

	post, err := s.posts.GetByID(ctx, attachment.PostID)
	if err != nil {
		return err
	}
	if post == nil {
		return ErrPostNotFound
	}

	if !CanEdit(post, userID, isAdmin) {
		return ErrNotAuthorized
	}

	return s.attachments.Delete(ctx, attachmentID)
}

func (s *Service) GetAttachmentsForPosts(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostAttachment, error) {
	return s.attachments.ListByPostIDs(ctx, postIDs)
}

func (s *Service) GetMentionsForPosts(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]MentionedUser, error) {
	return s.mentions.ListByPostIDs(ctx, postIDs)
}

func (s *Service) GetUserLikesForPosts(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	return s.likes.GetUserLikes(ctx, postIDs, userID)
}

func (s *Service) GetPostByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	return s.posts.GetByID(ctx, id)
}

func (s *Service) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*PostAttachment, error) {
	return s.attachments.GetByID(ctx, id)
}

func (s *Service) CanAccessScope(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) (bool, error) {
	return s.scopes.CanAccessScope(ctx, userID, scopeType, scopeID)
}

func (s *Service) processMentions(ctx context.Context, postID uuid.UUID, body string) error {
	usernames := ParseMentions(body)
	if len(usernames) == 0 {
		return nil
	}

	userIDMap, err := s.users.GetUserIDsByUsernames(ctx, usernames)
	if err != nil {
		return err
	}

	var userIDs []uuid.UUID
	for _, id := range userIDMap {
		userIDs = append(userIDs, id)
	}

	if len(userIDs) == 0 {
		return nil
	}

	if err := s.mentions.CreateBatch(ctx, postID, userIDs); err != nil {
		return err
	}

	if s.notifier != nil {
		title := "You were mentioned"
		_ = s.notifier.SendBulk(ctx, userIDs, "mentioned", title, nil, map[string]any{
			"post_id": postID,
		})
	}

	return nil
}

func (s *Service) checkMuted(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) error {
	if s.mutes == nil {
		return nil
	}

	var offeringID *uuid.UUID
	if scopeType == ScopeCourse {
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
