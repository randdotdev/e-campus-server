package management

import (
	"testing"
	"time"
)

func TestApplicationStateRules(t *testing.T) {
	tests := []struct {
		status      ApplicationStatus
		canUpdate   bool
		canWithdraw bool
		canReview   bool
	}{
		{ApplicationPending, false, true, true},
		{ApplicationNeedsRevision, true, true, false},
		{ApplicationApproved, false, false, false},
		{ApplicationRejected, false, false, false},
		{ApplicationWithdrawn, false, false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := CanUpdateApplication(tt.status); got != tt.canUpdate {
				t.Errorf("CanUpdateApplication = %v, want %v", got, tt.canUpdate)
			}
			if got := CanWithdrawApplication(tt.status); got != tt.canWithdraw {
				t.Errorf("CanWithdrawApplication = %v, want %v", got, tt.canWithdraw)
			}
			if got := CanReviewApplication(tt.status); got != tt.canReview {
				t.Errorf("CanReviewApplication = %v, want %v", got, tt.canReview)
			}
		})
	}
}

func TestValidReviewStatus(t *testing.T) {
	for _, valid := range []ApplicationStatus{ApplicationApproved, ApplicationRejected, ApplicationNeedsRevision} {
		if !ValidReviewStatus(valid) {
			t.Errorf("ValidReviewStatus(%q) = false, want true", valid)
		}
	}
	for _, invalid := range []ApplicationStatus{ApplicationPending, ApplicationWithdrawn, ""} {
		if ValidReviewStatus(invalid) {
			t.Errorf("ValidReviewStatus(%q) = true, want false", invalid)
		}
	}
}

func TestApplicantAge(t *testing.T) {
	on := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		dob     string
		want    int
		wantErr bool
	}{
		{"birthday passed this year", "2000-06-30", 26, false},
		{"birthday today", "2000-07-01", 26, false},
		{"birthday later this year", "2000-07-02", 25, false},
		{"malformed date", "01/02/2000", 0, true},
		{"empty", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplicantAge(tt.dob, on)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ApplicantAge(%q) = %d, want %d", tt.dob, got, tt.want)
			}
		})
	}
}
