package user

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestCheckPassword(t *testing.T) {
	password := "password123"
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}
	hash := string(hashBytes)

	if !checkPassword(password, hash) {
		t.Error("checkPassword should return true for correct password")
	}

	if checkPassword("wrongpassword", hash) {
		t.Error("checkPassword should return false for wrong password")
	}
}

func TestDerefInt(t *testing.T) {
	val := 42
	ptr := &val

	if derefInt(ptr, 0) != 42 {
		t.Error("derefInt should return pointer value")
	}

	if derefInt(nil, 10) != 10 {
		t.Error("derefInt should return default for nil")
	}
}

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

func TestToRolesResponse(t *testing.T) {
	roles := []Role{
		{ID: uuid.New(), Permission: "admin", ScopeType: "university"},
		{ID: uuid.New(), Permission: "viewer", ScopeType: "college"},
	}

	resp := ToRolesResponse(roles)

	if len(resp) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(resp))
	}
	if resp[0].Permission != "admin" {
		t.Error("first role permission should be admin")
	}
	if resp[1].ScopeType != "college" {
		t.Error("second role scope_type should be college")
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
	req := UpdateStaffProfileRequest{}
	if req.SalaryString() != nil {
		t.Error("nil salary should return nil string")
	}

	salary := 1234.56
	req.Salary = &salary
	result := req.SalaryString()
	if result == nil || *result != "1234.56" {
		t.Errorf("expected '1234.56', got %v", result)
	}

	salary = 100.0
	req.Salary = &salary
	result = req.SalaryString()
	if result == nil || *result != "100.00" {
		t.Errorf("expected '100.00', got %v", result)
	}
}

func TestChangePasswordRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		current string
		new     string
		same    bool
	}{
		{"different passwords", "oldpass123", "newpass456", false},
		{"same passwords", "samepass123", "samepass123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChangePasswordRequest{
				CurrentPassword: tt.current,
				NewPassword:     tt.new,
			}
			same := req.CurrentPassword == req.NewPassword
			if same != tt.same {
				t.Errorf("expected same=%v, got %v", tt.same, same)
			}
		})
	}
}

func TestCreateStaffUserRequest_Fields(t *testing.T) {
	degree := "phd"
	salary := 1500.0

	req := CreateStaffUserRequest{
		Email:      "staff@test.com",
		Password:   "password123",
		FullNameEN: "Test Staff",
		StaffProfile: UpdateStaffProfileRequest{
			HighestDegree: &degree,
			Salary:        &salary,
		},
	}

	if req.Email != "staff@test.com" {
		t.Error("email should match")
	}
	if req.StaffProfile.HighestDegree == nil || *req.StaffProfile.HighestDegree != "phd" {
		t.Error("highest degree should be phd")
	}

	salaryStr := req.StaffProfile.SalaryString()
	if salaryStr == nil || *salaryStr != "1500.00" {
		t.Errorf("expected salary string '1500.00', got %v", salaryStr)
	}
}

func TestCreateRoleRequest_Validation(t *testing.T) {
	tests := []struct {
		name       string
		permission string
		scopeType  string
		valid      bool
	}{
		{"valid admin university", "admin", "university", true},
		{"valid viewer college", "viewer", "college", true},
		{"valid operator department", "operator", "department", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreateRoleRequest{
				Permission: tt.permission,
				ScopeType:  tt.scopeType,
			}
			if req.Permission != tt.permission {
				t.Errorf("permission should be %s", tt.permission)
			}
			if req.ScopeType != tt.scopeType {
				t.Errorf("scope_type should be %s", tt.scopeType)
			}
		})
	}
}

func TestScopeIDValidation(t *testing.T) {
	scopeID := uuid.New()

	tests := []struct {
		name      string
		scopeType string
		scopeID   *uuid.UUID
		wantErr   bool
		errType   string
	}{
		{"university without scope_id", "university", nil, false, ""},
		{"university with scope_id", "university", &scopeID, true, "not_allowed"},
		{"college without scope_id", "college", nil, true, "required"},
		{"college with scope_id", "college", &scopeID, false, ""},
		{"department without scope_id", "department", nil, true, "required"},
		{"department with scope_id", "department", &scopeID, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hasErr bool
			var errType string

			if tt.scopeType == "university" && tt.scopeID != nil {
				hasErr = true
				errType = "not_allowed"
			} else if tt.scopeType != "university" && tt.scopeID == nil {
				hasErr = true
				errType = "required"
			}

			if hasErr != tt.wantErr {
				t.Errorf("expected error=%v, got %v", tt.wantErr, hasErr)
			}
			if hasErr && errType != tt.errType {
				t.Errorf("expected errType=%s, got %s", tt.errType, errType)
			}
		})
	}
}
