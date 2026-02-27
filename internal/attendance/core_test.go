package attendance

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsValidPercentage(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected bool
	}{
		{"zero", 0, true},
		{"25", 25, true},
		{"50", 50, true},
		{"75", 75, true},
		{"100", 100, true},
		{"10 invalid", 10, false},
		{"99 invalid", 99, false},
		{"negative", -25, false},
		{"over 100", 125, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPercentage(tt.input); got != tt.expected {
				t.Errorf("IsValidPercentage(%d) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidExcuseDecision(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"approved", ExcuseStatusApproved, true},
		{"rejected", ExcuseStatusRejected, true},
		{"pending invalid", ExcuseStatusPending, false},
		{"empty invalid", "", false},
		{"random invalid", "maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidExcuseDecision(tt.input); got != tt.expected {
				t.Errorf("IsValidExcuseDecision(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestComputeStatus(t *testing.T) {
	markerID := uuid.New()
	approved := ExcuseStatusApproved
	pending := ExcuseStatusPending
	rejected := ExcuseStatusRejected

	tests := []struct {
		name         string
		markedBy     *uuid.UUID
		percentage   int
		excuseStatus *string
		expected     string
	}{
		{"unmarked no excuse", nil, 0, nil, StatusUnmarked},
		{"unmarked with pending", nil, 0, &pending, StatusUnmarked},
		{"absent no excuse", &markerID, 0, nil, StatusAbsent},
		{"absent with pending", &markerID, 0, &pending, StatusAbsent},
		{"absent with rejected", &markerID, 0, &rejected, StatusAbsent},
		{"absent with approved", &markerID, 0, &approved, StatusExcused},
		{"attended 50", &markerID, 50, nil, StatusAttended},
		{"attended 100", &markerID, 100, nil, StatusAttended},
		{"attended with approved", &markerID, 100, &approved, StatusExcused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeStatus(tt.markedBy, tt.percentage, tt.excuseStatus); got != tt.expected {
				t.Errorf("ComputeStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateAttendanceRate(t *testing.T) {
	tests := []struct {
		name     string
		records  []AttendanceInput
		expected float64
	}{
		{
			"empty records",
			nil,
			100,
		},
		{
			"all attended 100%",
			[]AttendanceInput{
				{DurationHours: 2, Percentage: 100, IsExcused: false},
				{DurationHours: 2, Percentage: 100, IsExcused: false},
			},
			100,
		},
		{
			"all absent",
			[]AttendanceInput{
				{DurationHours: 2, Percentage: 0, IsExcused: false},
				{DurationHours: 2, Percentage: 0, IsExcused: false},
			},
			0,
		},
		{
			"mixed attendance",
			[]AttendanceInput{
				{DurationHours: 2, Percentage: 100, IsExcused: false},
				{DurationHours: 2, Percentage: 50, IsExcused: false},
			},
			75,
		},
		{
			"with excused",
			[]AttendanceInput{
				{DurationHours: 2, Percentage: 100, IsExcused: false},
				{DurationHours: 2, Percentage: 0, IsExcused: true},
			},
			100,
		},
		{
			"all excused",
			[]AttendanceInput{
				{DurationHours: 2, Percentage: 0, IsExcused: true},
				{DurationHours: 2, Percentage: 0, IsExcused: true},
			},
			100,
		},
		{
			"partial attendance with excused",
			[]AttendanceInput{
				{DurationHours: 2, Percentage: 50, IsExcused: false},
				{DurationHours: 2, Percentage: 0, IsExcused: true},
			},
			50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateAttendanceRate(tt.records); got != tt.expected {
				t.Errorf("CalculateAttendanceRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateCourseAttendanceRate(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		attended int
		excused  int
		expected float64
	}{
		{"100% attendance", 10, 10, 0, 100},
		{"50% attendance", 10, 5, 0, 50},
		{"with excused", 10, 8, 2, 100},
		{"all excused", 10, 0, 10, 100},
		{"zero total", 0, 0, 0, 100},
		{"partial with excused", 10, 4, 2, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculateCourseAttendanceRate(tt.total, tt.attended, tt.excused); got != tt.expected {
				t.Errorf("CalculateCourseAttendanceRate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateSummary(t *testing.T) {
	tests := []struct {
		name     string
		summary  AttendanceSummary
		expected float64
	}{
		{
			"100% attendance",
			AttendanceSummary{TotalHours: 10, AttendedHours: 10, ExcusedHours: 0},
			100,
		},
		{
			"50% attendance",
			AttendanceSummary{TotalHours: 10, AttendedHours: 5, ExcusedHours: 0},
			50,
		},
		{
			"with excused hours",
			AttendanceSummary{TotalHours: 10, AttendedHours: 8, ExcusedHours: 2},
			100,
		},
		{
			"all excused",
			AttendanceSummary{TotalHours: 10, AttendedHours: 0, ExcusedHours: 10},
			100,
		},
		{
			"zero total",
			AttendanceSummary{TotalHours: 0, AttendedHours: 0, ExcusedHours: 0},
			100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CalculateSummary(&tt.summary)
			if tt.summary.AttendanceRate != tt.expected {
				t.Errorf("CalculateSummary() rate = %v, want %v", tt.summary.AttendanceRate, tt.expected)
			}
		})
	}
}
