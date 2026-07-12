package classroom_test

import (
	"testing"
	"time"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

func TestRegistrationOpen(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	if !classroom.RegistrationOpen(nil, now) {
		t.Error("no deadline keeps registration open")
	}
	if !classroom.RegistrationOpen(&future, now) {
		t.Error("future deadline keeps registration open")
	}
	if classroom.RegistrationOpen(&past, now) {
		t.Error("past deadline closes registration")
	}
}

func TestTeamSizeFits(t *testing.T) {
	if !classroom.TeamSizeFits(3, 2, 5) || !classroom.TeamSizeFits(2, 2, 5) || !classroom.TeamSizeFits(5, 2, 5) {
		t.Error("bounds are inclusive")
	}
	if classroom.TeamSizeFits(1, 2, 5) || classroom.TeamSizeFits(6, 2, 5) {
		t.Error("outside the bounds refuses")
	}
}

func TestCanViewRegistrations(t *testing.T) {
	tests := []struct {
		name         string
		v            classroom.ProjectVisibility
		isRegistered bool
		isStaff      bool
		want         bool
	}{
		{"hidden: staff only", classroom.ProjectHidden, true, false, false},
		{"hidden: staff sees", classroom.ProjectHidden, false, true, true},
		{"registered: member sees", classroom.ProjectRegistered, true, false, true},
		{"registered: outsider blind", classroom.ProjectRegistered, false, false, false},
		{"all: everyone", classroom.ProjectAll, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classroom.CanViewRegistrations(tt.v, tt.isRegistered, tt.isStaff); got != tt.want {
				t.Errorf("CanViewRegistrations = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQARules(t *testing.T) {
	if !classroom.ValidQAText("a title", "a body") {
		t.Error("plain text is valid")
	}
	if classroom.ValidQAText("  ", "body") || classroom.ValidQAText("title", "") {
		t.Error("blank title or body refuses")
	}

	if got := classroom.DefaultTeamName("Alan"); got != "Alan's Team" {
		t.Errorf("DefaultTeamName = %q", got)
	}
}

func TestSampleQuestions(t *testing.T) {
	easy := classroom.Difficulty("easy")
	pool := []classroom.Question{
		{Difficulty: &easy}, {Difficulty: &easy}, {Difficulty: &easy},
	}
	got, warnings := classroom.SampleQuestions(pool, classroom.SampleCounts{Easy: 2, Hard: 1})
	if len(got) != 2 {
		t.Errorf("sampled %d, want 2 easy", len(got))
	}
	if len(warnings) != 1 {
		t.Errorf("want one warning for the empty hard tier, got %v", warnings)
	}
}
