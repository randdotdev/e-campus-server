package project

import (
	"testing"
	"time"
)

func TestIsPublished(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		publishAt *time.Time
		want      bool
	}{
		{"nil publish at", nil, true},
		{"past publish at", &past, true},
		{"future publish at", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPublished(tt.publishAt, now); got != tt.want {
				t.Errorf("IsPublished() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRegistrationClosed(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name     string
		deadline *time.Time
		want     bool
	}{
		{"nil deadline", nil, false},
		{"past deadline", &past, true},
		{"future deadline", &future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRegistrationClosed(tt.deadline, now); got != tt.want {
				t.Errorf("IsRegistrationClosed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDeadlinePassed(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name     string
		deadline time.Time
		want     bool
	}{
		{"past deadline", past, true},
		{"future deadline", future, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDeadlinePassed(tt.deadline, now); got != tt.want {
				t.Errorf("IsDeadlinePassed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanSubmit(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name      string
		deadline  time.Time
		allowLate bool
		want      bool
	}{
		{"before deadline", future, false, true},
		{"after deadline no late", past, false, false},
		{"after deadline allow late", past, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanSubmit(tt.deadline, tt.allowLate, now); got != tt.want {
				t.Errorf("CanSubmit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLateSubmission(t *testing.T) {
	deadline := time.Now()
	before := deadline.Add(-time.Hour)
	after := deadline.Add(time.Hour)

	tests := []struct {
		name        string
		submittedAt time.Time
		want        bool
	}{
		{"before deadline", before, false},
		{"after deadline", after, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLateSubmission(deadline, tt.submittedAt); got != tt.want {
				t.Errorf("IsLateSubmission() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasContent(t *testing.T) {
	content := "some content"
	empty := ""

	tests := []struct {
		name    string
		content *string
		files   []SubmissionFile
		want    bool
	}{
		{"has content", &content, nil, true},
		{"has files", nil, []SubmissionFile{{}}, true},
		{"empty content with files", &empty, []SubmissionFile{{}}, true},
		{"nil content no files", nil, nil, false},
		{"empty content no files", &empty, nil, false},
		{"empty content empty files", &empty, []SubmissionFile{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasContent(tt.content, tt.files); got != tt.want {
				t.Errorf("HasContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidScore(t *testing.T) {
	tests := []struct {
		name     string
		score    float64
		maxScore float64
		want     bool
	}{
		{"zero valid", 0, 100, true},
		{"max valid", 100, 100, true},
		{"middle valid", 50, 100, true},
		{"negative invalid", -1, 100, false},
		{"over max invalid", 101, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidScore(tt.score, tt.maxScore); got != tt.want {
				t.Errorf("IsValidScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidVisibility(t *testing.T) {
	tests := []struct {
		name       string
		visibility string
		want       bool
	}{
		{"hidden valid", VisibilityHidden, true},
		{"registered valid", VisibilityRegistered, true},
		{"all valid", VisibilityAll, true},
		{"invalid", "invalid", false},
		{"empty invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidVisibility(tt.visibility); got != tt.want {
				t.Errorf("IsValidVisibility() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidMemberRange(t *testing.T) {
	tests := []struct {
		name string
		min  int
		max  int
		want bool
	}{
		{"valid range", 2, 5, true},
		{"equal valid", 3, 3, true},
		{"min 1 valid", 1, 10, true},
		{"min 0 invalid", 0, 5, false},
		{"max less than min invalid", 5, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMemberRange(tt.min, tt.max); got != tt.want {
				t.Errorf("IsValidMemberRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidMergeTarget(t *testing.T) {
	three := 3
	one := 1
	ten := 10

	tests := []struct {
		name   string
		target *int
		min    int
		max    int
		want   bool
	}{
		{"nil valid", nil, 2, 5, true},
		{"in range valid", &three, 2, 5, true},
		{"equals min valid", &three, 3, 5, true},
		{"equals max valid", &three, 2, 3, true},
		{"below min invalid", &one, 2, 5, false},
		{"above max invalid", &ten, 2, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidMergeTarget(tt.target, tt.min, tt.max); got != tt.want {
				t.Errorf("IsValidMergeTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanViewRegistrations(t *testing.T) {
	tests := []struct {
		name         string
		visibility   string
		isRegistered bool
		isTeacher    bool
		want         bool
	}{
		{"teacher can view hidden", VisibilityHidden, false, true, true},
		{"teacher can view all", VisibilityAll, false, true, true},
		{"registered can view all", VisibilityAll, true, false, true},
		{"unregistered can view all", VisibilityAll, false, false, true},
		{"registered can view registered", VisibilityRegistered, true, false, true},
		{"unregistered cannot view registered", VisibilityRegistered, false, false, false},
		{"registered cannot view hidden", VisibilityHidden, true, false, false},
		{"unregistered cannot view hidden", VisibilityHidden, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanViewRegistrations(tt.visibility, tt.isRegistered, tt.isTeacher); got != tt.want {
				t.Errorf("CanViewRegistrations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldMergeTeam(t *testing.T) {
	tests := []struct {
		name        string
		memberCount int
		minMembers  int
		want        bool
	}{
		{"below min", 2, 3, true},
		{"equals min", 3, 3, false},
		{"above min", 4, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldMergeTeam(tt.memberCount, tt.minMembers); got != tt.want {
				t.Errorf("ShouldMergeTeam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyProjectUpdates(t *testing.T) {
	now := time.Now()
	newTitle := "New Title"
	newBody := "New Body"
	newDeadline := now.Add(time.Hour * 24)
	newMaxScore := 150.0
	newMinMembers := 3
	newMaxMembers := 8
	newMergeTarget := 5
	newRegDeadline := now.Add(time.Hour * 12)
	newVisibility := VisibilityAll
	newAllowLate := true
	newPublishAt := now.Add(-time.Hour)

	p := &Project{
		Title:      "Old Title",
		MaxScore:   100,
		MinMembers: 2,
		MaxMembers: 5,
		Visibility: VisibilityHidden,
		AllowLate:  false,
		Deadline:   now,
	}

	updates := ProjectUpdates{
		Title:                &newTitle,
		Body:                 &newBody,
		Deadline:             &newDeadline,
		MaxScore:             &newMaxScore,
		MinMembers:           &newMinMembers,
		MaxMembers:           &newMaxMembers,
		MergeTarget:          &newMergeTarget,
		RegistrationDeadline: &newRegDeadline,
		Visibility:           &newVisibility,
		AllowLate:            &newAllowLate,
		PublishAt:            &newPublishAt,
	}

	ApplyProjectUpdates(p, updates)

	if p.Title != newTitle {
		t.Errorf("Title = %v, want %v", p.Title, newTitle)
	}
	if p.Body == nil || *p.Body != newBody {
		t.Errorf("Body = %v, want %v", p.Body, newBody)
	}
	if !p.Deadline.Equal(newDeadline) {
		t.Errorf("Deadline = %v, want %v", p.Deadline, newDeadline)
	}
	if p.MaxScore != newMaxScore {
		t.Errorf("MaxScore = %v, want %v", p.MaxScore, newMaxScore)
	}
	if p.MinMembers != newMinMembers {
		t.Errorf("MinMembers = %v, want %v", p.MinMembers, newMinMembers)
	}
	if p.MaxMembers != newMaxMembers {
		t.Errorf("MaxMembers = %v, want %v", p.MaxMembers, newMaxMembers)
	}
	if p.MergeTarget == nil || *p.MergeTarget != newMergeTarget {
		t.Errorf("MergeTarget = %v, want %v", p.MergeTarget, newMergeTarget)
	}
	if p.Visibility != newVisibility {
		t.Errorf("Visibility = %v, want %v", p.Visibility, newVisibility)
	}
	if p.AllowLate != newAllowLate {
		t.Errorf("AllowLate = %v, want %v", p.AllowLate, newAllowLate)
	}
}
