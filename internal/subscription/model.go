// Package subscription handles subscription and tier management.
package subscription

import (
	"time"

	"github.com/google/uuid"
)

type TierLimits struct {
	Tier                     string    `db:"tier"`
	MaxColleges              int       `db:"max_colleges"`
	MaxDepartmentsPerCollege int       `db:"max_departments_per_college"`
	MaxProgramsPerDepartment int       `db:"max_programs_per_department"`
	MaxStudentsPerProgram    int       `db:"max_students_per_program"`
	MaxApplicationsPerUser   int       `db:"max_applications_per_user"`
	MaxStaffUsers            int       `db:"max_staff_users"`
	MaxStorageBytes          int64     `db:"max_storage_bytes"`
	MaxFileSizeBytes         int64     `db:"max_file_size_bytes"`
	UpdatedAt                time.Time `db:"updated_at"`
}

type Subscription struct {
	ID                      uuid.UUID  `db:"id"`
	Tier                    string     `db:"tier"`
	MaxCollegesOverride     *int       `db:"max_colleges_override"`
	MaxDepartmentsOverride  *int       `db:"max_departments_override"`
	MaxProgramsOverride     *int       `db:"max_programs_override"`
	MaxStudentsOverride     *int       `db:"max_students_override"`
	MaxApplicationsOverride *int       `db:"max_applications_override"`
	MaxStaffOverride        *int       `db:"max_staff_override"`
	MaxStorageOverride      *int64     `db:"max_storage_override"`
	MaxFileSizeOverride     *int64     `db:"max_file_size_override"`
	ExpiresAt               *time.Time `db:"expires_at"`
	UpdatedBy               *uuid.UUID `db:"updated_by"`
	UpdatedAt               time.Time  `db:"updated_at"`
	CreatedAt               time.Time  `db:"created_at"`
}

type History struct {
	ID                      uuid.UUID  `db:"id"`
	Tier                    string     `db:"tier"`
	MaxCollegesOverride     *int       `db:"max_colleges_override"`
	MaxDepartmentsOverride  *int       `db:"max_departments_override"`
	MaxProgramsOverride     *int       `db:"max_programs_override"`
	MaxStudentsOverride     *int       `db:"max_students_override"`
	MaxApplicationsOverride *int       `db:"max_applications_override"`
	MaxStaffOverride        *int       `db:"max_staff_override"`
	MaxStorageOverride      *int64     `db:"max_storage_override"`
	MaxFileSizeOverride     *int64     `db:"max_file_size_override"`
	ExpiresAt               *time.Time `db:"expires_at"`
	ChangedBy               *uuid.UUID `db:"changed_by"`
	ChangedAt               time.Time  `db:"changed_at"`
	ChangeReason            *string    `db:"change_reason"`
}

const (
	TierFree    = "free"
	TierBasic   = "basic"
	TierPremium = "premium"
)
