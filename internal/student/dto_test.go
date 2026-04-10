package student

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToStudentResponse(t *testing.T) {
	t.Run("nil student returns nil", func(t *testing.T) {
		if got := ToStudentResponse(nil); got != nil {
			t.Errorf("ToStudentResponse(nil) = %v, want nil", got)
		}
	})

	t.Run("valid student", func(t *testing.T) {
		id := uuid.New()
		userID := uuid.New()
		programID := uuid.New()
		now := time.Now()

		s := &StudentSummary{
			ID:                id,
			UserID:            userID,
			ProgramID:         programID,
			AdmissionYear:     2022,
			CurrentCohortYear: 2022,
			CurrentYear:       2,
			Shift:             ShiftDay,
			Tuition:           TuitionFree,
			Status:            StatusActive,
			EnrolledAt:        now,
			CreatedAt:         now,
		}

		resp := ToStudentResponse(s)

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
		if resp.UserID != userID {
			t.Errorf("UserID = %v, want %v", resp.UserID, userID)
		}
		if resp.ProgramID != programID {
			t.Errorf("ProgramID = %v, want %v", resp.ProgramID, programID)
		}
		if resp.AdmissionYear != 2022 {
			t.Errorf("AdmissionYear = %d, want 2022", resp.AdmissionYear)
		}
		if resp.CurrentCohortYear != 2022 {
			t.Errorf("CurrentCohortYear = %d, want 2022", resp.CurrentCohortYear)
		}
		if resp.CurrentYear != 2 {
			t.Errorf("CurrentYear = %d, want 2", resp.CurrentYear)
		}
		if resp.Shift != ShiftDay {
			t.Errorf("Shift = %s, want %s", resp.Shift, ShiftDay)
		}
		if resp.Tuition != TuitionFree {
			t.Errorf("Tuition = %s, want %s", resp.Tuition, TuitionFree)
		}
		if resp.Status != StatusActive {
			t.Errorf("Status = %s, want %s", resp.Status, StatusActive)
		}
		if resp.EnrolledAt != now.Format(time.RFC3339) {
			t.Errorf("EnrolledAt = %s, want %s", resp.EnrolledAt, now.Format(time.RFC3339))
		}
	})
}

func TestToStudentsResponse(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := ToStudentsResponse([]StudentSummary{})
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("multiple students", func(t *testing.T) {
		now := time.Now()
		students := []StudentSummary{
			{ID: uuid.New(), Status: StatusActive, EnrolledAt: now, CreatedAt: now},
			{ID: uuid.New(), Status: StatusGraduated, EnrolledAt: now, CreatedAt: now},
		}

		result := ToStudentsResponse(students)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].Status != StatusActive {
			t.Errorf("result[0].Status = %s, want %s", result[0].Status, StatusActive)
		}
		if result[1].Status != StatusGraduated {
			t.Errorf("result[1].Status = %s, want %s", result[1].Status, StatusGraduated)
		}
	})
}

func TestToLeaveResponse(t *testing.T) {
	t.Run("nil leave returns nil", func(t *testing.T) {
		if got := ToLeaveResponse(nil, nil); got != nil {
			t.Errorf("ToLeaveResponse(nil, nil) = %v, want nil", got)
		}
	})

	t.Run("leave without optional fields", func(t *testing.T) {
		id := uuid.New()
		studentID := uuid.New()
		now := time.Now()

		l := &Leave{
			ID:        id,
			StudentID: studentID,
			Type:      LeaveTypeShort,
			Reason:    "Personal reasons",
			CreatedAt: now,
		}

		resp := ToLeaveResponse(l, nil)

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
		if resp.StudentID != studentID {
			t.Errorf("StudentID = %v, want %v", resp.StudentID, studentID)
		}
		if resp.Type != LeaveTypeShort {
			t.Errorf("Type = %s, want %s", resp.Type, LeaveTypeShort)
		}
		if resp.Reason != "Personal reasons" {
			t.Errorf("Reason = %s, want Personal reasons", resp.Reason)
		}
		if resp.StartDate != nil {
			t.Errorf("StartDate = %v, want nil", resp.StartDate)
		}
		if resp.EndDate != nil {
			t.Errorf("EndDate = %v, want nil", resp.EndDate)
		}
		if resp.ApprovedBy != nil {
			t.Errorf("ApprovedBy = %v, want nil", resp.ApprovedBy)
		}
		if resp.ApprovedAt != nil {
			t.Errorf("ApprovedAt = %v, want nil", resp.ApprovedAt)
		}
	})

	t.Run("leave with all optional fields", func(t *testing.T) {
		id := uuid.New()
		studentID := uuid.New()
		academicYearID := uuid.New()
		approverID := uuid.New()
		now := time.Now()
		startDate := now.AddDate(0, 0, 1)
		endDate := now.AddDate(0, 1, 0)
		notes := "Some notes"
		semesterIDs := []uuid.UUID{uuid.New(), uuid.New()}

		l := &Leave{
			ID:             id,
			StudentID:      studentID,
			Type:           LeaveTypeSemester,
			AcademicYearID: &academicYearID,
			Reason:         "Medical leave",
			StartDate:      &startDate,
			EndDate:        &endDate,
			ApprovedBy:     &approverID,
			ApprovedAt:     &now,
			Notes:          &notes,
			CreatedAt:      now,
		}

		resp := ToLeaveResponse(l, semesterIDs)

		if resp.AcademicYearID == nil || *resp.AcademicYearID != academicYearID {
			t.Errorf("AcademicYearID = %v, want %v", resp.AcademicYearID, academicYearID)
		}
		if len(resp.SemesterIDs) != 2 {
			t.Errorf("SemesterIDs len = %d, want 2", len(resp.SemesterIDs))
		}
		if resp.StartDate == nil || *resp.StartDate != startDate.Format("2006-01-02") {
			t.Errorf("StartDate = %v, want %s", resp.StartDate, startDate.Format("2006-01-02"))
		}
		if resp.EndDate == nil || *resp.EndDate != endDate.Format("2006-01-02") {
			t.Errorf("EndDate = %v, want %s", resp.EndDate, endDate.Format("2006-01-02"))
		}
		if resp.ApprovedBy == nil || *resp.ApprovedBy != approverID {
			t.Errorf("ApprovedBy = %v, want %v", resp.ApprovedBy, approverID)
		}
		if resp.ApprovedAt == nil || *resp.ApprovedAt != now.Format(time.RFC3339) {
			t.Errorf("ApprovedAt = %v, want %s", resp.ApprovedAt, now.Format(time.RFC3339))
		}
		if resp.Notes == nil || *resp.Notes != notes {
			t.Errorf("Notes = %v, want %s", resp.Notes, notes)
		}
	})
}

