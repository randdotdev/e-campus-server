package academic

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToAcademicYearResponse(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if got := ToAcademicYearResponse(nil); got != nil {
			t.Errorf("ToAcademicYearResponse(nil) = %v, want nil", got)
		}
	})

	t.Run("valid academic year", func(t *testing.T) {
		id := uuid.New()
		start := time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC)
		created := time.Now()

		ay := &AcademicYear{
			ID:        id,
			Year:      2022,
			StartDate: start,
			EndDate:   end,
			Status:    AcademicYearStatusActive,
			CreatedAt: created,
		}

		resp := ToAcademicYearResponse(ay)

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
		if resp.Year != 2022 {
			t.Errorf("Year = %d, want 2022", resp.Year)
		}
		if resp.StartDate != "2022-09-01" {
			t.Errorf("StartDate = %s, want 2022-09-01", resp.StartDate)
		}
		if resp.EndDate != "2023-06-30" {
			t.Errorf("EndDate = %s, want 2023-06-30", resp.EndDate)
		}
		if resp.Status != AcademicYearStatusActive {
			t.Errorf("Status = %s, want %s", resp.Status, AcademicYearStatusActive)
		}
	})
}

func TestToAcademicYearsResponse(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := ToAcademicYearsResponse([]AcademicYear{})
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("multiple years", func(t *testing.T) {
		now := time.Now()
		years := []AcademicYear{
			{ID: uuid.New(), Year: 2022, StartDate: now, EndDate: now, Status: AcademicYearStatusArchived, CreatedAt: now},
			{ID: uuid.New(), Year: 2023, StartDate: now, EndDate: now, Status: AcademicYearStatusActive, CreatedAt: now},
		}

		result := ToAcademicYearsResponse(years)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].Year != 2022 {
			t.Errorf("result[0].Year = %d, want 2022", result[0].Year)
		}
		if result[1].Year != 2023 {
			t.Errorf("result[1].Year = %d, want 2023", result[1].Year)
		}
	})
}

func TestToSemesterResponse(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if got := ToSemesterResponse(nil); got != nil {
			t.Errorf("ToSemesterResponse(nil) = %v, want nil", got)
		}
	})

	t.Run("semester without optional fields", func(t *testing.T) {
		id := uuid.New()
		ayID := uuid.New()
		start := time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
		created := time.Now()

		s := &Semester{
			ID:             id,
			AcademicYearID: ayID,
			Semester:       SemesterTypeFall,
			StartDate:      start,
			EndDate:        end,
			PassThreshold:  50,
			Status:         SemesterStatusActive,
			CreatedAt:      created,
		}

		resp := ToSemesterResponse(s)

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
		if resp.AcademicYearID != ayID {
			t.Errorf("AcademicYearID = %v, want %v", resp.AcademicYearID, ayID)
		}
		if resp.Semester != SemesterTypeFall {
			t.Errorf("Semester = %s, want %s", resp.Semester, SemesterTypeFall)
		}
		if resp.StartDate != "2022-09-01" {
			t.Errorf("StartDate = %s, want 2022-09-01", resp.StartDate)
		}
		if resp.PassThreshold != 50 {
			t.Errorf("PassThreshold = %d, want 50", resp.PassThreshold)
		}
		if resp.RegistrationStart != nil {
			t.Errorf("RegistrationStart = %v, want nil", resp.RegistrationStart)
		}
	})

	t.Run("semester with all optional fields", func(t *testing.T) {
		id := uuid.New()
		ayID := uuid.New()
		start := time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
		regStart := time.Date(2022, 8, 15, 0, 0, 0, 0, time.UTC)
		regEnd := time.Date(2022, 8, 30, 0, 0, 0, 0, time.UTC)
		gradeStart := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)
		gradeEnd := time.Date(2023, 1, 20, 0, 0, 0, 0, time.UTC)
		created := time.Now()

		s := &Semester{
			ID:                id,
			AcademicYearID:    ayID,
			Semester:          SemesterTypeSpring,
			StartDate:         start,
			EndDate:           end,
			RegistrationStart: &regStart,
			RegistrationEnd:   &regEnd,
			GradeEntryStart:   &gradeStart,
			GradeEntryEnd:     &gradeEnd,
			PassThreshold:     60,
			Status:            SemesterStatusGrading,
			CreatedAt:         created,
		}

		resp := ToSemesterResponse(s)

		if resp.RegistrationStart == nil || *resp.RegistrationStart != "2022-08-15" {
			t.Errorf("RegistrationStart = %v, want 2022-08-15", resp.RegistrationStart)
		}
		if resp.RegistrationEnd == nil || *resp.RegistrationEnd != "2022-08-30" {
			t.Errorf("RegistrationEnd = %v, want 2022-08-30", resp.RegistrationEnd)
		}
		if resp.GradeEntryStart == nil || *resp.GradeEntryStart != "2023-01-10" {
			t.Errorf("GradeEntryStart = %v, want 2023-01-10", resp.GradeEntryStart)
		}
		if resp.GradeEntryEnd == nil || *resp.GradeEntryEnd != "2023-01-20" {
			t.Errorf("GradeEntryEnd = %v, want 2023-01-20", resp.GradeEntryEnd)
		}
	})
}

