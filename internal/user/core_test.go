package user

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCheckPassword(t *testing.T) {
	password := "password123"
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to generate hash: %v", err)
	}
	hash := string(hashBytes)

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "wrongpassword", hash, false},
		{"empty password", "", hash, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkPassword(tt.password, tt.hash); got != tt.want {
				t.Errorf("checkPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDerefInt(t *testing.T) {
	tests := []struct {
		name       string
		ptr        *int
		defaultVal int
		want       int
	}{
		{"nil pointer", nil, 10, 10},
		{"valid pointer", intPtr(42), 0, 42},
		{"zero value pointer", intPtr(0), 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := derefInt(tt.ptr, tt.defaultVal); got != tt.want {
				t.Errorf("derefInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
