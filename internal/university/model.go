// Package university handles academic structure: colleges, departments, programs.
package university

import (
	"time"

	"github.com/google/uuid"
)

type College struct {
	ID          uuid.UUID `db:"id"`
	NameEN      string    `db:"name_en"`
	NameLocal      *string   `db:"name_ku"`
	Code        string    `db:"code"`
	Description *string   `db:"description"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type Department struct {
	ID          uuid.UUID `db:"id"`
	CollegeID   uuid.UUID `db:"college_id"`
	NameEN      string    `db:"name_en"`
	NameLocal      *string   `db:"name_ku"`
	Code        string    `db:"code"`
	Description *string   `db:"description"`
	IsActive    bool      `db:"is_active"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

type Program struct {
	ID            uuid.UUID `db:"id"`
	DepartmentID  uuid.UUID `db:"department_id"`
	NameEN        string    `db:"name_en"`
	NameLocal        *string   `db:"name_ku"`
	Code          string    `db:"code"`
	DegreeType    string    `db:"degree_type"`
	DurationYears int       `db:"duration_years"`
	TotalCredits     int       `db:"total_credits"`
	MinAge        *int      `db:"min_age"`
	MaxAge        *int      `db:"max_age"`
	Description   *string   `db:"description"`
	IsActive      bool      `db:"is_active"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}