func TestToSemestersResponse(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := ToSemestersResponse([]Semester{})
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("multiple semesters", func(t *testing.T) {
		now := time.Now()
		ayID := uuid.New()
		semesters := []Semester{
			{ID: uuid.New(), AcademicYearID: ayID, Semester: SemesterTypeFall, StartDate: now, EndDate: now, Status: SemesterStatusFinalized, CreatedAt: now},
			{ID: uuid.New(), AcademicYearID: ayID, Semester: SemesterTypeSpring, StartDate: now, EndDate: now, Status: SemesterStatusActive, CreatedAt: now},
		}

		result := ToSemestersResponse(semesters)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].Semester != SemesterTypeFall {
			t.Errorf("result[0].Semester = %s, want %s", result[0].Semester, SemesterTypeFall)
		}
		if result[1].Semester != SemesterTypeSpring {
			t.Errorf("result[1].Semester = %s, want %s", result[1].Semester, SemesterTypeSpring)
		}
	})
}

func TestToCurriculumResponse(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if got := ToCurriculumResponse(nil); got != nil {
			t.Errorf("ToCurriculumResponse(nil) = %v, want nil", got)
		}
	})

	t.Run("valid curriculum", func(t *testing.T) {
		id := uuid.New()
		programID := uuid.New()
		courseID := uuid.New()
		created := time.Now()

		c := &Curriculum{
			ID:         id,
			ProgramID:  programID,
			CohortYear: 2022,
			Stage:      2,
			Semester:   SemesterTypeFall,
			CourseID:   courseID,
			IsRequired: true,
			CreatedAt:  created,
		}

		resp := ToCurriculumResponse(c)

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
		if resp.ProgramID != programID {
			t.Errorf("ProgramID = %v, want %v", resp.ProgramID, programID)
		}
		if resp.CohortYear != 2022 {
			t.Errorf("CohortYear = %d, want 2022", resp.CohortYear)
		}
		if resp.Stage != 2 {
			t.Errorf("Stage = %d, want 2", resp.Stage)
		}
		if resp.Semester != SemesterTypeFall {
			t.Errorf("Semester = %s, want %s", resp.Semester, SemesterTypeFall)
		}
		if resp.CourseID != courseID {
			t.Errorf("CourseID = %v, want %v", resp.CourseID, courseID)
		}
		if !resp.IsRequired {
			t.Error("IsRequired = false, want true")
		}
	})
}

func TestToCurriculumsResponse(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := ToCurriculumsResponse([]Curriculum{})
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("multiple curricula", func(t *testing.T) {
		now := time.Now()
		programID := uuid.New()
		curricula := []Curriculum{
			{ID: uuid.New(), ProgramID: programID, Stage: 1, Semester: SemesterTypeFall, IsRequired: true, CreatedAt: now},
			{ID: uuid.New(), ProgramID: programID, Stage: 1, Semester: SemesterTypeFall, IsRequired: false, CreatedAt: now},
		}

		result := ToCurriculumsResponse(curricula)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if !result[0].IsRequired {
			t.Error("result[0].IsRequired = false, want true")
		}
		if result[1].IsRequired {
			t.Error("result[1].IsRequired = true, want false")
		}
	})
}

func TestToRequirementResponse(t *testing.T) {
	t.Run("nil returns nil", func(t *testing.T) {
		if got := ToRequirementResponse(nil); got != nil {
			t.Errorf("ToRequirementResponse(nil) = %v, want nil", got)
		}
	})

	t.Run("valid requirement", func(t *testing.T) {
		id := uuid.New()
		programID := uuid.New()
		createdBy := uuid.New()
		now := time.Now()

		r := &SemesterRequirement{
			ID:         id,
			ProgramID:  programID,
			CohortYear: 2022,
			Stage:      3,
			Semester:   SemesterTypeSpring,
			MinCredits: 15,
			CreatedBy:  createdBy,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		resp := ToRequirementResponse(r)

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
		if resp.ProgramID != programID {
			t.Errorf("ProgramID = %v, want %v", resp.ProgramID, programID)
		}
		if resp.CohortYear != 2022 {
			t.Errorf("CohortYear = %d, want 2022", resp.CohortYear)
		}
		if resp.Stage != 3 {
			t.Errorf("Stage = %d, want 3", resp.Stage)
		}
		if resp.Semester != SemesterTypeSpring {
			t.Errorf("Semester = %s, want %s", resp.Semester, SemesterTypeSpring)
		}
		if resp.MinCredits != 15 {
			t.Errorf("MinCredits = %d, want 15", resp.MinCredits)
		}
	})
}

func TestToRequirementsResponse(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := ToRequirementsResponse([]SemesterRequirement{})
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("multiple requirements", func(t *testing.T) {
		now := time.Now()
		programID := uuid.New()
		requirements := []SemesterRequirement{
			{ID: uuid.New(), ProgramID: programID, Stage: 1, MinCredits: 15, CreatedAt: now, UpdatedAt: now},
			{ID: uuid.New(), ProgramID: programID, Stage: 2, MinCredits: 18, CreatedAt: now, UpdatedAt: now},
		}

		result := ToRequirementsResponse(requirements)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].MinCredits != 15 {
			t.Errorf("result[0].MinCredits = %d, want 15", result[0].MinCredits)
		}
		if result[1].MinCredits != 18 {
			t.Errorf("result[1].MinCredits = %d, want 18", result[1].MinCredits)
		}
	})
}