func TestToLeavesResponse(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := ToLeavesResponse([]Leave{})
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("multiple leaves", func(t *testing.T) {
		now := time.Now()
		leaves := []Leave{
			{ID: uuid.New(), Type: LeaveTypeShort, Reason: "Reason 1", CreatedAt: now},
			{ID: uuid.New(), Type: LeaveTypeSemester, Reason: "Reason 2", CreatedAt: now},
		}

		result := ToLeavesResponse(leaves)
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
		if result[0].Type != LeaveTypeShort {
			t.Errorf("result[0].Type = %s, want %s", result[0].Type, LeaveTypeShort)
		}
		if result[1].Type != LeaveTypeSemester {
			t.Errorf("result[1].Type = %s, want %s", result[1].Type, LeaveTypeSemester)
		}
	})
}

func TestToCohortHistoryResponse(t *testing.T) {
	t.Run("nil history returns nil", func(t *testing.T) {
		if got := ToCohortHistoryResponse(nil); got != nil {
			t.Errorf("ToCohortHistoryResponse(nil) = %v, want nil", got)
		}
	})

	t.Run("valid cohort history", func(t *testing.T) {
		id := uuid.New()
		studentID := uuid.New()
		now := time.Now()
		notes := "Failed due to poor grades"

		h := &CohortHistory{
			ID:             id,
			StudentID:      studentID,
			FromCohortYear: 2022,
			ToCohortYear:   2023,
			FromYear:       2,
			ToYear:         2,
			Reason:         CohortChangeReasonFailed,
			Notes:          &notes,
			ChangedAt:      now,
		}

		resp := ToCohortHistoryResponse(h)

		if resp.ID != id {
			t.Errorf("ID = %v, want %v", resp.ID, id)
		}
		if resp.StudentID != studentID {
			t.Errorf("StudentID = %v, want %v", resp.StudentID, studentID)
		}
		if resp.FromCohortYear != 2022 {
			t.Errorf("FromCohortYear = %d, want 2022", resp.FromCohortYear)
		}
		if resp.ToCohortYear != 2023 {
			t.Errorf("ToCohortYear = %d, want 2023", resp.ToCohortYear)
		}
		if resp.FromYear != 2 {
			t.Errorf("FromYear = %d, want 2", resp.FromYear)
		}
		if resp.ToYear != 2 {
			t.Errorf("ToYear = %d, want 2", resp.ToYear)
		}
		if resp.Reason != CohortChangeReasonFailed {
			t.Errorf("Reason = %s, want %s", resp.Reason, CohortChangeReasonFailed)
		}
		if resp.Notes == nil || *resp.Notes != notes {
			t.Errorf("Notes = %v, want %s", resp.Notes, notes)
		}
	})

	t.Run("cohort history without notes", func(t *testing.T) {
		now := time.Now()
		h := &CohortHistory{
			ID:             uuid.New(),
			StudentID:      uuid.New(),
			FromCohortYear: 2022,
			ToCohortYear:   2023,
			FromYear:       1,
			ToYear:         1,
			Reason:         CohortChangeReasonTransferred,
			Notes:          nil,
			ChangedAt:      now,
		}

		resp := ToCohortHistoryResponse(h)

		if resp.Notes != nil {
			t.Errorf("Notes = %v, want nil", resp.Notes)
		}
	})
}

func TestToCohortHistoriesResponse(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		result := ToCohortHistoriesResponse([]CohortHistory{})
		if len(result) != 0 {
			t.Errorf("len = %d, want 0", len(result))
		}
	})

	t.Run("multiple histories", func(t *testing.T) {
		now := time.Now()
		histories := []CohortHistory{
			{ID: uuid.New(), Reason: CohortChangeReasonFailed, ChangedAt: now},
			{ID: uuid.New(), Reason: CohortChangeReasonTransferred, ChangedAt: now},
			{ID: uuid.New(), Reason: CohortChangeReasonReturned, ChangedAt: now},
		}

		result := ToCohortHistoriesResponse(histories)
		if len(result) != 3 {
			t.Errorf("len = %d, want 3", len(result))
		}
		if result[0].Reason != CohortChangeReasonFailed {
			t.Errorf("result[0].Reason = %s, want %s", result[0].Reason, CohortChangeReasonFailed)
		}
		if result[1].Reason != CohortChangeReasonTransferred {
			t.Errorf("result[1].Reason = %s, want %s", result[1].Reason, CohortChangeReasonTransferred)
		}
		if result[2].Reason != CohortChangeReasonReturned {
			t.Errorf("result[2].Reason = %s, want %s", result[2].Reason, CohortChangeReasonReturned)
		}
	})
}
