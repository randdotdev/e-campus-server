package management

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// maxUpdateRetries bounds the optimistic-concurrency retry loop used by every
// single-row entity edit in this context (read → apply → compare-and-swap).
const maxUpdateRetries = 3

// ── Value objects ─────────────────────────────────────────────────────────────

// LocalizedText holds the localized variants of a single text field — a name,
// title, or description — keyed by language code and stored as JSONB. "en" is
// the canonical fallback; other keys carry translations (e.g. Kurdish,
// Arabic). Example: {"en": "Computer Science", "ku": "زانستی کۆمپیوتەر"}
type LocalizedText map[string]string

// Scan implements sql.Scanner for the JSONB column.
func (l *LocalizedText) Scan(src any) error {
	if src == nil {
		*l = make(LocalizedText)
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, l)
	case string:
		return json.Unmarshal([]byte(v), l)
	}
	*l = make(LocalizedText)
	return nil
}

// Value implements driver.Valuer for the JSONB column.
func (l LocalizedText) Value() (driver.Value, error) {
	if l == nil {
		return json.Marshal(map[string]string{})
	}
	return json.Marshal(l)
}

// Get returns text for the given language with fallback to English.
func (l LocalizedText) Get(lang string) string {
	if l == nil {
		return ""
	}
	if v, ok := l[lang]; ok && v != "" {
		return v
	}
	return l["en"]
}

// Limits is the subset of subscription tier limits the structure services
// enforce.
type Limits struct {
	MaxColleges              int
	MaxDepartmentsPerCollege int
	MaxProgramsPerDepartment int
}

// LimitsProvider supplies the institution's current resource limits.
type LimitsProvider interface {
	GetLimits(ctx context.Context) (Limits, error)
}

// ── Entities ──────────────────────────────────────────────────────────────────

// College is the top level of the university structure.
type College struct {
	ID          uuid.UUID     `db:"id"`
	NameEN      string        `db:"name_en"`
	NameLocal   *string       `db:"name_local"`
	Code        string        `db:"code"`
	Description LocalizedText `db:"description"`
	IsActive    bool          `db:"is_active"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
	Version     int64         `db:"version"`

	About   LocalizedText `db:"about"`
	Founded *int          `db:"founded"`
	Phone   *string       `db:"phone"`
	Email   *string       `db:"email"`
	LogoURL *string       `db:"logo_url"`
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// ValidCode reports whether a structure code is well-formed: 2–20 characters
// of letters, digits, and underscores.
func ValidCode(code string) bool {
	if len(code) < 2 || len(code) > 20 {
		return false
	}
	for _, r := range code {
		if !isAlphanumOrUnderscore(r) {
			return false
		}
	}
	return true
}

func isAlphanumOrUnderscore(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}

// CanCreateWithinLimit reports whether another entity may be created under
// the given subscription limit.
func CanCreateWithinLimit(currentCount, limit int) bool { return currentCount < limit }

// ── Ports ─────────────────────────────────────────────────────────────────────

// CollegeRepository persists colleges.
//
// GetCollege returns ErrCollegeNotFound. UpdateCollege is an optimistic
// compare-and-swap keyed on version (zero rows → ErrConflict), the
// cross-replica guard against lost updates.
type CollegeRepository interface {
	CreateCollege(ctx context.Context, college *College) error
	GetCollege(ctx context.Context, id uuid.UUID) (*College, error)
	ListColleges(ctx context.Context, params pagination.PageParams, filter CollegeFilter) ([]College, bool, error)
	UpdateCollege(ctx context.Context, college *College, expectedVersion int64) (int64, error)
	CollegeCodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error)
	CountColleges(ctx context.Context) (int, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// CollegeFilter narrows college lists; nil fields are ignored.
type CollegeFilter struct {
	IsActive *bool
}

// CollegeUpdate is a partial edit of a college; nil fields are left
// unchanged.
type CollegeUpdate struct {
	NameEN      *string
	NameLocal   *string
	Code        *string
	Description LocalizedText
	IsActive    *bool
	About       LocalizedText
	Founded     *int
	Phone       *string
	Email       *string
	LogoURL     *string
}

// ── Service ───────────────────────────────────────────────────────────────────

// CollegeService manages colleges under the institution's subscription
// limits.
type CollegeService struct {
	repo   CollegeRepository
	limits LimitsProvider
}

// NewCollegeService wires a college service.
func NewCollegeService(repo CollegeRepository, limits LimitsProvider) *CollegeService {
	return &CollegeService{repo: repo, limits: limits}
}

// Create adds a college, enforcing the subscription's college limit and code
// uniqueness.
func (s *CollegeService) Create(ctx context.Context, college *College) (*College, error) {
	limits, err := s.limits.GetLimits(ctx)
	if err != nil {
		return nil, err
	}
	count, err := s.repo.CountColleges(ctx)
	if err != nil {
		return nil, err
	}
	if !CanCreateWithinLimit(count, limits.MaxColleges) {
		return nil, ErrCollegeLimitReached
	}

	exists, err := s.repo.CollegeCodeExists(ctx, college.Code, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCodeExists
	}

	if err := s.repo.CreateCollege(ctx, college); err != nil {
		return nil, err
	}
	return college, nil
}

// Get fetches one college.
func (s *CollegeService) Get(ctx context.Context, id uuid.UUID) (*College, error) {
	return s.repo.GetCollege(ctx, id)
}

// List pages through colleges matching the filter.
func (s *CollegeService) List(ctx context.Context, params pagination.PageParams, filter CollegeFilter) ([]College, bool, error) {
	return s.repo.ListColleges(ctx, params, filter)
}

// Update applies the patch under optimistic concurrency: each attempt
// re-reads the current row, re-applies the patch, and compare-and-swaps on
// version, so concurrent edits to different fields merge instead of
// clobbering one another.
func (s *CollegeService) Update(ctx context.Context, id uuid.UUID, upd CollegeUpdate) (*College, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		college, err := s.repo.GetCollege(ctx, id)
		if err != nil {
			return nil, err
		}

		if upd.Code != nil && *upd.Code != college.Code {
			exists, err := s.repo.CollegeCodeExists(ctx, *upd.Code, &id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrCodeExists
			}
			college.Code = *upd.Code
		}

		applyCollegePatch(college, upd)

		newVersion, err := s.repo.UpdateCollege(ctx, college, college.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		college.Version = newVersion
		return college, nil
	}
	return nil, ErrConflict
}

func applyCollegePatch(college *College, upd CollegeUpdate) {
	if upd.NameEN != nil {
		college.NameEN = *upd.NameEN
	}
	if upd.NameLocal != nil {
		college.NameLocal = upd.NameLocal
	}
	if upd.Description != nil {
		college.Description = upd.Description
	}
	if upd.IsActive != nil {
		college.IsActive = *upd.IsActive
	}
	if upd.About != nil {
		college.About = upd.About
	}
	if upd.Founded != nil {
		college.Founded = upd.Founded
	}
	if upd.Phone != nil {
		college.Phone = upd.Phone
	}
	if upd.Email != nil {
		college.Email = upd.Email
	}
	if upd.LogoURL != nil {
		college.LogoURL = upd.LogoURL
	}
}
