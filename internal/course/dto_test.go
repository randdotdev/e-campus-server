package course

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToCourseResponse(t *testing.T) {
	now := time.Now()
	reqID := uuid.New()
	deptID := uuid.New()

	course := &Course{
		ID:            uuid.New(),
		DepartmentID:  deptID,
		Code:          "CS101",
		NameEN:        "Intro to CS",
		NameLocal:     ptr("زانستی کۆمپیوتەر"),
		SubtitleEN:    ptr("Basics"),
		SubtitleLocal: ptr("بنەماکان"),
		GroupOrder:    1,
		Requires:      &reqID,
		Credits:       6,
		DescriptionEN: ptr("Course description"),
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	resp := ToCourseResponse(course)

	if resp.ID != course.ID {
		t.Errorf("ID = %v, want %v", resp.ID, course.ID)
	}
	if resp.DepartmentID != course.DepartmentID {
		t.Errorf("DepartmentID = %v, want %v", resp.DepartmentID, course.DepartmentID)
	}
	if resp.Code != course.Code {
		t.Errorf("Code = %v, want %v", resp.Code, course.Code)
	}
	if resp.NameEN != course.NameEN {
		t.Errorf("NameEN = %v, want %v", resp.NameEN, course.NameEN)
	}
	if *resp.SubtitleEN != *course.SubtitleEN {
		t.Errorf("SubtitleEN = %v, want %v", *resp.SubtitleEN, *course.SubtitleEN)
	}
	if resp.GroupOrder != course.GroupOrder {
		t.Errorf("GroupOrder = %v, want %v", resp.GroupOrder, course.GroupOrder)
	}
	if *resp.Requires != *course.Requires {
		t.Errorf("Requires = %v, want %v", *resp.Requires, *course.Requires)
	}
	if resp.Credits != course.Credits {
		t.Errorf("Credits = %v, want %v", resp.Credits, course.Credits)
	}
	if resp.IsActive != course.IsActive {
		t.Errorf("IsActive = %v, want %v", resp.IsActive, course.IsActive)
	}
}

func TestToCoursesResponse(t *testing.T) {
	courses := []Course{
		{ID: uuid.New(), Code: "CS101", NameEN: "Course 1"},
		{ID: uuid.New(), Code: "CS102", NameEN: "Course 2"},
	}

	resp := ToCoursesResponse(courses)

	if len(resp) != 2 {
		t.Fatalf("len = %d, want 2", len(resp))
	}
	if resp[0].Code != "CS101" {
		t.Errorf("resp[0].Code = %v, want CS101", resp[0].Code)
	}
	if resp[1].Code != "CS102" {
		t.Errorf("resp[1].Code = %v, want CS102", resp[1].Code)
	}
}

func TestToOfferingResponse(t *testing.T) {
	now := time.Now()
	offering := &Offering{
		ID:         uuid.New(),
		CourseID:   uuid.New(),
		SemesterID: uuid.New(),
		CohortYear: 2024,
		Shift:      ShiftDay,
		IsActive:   true,
		CreatedAt:  now,
	}

	resp := ToOfferingResponse(offering)

	if resp.ID != offering.ID {
		t.Errorf("ID = %v, want %v", resp.ID, offering.ID)
	}
	if resp.CohortYear != 2024 {
		t.Errorf("CohortYear = %v, want 2024", resp.CohortYear)
	}
	if resp.Shift != ShiftDay {
		t.Errorf("Shift = %v, want %v", resp.Shift, ShiftDay)
	}
}

func TestToTeacherResponse(t *testing.T) {
	now := time.Now()
	teacher := &Teacher{
		ID:         uuid.New(),
		OfferingID: uuid.New(),
		UserID:     uuid.New(),
		Role:       TeacherRoleTeacher,
		CreatedAt:  now,
	}

	resp := ToTeacherResponse(teacher)

	if resp.ID != teacher.ID {
		t.Errorf("ID = %v, want %v", resp.ID, teacher.ID)
	}
	if resp.Role != TeacherRoleTeacher {
		t.Errorf("Role = %v, want %v", resp.Role, TeacherRoleTeacher)
	}
}

func TestToSectionResponse(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	past := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		unlockAt   *time.Time
		wantLocked bool
	}{
		{"nil unlock is open", nil, false},
		{"past unlock is open", &past, false},
		{"future unlock is locked", &future, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			section := &Section{
				ID:         uuid.New(),
				OfferingID: uuid.New(),
				Title:      "Week 1",
				OrderIndex: 0,
				UnlockAt:   tt.unlockAt,
				CreatedAt:  now,
			}

			resp := ToSectionResponse(section, now)

			if resp.IsUnlocked == tt.wantLocked {
				t.Errorf("IsUnlocked = %v, want %v", resp.IsUnlocked, !tt.wantLocked)
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
