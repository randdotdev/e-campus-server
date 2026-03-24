package project

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToProjectResponse(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	body := "Project body"

	tests := []struct {
		name             string
		project          *Project
		wantPublished    bool
		wantRegOpen      bool
		wantDeadlinePast bool
	}{
		{
			name: "published open not past",
			project: &Project{
				ID:                   uuid.New(),
				OfferingID:           uuid.New(),
				Title:                "Test",
				Body:                 &body,
				Deadline:             future,
				MaxScore:             100,
				MinMembers:           2,
				MaxMembers:           5,
				RegistrationDeadline: &future,
				Visibility:           VisibilityAll,
				PublishAt:            &past,
				CreatedAt:            now,
			},
			wantPublished:    true,
			wantRegOpen:      true,
			wantDeadlinePast: false,
		},
		{
			name: "not published",
			project: &Project{
				ID:         uuid.New(),
				OfferingID: uuid.New(),
				Title:      "Test",
				Deadline:   future,
				MaxScore:   100,
				MinMembers: 2,
				MaxMembers: 5,
				Visibility: VisibilityHidden,
				PublishAt:  &future,
				CreatedAt:  now,
			},
			wantPublished:    false,
			wantRegOpen:      true,
			wantDeadlinePast: false,
		},
		{
			name: "registration closed",
			project: &Project{
				ID:                   uuid.New(),
				OfferingID:           uuid.New(),
				Title:                "Test",
				Deadline:             future,
				MaxScore:             100,
				MinMembers:           2,
				MaxMembers:           5,
				RegistrationDeadline: &past,
				Visibility:           VisibilityAll,
				CreatedAt:            now,
			},
			wantPublished:    true,
			wantRegOpen:      false,
			wantDeadlinePast: false,
		},
		{
			name: "deadline passed",
			project: &Project{
				ID:         uuid.New(),
				OfferingID: uuid.New(),
				Title:      "Test",
				Deadline:   past,
				MaxScore:   100,
				MinMembers: 2,
				MaxMembers: 5,
				Visibility: VisibilityAll,
				CreatedAt:  now,
			},
			wantPublished:    true,
			wantRegOpen:      true,
			wantDeadlinePast: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToProjectResponse(tt.project, now)

			if got.ID != tt.project.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.project.ID)
			}
			if got.Title != tt.project.Title {
				t.Errorf("Title = %v, want %v", got.Title, tt.project.Title)
			}
			if got.IsPublished != tt.wantPublished {
				t.Errorf("IsPublished = %v, want %v", got.IsPublished, tt.wantPublished)
			}
			if got.IsRegistrationOpen != tt.wantRegOpen {
				t.Errorf("IsRegistrationOpen = %v, want %v", got.IsRegistrationOpen, tt.wantRegOpen)
			}
			if got.IsDeadlinePassed != tt.wantDeadlinePast {
				t.Errorf("IsDeadlinePassed = %v, want %v", got.IsDeadlinePassed, tt.wantDeadlinePast)
			}
		})
	}
}

func TestToProjectsResponse(t *testing.T) {
	now := time.Now()
	projects := []Project{
		{ID: uuid.New(), Title: "Project 1", Deadline: now, MaxScore: 100, MinMembers: 2, MaxMembers: 5},
		{ID: uuid.New(), Title: "Project 2", Deadline: now, MaxScore: 100, MinMembers: 2, MaxMembers: 5},
	}

	got := ToProjectsResponse(projects, now)

	if len(got) != 2 {
		t.Fatalf("len = %v, want 2", len(got))
	}
	if got[0].Title != "Project 1" {
		t.Errorf("got[0].Title = %v, want Project 1", got[0].Title)
	}
	if got[1].Title != "Project 2" {
		t.Errorf("got[1].Title = %v, want Project 2", got[1].Title)
	}
}

