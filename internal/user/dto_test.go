package user

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToUserResponse(t *testing.T) {
	now := time.Now()
	fullNameKU := "ناوی تەواو"

	user := &User{
		ID:         uuid.New(),
		Email:      "test@example.com",
		FullNameEN: "Test User",
		FullNameKU: &fullNameKU,
		IsVerified: true,
		IsActive:   true,
		CreatedAt:  now,
	}

	resp := ToUserResponse(user)

	if resp.ID != user.ID {
		t.Error("ID should match")
	}
	if resp.Email != user.Email {
		t.Error("Email should match")
	}
	if resp.FullNameEN != user.FullNameEN {
		t.Error("FullNameEN should match")
	}
	if resp.FullNameKU == nil || *resp.FullNameKU != fullNameKU {
		t.Error("FullNameKU should match")
	}
	if resp.IsVerified != user.IsVerified {
		t.Error("IsVerified should match")
	}
}

func TestToRoleResponse(t *testing.T) {
	role := &Role{ID: uuid.New(), Permission: "admin", ScopeType: "university"}

	resp := ToRoleResponse(role)

	if resp == nil {
		t.Fatal("response should not be nil")
	}
	if resp.Permission != "admin" {
		t.Error("role permission should be admin")
	}
	if resp.ScopeType != "university" {
		t.Error("role scope_type should be university")
	}
}

func TestToRoleResponse_Nil(t *testing.T) {
	resp := ToRoleResponse(nil)
	if resp != nil {
		t.Error("nil input should return nil")
	}
}

func TestToSessionsResponse(t *testing.T) {
	now := time.Now()
	device := "Mozilla/5.0"
	ip := "192.168.1.1"

	sessions := []Session{
		{
			ID:        uuid.New(),
			Device:    &device,
			IPAddress: &ip,
			CreatedAt: now,
			ExpiresAt: now.Add(7 * 24 * time.Hour),
		},
	}

	resp := ToSessionsResponse(sessions)

	if len(resp) != 1 {
		t.Fatalf("expected 1 session, got %d", len(resp))
	}
	if resp[0].Device == nil || *resp[0].Device != device {
		t.Error("device should match")
	}
}

func TestToStaffProfileResponse(t *testing.T) {
	if ToStaffProfileResponse(nil) != nil {
		t.Error("nil input should return nil")
	}

	salary := "1500.50"
	currency := "USD"
	degree := "phd"

	profile := &StaffProfile{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		HighestDegree:  &degree,
		YearsOfService: 5,
		Salary:         &salary,
		SalaryCurrency: &currency,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	resp := ToStaffProfileResponse(profile)

	if resp == nil {
		t.Fatal("response should not be nil")
	}
	if resp.YearsOfService != 5 {
		t.Error("years of service should match")
	}
	if resp.Salary == nil || *resp.Salary != 1500.50 {
		t.Errorf("salary should be 1500.50, got %v", resp.Salary)
	}
	if resp.SalaryCurrency == nil || *resp.SalaryCurrency != "USD" {
		t.Error("salary currency should be USD")
	}
}

func TestUpdateStaffProfileRequest_SalaryString(t *testing.T) {
	tests := []struct {
		name   string
		salary *float64
		want   *string
	}{
		{"nil salary", nil, nil},
		{"valid salary", float64Ptr(1234.56), strPtr("1234.56")},
		{"round salary", float64Ptr(100.0), strPtr("100.00")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := UpdateStaffProfileRequest{Salary: tt.salary}
			result := req.SalaryString()
			if tt.want == nil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if tt.want != nil && (result == nil || *result != *tt.want) {
				t.Errorf("expected %v, got %v", *tt.want, result)
			}
		})
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}

func strPtr(s string) *string {
	return &s
}
