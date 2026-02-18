package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToApplicationResponse(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	programID := uuid.New()
	reviewerID := uuid.New()
	reviewedAt := time.Now().Add(-time.Hour)
	reviewNotes := "Looks good"
	now := time.Now()

	personalExtra := json.RawMessage(`{"phone":"1234567890"}`)
	academic := json.RawMessage(`{"gpa":3.8}`)
	documents := json.RawMessage(`[{"type":"id","url":"https://example.com/id.pdf"}]`)

	app := &Application{
		ID:            id,
		UserID:        &userID,
		ProgramID:     programID,
		AdmissionYear: 2024,
		StudyType:     StudyTypeMorning,
		DateOfBirth:   "2000-01-15",
		Gender:        "male",
		Nationality:   "Iraq",
		PersonalExtra: personalExtra,
		Academic:      academic,
		Documents:     documents,
		Status:        StatusApproved,
		ReviewedBy:    &reviewerID,
		ReviewedAt:    &reviewedAt,
		ReviewNotes:   &reviewNotes,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	resp := ToApplicationResponse(app)

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.UserID == nil || *resp.UserID != userID {
		t.Errorf("UserID = %v, want %v", resp.UserID, userID)
	}
	if resp.ProgramID != programID {
		t.Errorf("ProgramID = %v, want %v", resp.ProgramID, programID)
	}
	if resp.AdmissionYear != 2024 {
		t.Errorf("AdmissionYear = %v, want 2024", resp.AdmissionYear)
	}
	if resp.StudyType != StudyTypeMorning {
		t.Errorf("StudyType = %v, want %v", resp.StudyType, StudyTypeMorning)
	}
	if resp.DateOfBirth != "2000-01-15" {
		t.Errorf("DateOfBirth = %v, want 2000-01-15", resp.DateOfBirth)
	}
	if resp.Gender != "male" {
		t.Errorf("Gender = %v, want male", resp.Gender)
	}
	if resp.Nationality != "Iraq" {
		t.Errorf("Nationality = %v, want Iraq", resp.Nationality)
	}
	if resp.Status != StatusApproved {
		t.Errorf("Status = %v, want %v", resp.Status, StatusApproved)
	}
	if resp.ReviewedBy == nil || *resp.ReviewedBy != reviewerID {
		t.Errorf("ReviewedBy = %v, want %v", resp.ReviewedBy, reviewerID)
	}
	if resp.ReviewNotes == nil || *resp.ReviewNotes != reviewNotes {
		t.Errorf("ReviewNotes = %v, want %v", resp.ReviewNotes, reviewNotes)
	}

	// Check JSONB fields were unmarshaled correctly
	if resp.PersonalExtra["phone"] != "1234567890" {
		t.Errorf("PersonalExtra[phone] = %v, want 1234567890", resp.PersonalExtra["phone"])
	}
	if resp.Academic["gpa"] != 3.8 {
		t.Errorf("Academic[gpa] = %v, want 3.8", resp.Academic["gpa"])
	}
	if len(resp.Documents) != 1 {
		t.Errorf("len(Documents) = %v, want 1", len(resp.Documents))
	}
}

func TestToApplicationResponse_EmptyJSONB(t *testing.T) {
	app := &Application{
		ID:            uuid.New(),
		ProgramID:     uuid.New(),
		AdmissionYear: 2024,
		StudyType:     StudyTypeMorning,
		DateOfBirth:   "2000-01-15",
		Gender:        "female",
		Nationality:   "Iraq",
		PersonalExtra: nil,
		Academic:      nil,
		Documents:     nil,
		Status:        StatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	resp := ToApplicationResponse(app)

	if resp.PersonalExtra == nil {
		t.Error("PersonalExtra should not be nil")
	}
	if len(resp.PersonalExtra) != 0 {
		t.Errorf("len(PersonalExtra) = %v, want 0", len(resp.PersonalExtra))
	}
	if resp.Academic == nil {
		t.Error("Academic should not be nil")
	}
	if len(resp.Academic) != 0 {
		t.Errorf("len(Academic) = %v, want 0", len(resp.Academic))
	}
	if resp.Documents == nil {
		t.Error("Documents should not be nil")
	}
	if len(resp.Documents) != 0 {
		t.Errorf("len(Documents) = %v, want 0", len(resp.Documents))
	}
}

func TestToApplicationsResponse(t *testing.T) {
	apps := []Application{
		{
			ID:            uuid.New(),
			ProgramID:     uuid.New(),
			AdmissionYear: 2024,
			StudyType:     StudyTypeMorning,
			DateOfBirth:   "2000-01-15",
			Gender:        "male",
			Nationality:   "Iraq",
			Status:        StatusPending,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            uuid.New(),
			ProgramID:     uuid.New(),
			AdmissionYear: 2025,
			StudyType:     StudyTypeEvening,
			DateOfBirth:   "1999-05-20",
			Gender:        "female",
			Nationality:   "Turkey",
			Status:        StatusApproved,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	resp := ToApplicationsResponse(apps)

	if len(resp) != 2 {
		t.Errorf("len(resp) = %v, want 2", len(resp))
	}
	if resp[0].ID != apps[0].ID {
		t.Errorf("resp[0].ID = %v, want %v", resp[0].ID, apps[0].ID)
	}
	if resp[1].ID != apps[1].ID {
		t.Errorf("resp[1].ID = %v, want %v", resp[1].ID, apps[1].ID)
	}
}

func TestToApplicationsResponse_Empty(t *testing.T) {
	resp := ToApplicationsResponse([]Application{})

	if resp == nil {
		t.Error("resp should not be nil")
	}
	if len(resp) != 0 {
		t.Errorf("len(resp) = %v, want 0", len(resp))
	}
}