func TestToSubmissionResponse(t *testing.T) {
	now := time.Now()
	deadline := now.Add(time.Hour)
	submittedAt := now

	t.Run("with files", func(t *testing.T) {
		s := &Submission{
			ID:             uuid.New(),
			ProjectID:      uuid.New(),
			ProjectGroupID: uuid.New(),
			SubmittedAt:    &submittedAt,
			CreatedAt:      now,
		}
		files := []SubmissionFile{
			{ID: uuid.New(), StoredFileID: uuid.New(), DisplayName: "file1.pdf", OrderIndex: 0},
			{ID: uuid.New(), StoredFileID: uuid.New(), DisplayName: "file2.pdf", OrderIndex: 1},
		}

		got := ToSubmissionResponse(s, files, deadline)

		if !got.IsSubmitted {
			t.Error("expected IsSubmitted to be true")
		}
		if got.IsLate {
			t.Error("expected IsLate to be false")
		}
		if len(got.Files) != 2 {
			t.Errorf("len(Files) = %v, want 2", len(got.Files))
		}
	})

	t.Run("late submission", func(t *testing.T) {
		lateDeadline := now.Add(-time.Hour)
		s := &Submission{
			ID:             uuid.New(),
			ProjectID:      uuid.New(),
			ProjectGroupID: uuid.New(),
			SubmittedAt:    &submittedAt,
			CreatedAt:      now,
		}

		got := ToSubmissionResponse(s, nil, lateDeadline)

		if !got.IsLate {
			t.Error("expected IsLate to be true")
		}
	})

	t.Run("not submitted", func(t *testing.T) {
		s := &Submission{
			ID:             uuid.New(),
			ProjectID:      uuid.New(),
			ProjectGroupID: uuid.New(),
			CreatedAt:      now,
		}

		got := ToSubmissionResponse(s, nil, deadline)

		if got.IsSubmitted {
			t.Error("expected IsSubmitted to be false")
		}
		if got.IsLate {
			t.Error("expected IsLate to be false for non-submitted")
		}
	})
}

func TestToProjectGroupResponse(t *testing.T) {
	now := time.Now()
	name := "Group A"
	title := "Our Project"
	teamID := uuid.New()

	g := &ProjectGroupWithMembers{
		ProjectGroup: ProjectGroup{
			ID:           uuid.New(),
			ProjectID:    uuid.New(),
			Name:         &name,
			ProjectTitle: &title,
			LeaderID:     uuid.New(),
			Finalized:    true,
			CreatedAt:    now,
		},
		Members: []GroupMemberInfo{
			{StudentID: uuid.New(), StudentName: "Ali", FromTeamID: &teamID},
			{StudentID: uuid.New(), StudentName: "Bob", FromTeamID: &teamID},
		},
		MemberCount: 2,
	}

	got := ToProjectGroupResponse(g)

	if *got.Name != "Group A" {
		t.Errorf("Name = %v, want Group A", *got.Name)
	}
	if !got.Finalized {
		t.Error("expected Finalized to be true")
	}
	if len(got.Members) != 2 {
		t.Errorf("len(Members) = %v, want 2", len(got.Members))
	}
	if got.Members[0].StudentName != "Ali" {
		t.Errorf("Members[0].StudentName = %v, want Ali", got.Members[0].StudentName)
	}
}

func TestToMyGradeResponse(t *testing.T) {
	score := 85.0
	feedback := "Good work"
	now := time.Now()

	t.Run("public scores", func(t *testing.T) {
		g := &Grade{
			Score:    &score,
			Feedback: &feedback,
			GradedAt: &now,
		}

		got := ToMyGradeResponse(g, true)

		if !got.IsPublic {
			t.Error("expected IsPublic to be true")
		}
		if got.Score == nil || *got.Score != 85.0 {
			t.Errorf("Score = %v, want 85.0", got.Score)
		}
		if got.Feedback == nil || *got.Feedback != "Good work" {
			t.Errorf("Feedback = %v, want Good work", got.Feedback)
		}
	})

	t.Run("private scores", func(t *testing.T) {
		g := &Grade{
			Score:    &score,
			Feedback: &feedback,
			GradedAt: &now,
		}

		got := ToMyGradeResponse(g, false)

		if got.IsPublic {
			t.Error("expected IsPublic to be false")
		}
		if got.Score != nil {
			t.Errorf("expected Score to be nil when not public, got %v", got.Score)
		}
		if got.Feedback != nil {
			t.Errorf("expected Feedback to be nil when not public, got %v", got.Feedback)
		}
	})

	t.Run("nil grade", func(t *testing.T) {
		got := ToMyGradeResponse(nil, true)

		if !got.IsPublic {
			t.Error("expected IsPublic to be true")
		}
		if got.Score != nil {
			t.Errorf("expected Score to be nil, got %v", got.Score)
		}
	})
}

func TestToFileInputs(t *testing.T) {
	files := []FileInputRequest{
		{StoredFileID: uuid.New(), DisplayName: "file1.pdf"},
		{StoredFileID: uuid.New(), DisplayName: "file2.pdf"},
	}

	got := ToFileInputs(files)

	if len(got) != 2 {
		t.Fatalf("len = %v, want 2", len(got))
	}
	if got[0].DisplayName != "file1.pdf" {
		t.Errorf("got[0].DisplayName = %v, want file1.pdf", got[0].DisplayName)
	}
	if got[1].DisplayName != "file2.pdf" {
		t.Errorf("got[1].DisplayName = %v, want file2.pdf", got[1].DisplayName)
	}
}
