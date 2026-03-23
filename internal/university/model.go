// Package university handles academic structure: colleges, departments, programs.
package university

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// LocalizedText stores text in multiple languages as JSONB.
// Example: {"en": "About us", "ku": "دەربارەی ئێمە"}
type LocalizedText map[string]string

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

type College struct {
	ID          uuid.UUID `db:"id"`
	NameEN      string    `db:"name_en"`
	NameLocal   *string   `db:"name_local"`
	Code        string    `db:"code"`
	Description *string   `db:"description"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`

	About   LocalizedText `db:"about"`
	Founded *int          `db:"founded"`
	Phone   *string       `db:"phone"`
	Email   *string       `db:"email"`
	LogoURL *string       `db:"logo_url"`
}

type Department struct {
	ID          uuid.UUID `db:"id"`
	CollegeID   uuid.UUID `db:"college_id"`
	NameEN      string    `db:"name_en"`
	NameLocal   *string   `db:"name_local"`
	Code        string    `db:"code"`
	Description *string   `db:"description"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`

	About   LocalizedText `db:"about"`
	Founded *int          `db:"founded"`
	Phone   *string       `db:"phone"`
	Email   *string       `db:"email"`
	LogoURL *string       `db:"logo_url"`
}

type Program struct {
	ID            uuid.UUID `db:"id"`
	DepartmentID  uuid.UUID `db:"department_id"`
	NameEN        string    `db:"name_en"`
	NameLocal     *string   `db:"name_local"`
	Code          string    `db:"code"`
	DegreeType    string    `db:"degree_type"`
	DurationYears int       `db:"duration_years"`
	TotalCredits  int       `db:"total_credits"`
	MinAge        *int      `db:"min_age"`
	MaxAge        *int      `db:"max_age"`
	Description   *string   `db:"description"`
	IsActive      bool      `db:"is_active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
