package attendance

import (
	"slices"

	"github.com/google/uuid"
)

func IsValidPercentage(p int) bool {
	return slices.Contains(ValidPercentages, p)
}

func IsValidExcuseDecision(status string) bool {
	return status == ExcuseStatusApproved || status == ExcuseStatusRejected
}

func ComputeStatus(markedBy *uuid.UUID, percentage int, excuseStatus *string) string {
	if markedBy == nil {
		return StatusUnmarked
	}
	if excuseStatus != nil && *excuseStatus == ExcuseStatusApproved {
		return StatusExcused
	}
	if percentage > 0 {
		return StatusAttended
	}
	return StatusAbsent
}

type AttendanceInput struct {
	DurationHours float64
	Percentage    int
	IsExcused     bool
}

func CalculateAttendanceRate(records []AttendanceInput) float64 {
	var totalHours, attendedHours float64
	for _, r := range records {
		if r.IsExcused {
			continue
		}
		totalHours += r.DurationHours
		attendedHours += r.DurationHours * float64(r.Percentage) / 100
	}
	if totalHours == 0 {
		return 100
	}
	return attendedHours / totalHours * 100
}

func CalculateSummary(summary *AttendanceSummary) {
	effectiveTotal := summary.TotalHours - summary.ExcusedHours
	if effectiveTotal <= 0 {
		summary.AttendanceRate = 100
		return
	}
	summary.AttendanceRate = (summary.AttendedHours / effectiveTotal) * 100
}

func CalculateCourseAttendanceRate(total, attended, excused int) float64 {
	effective := total - excused
	if effective <= 0 {
		return 100
	}
	return float64(attended) / float64(effective) * 100
}
