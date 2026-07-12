package classroom_test

import (
	"testing"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

func TestValidPercentage(t *testing.T) {
	for _, valid := range []int{0, 25, 50, 75, 100} {
		if !classroom.ValidPercentage(valid) {
			t.Errorf("ValidPercentage(%d) = false, want true", valid)
		}
	}
	for _, invalid := range []int{-25, 10, 99, 125} {
		if classroom.ValidPercentage(invalid) {
			t.Errorf("ValidPercentage(%d) = true, want false", invalid)
		}
	}
}

func TestSummaryRate(t *testing.T) {
	tests := []struct {
		name    string
		summary classroom.AttendanceSummary
		want    float64
	}{
		{"full attendance", classroom.AttendanceSummary{TotalHours: 10, AttendedHours: 10}, 100},
		{"half attendance", classroom.AttendanceSummary{TotalHours: 10, AttendedHours: 5}, 50},
		{"excused hours leave the denominator", classroom.AttendanceSummary{TotalHours: 10, AttendedHours: 4, ExcusedHours: 2}, 50},
		{"everything excused", classroom.AttendanceSummary{TotalHours: 10, ExcusedHours: 10}, 100},
		{"nothing asked", classroom.AttendanceSummary{}, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classroom.SummaryRate(&tt.summary); got != tt.want {
				t.Errorf("SummaryRate = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidExcuseDecision(t *testing.T) {
	if !classroom.ValidExcuseDecision(classroom.ExcuseApproved) || !classroom.ValidExcuseDecision(classroom.ExcuseRejected) {
		t.Error("approved and rejected are the two decisions")
	}
	if classroom.ValidExcuseDecision(classroom.ExcusePending) {
		t.Error("pending is a state, not a decision")
	}
}
