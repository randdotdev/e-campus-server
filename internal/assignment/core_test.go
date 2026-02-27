package assignment

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
		now       time.Time
		want      bool
	}{
		{"nil publishAt", nil, now, false},
		{"past publishAt", &past, now, true},
		{"future publishAt", &future, now, false},
		{"exact time", &now, now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPublished(tt.publishAt, tt.now); got != tt.want {
				t.Errorf("IsPublished() = %v, want %v", got, tt.want)
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
		now       time.Time
		want      bool
	}{
		{"before deadline", future, false, now, true},
		{"after deadline no late", past, false, now, false},
		{"after deadline allow late", past, true, now, true},
		{"exact deadline", now, false, now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanSubmit(tt.deadline, tt.allowLate, tt.now); got != tt.want {
				t.Errorf("CanSubmit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLate(t *testing.T) {
	deadline := time.Date(2024, 1, 15, 23, 59, 0, 0, time.UTC)

	tests := []struct {
		name        string
		submittedAt time.Time
		want        bool
	}{
		{"on time", deadline.Add(-time.Hour), false},
		{"exact deadline", deadline, false},
		{"late", deadline.Add(time.Hour), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLate(deadline, tt.submittedAt); got != tt.want {
				t.Errorf("IsLate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLateness(t *testing.T) {
	deadline := time.Date(2024, 1, 15, 23, 59, 0, 0, time.UTC)

	tests := []struct {
		name        string
		submittedAt time.Time
		want        time.Duration
	}{
		{"on time", deadline.Add(-time.Hour), 0},
		{"1 hour late", deadline.Add(time.Hour), time.Hour},
		{"2 hours late", deadline.Add(2 * time.Hour), 2 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Lateness(deadline, tt.submittedAt); got != tt.want {
				t.Errorf("Lateness() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanStudentModify(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	graded := now.Add(-2 * time.Hour)

	tests := []struct {
		name      string
		deadline  time.Time
		allowLate bool
		gradedAt  *time.Time
		now       time.Time
		want      bool
	}{
		{"before deadline not graded", future, false, nil, now, true},
		{"before deadline graded", future, false, &graded, now, false},
		{"after deadline no late", past, false, nil, now, false},
		{"after deadline allow late", past, true, nil, now, true},
		{"after deadline allow late but graded", past, true, &graded, now, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanStudentModify(tt.deadline, tt.allowLate, tt.gradedAt, tt.now); got != tt.want {
				t.Errorf("CanStudentModify() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDraft(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		submittedAt *time.Time
		want        bool
	}{
		{"nil is draft", nil, true},
		{"with time is not draft", &now, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDraft(tt.submittedAt); got != tt.want {
				t.Errorf("IsDraft() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		submittedAt *time.Time
		gradedAt    *time.Time
		want        string
	}{
		{"draft", nil, nil, StatusDraft},
		{"submitted", &now, nil, StatusSubmitted},
		{"graded", &now, &now, StatusGraded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeStatus(tt.submittedAt, tt.gradedAt); got != tt.want {
				t.Errorf("ComputeStatus() = %v, want %v", got, tt.want)
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
		{"zero", 0, 100, true},
		{"max", 100, 100, true},
		{"middle", 50, 100, true},
		{"negative", -1, 100, false},
		{"over max", 101, 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidScore(tt.score, tt.maxScore); got != tt.want {
				t.Errorf("IsValidScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanSeeScore(t *testing.T) {
	tests := []struct {
		name         string
		scoresPublic bool
		want         bool
	}{
		{"public", true, true},
		{"private", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanSeeScore(tt.scoresPublic); got != tt.want {
				t.Errorf("CanSeeScore() = %v, want %v", got, tt.want)
			}
		})
	}
}
