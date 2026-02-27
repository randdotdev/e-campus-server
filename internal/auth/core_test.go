package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	password := "password123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == password {
		t.Error("hash should not equal plaintext password")
	}

	if len(hash) < 50 {
		t.Error("hash seems too short for bcrypt")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "password123"
	hash, _ := HashPassword(password)

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{"correct password", password, true},
		{"wrong password", "wrongpassword", false},
		{"empty password", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPassword(tt.password, hash); got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	token := "test-token-123"

	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Error("HashToken should be deterministic")
	}

	if hash1 == token {
		t.Error("hash should not equal original token")
	}

	if len(hash1) != 64 {
		t.Errorf("SHA256 hex should be 64 chars, got %d", len(hash1))
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token1 := GenerateRefreshToken()
	token2 := GenerateRefreshToken()

	if token1 == token2 {
		t.Error("GenerateRefreshToken should produce unique tokens")
	}

	if len(token1) != 36 {
		t.Errorf("UUID string should be 36 chars, got %d", len(token1))
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"past time", time.Now().Add(-time.Hour), true},
		{"future time", time.Now().Add(time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTokenExpired(tt.expiresAt); got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTokenUsed(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		usedAt *time.Time
		want   bool
	}{
		{"nil", nil, false},
		{"non-nil", &now, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTokenUsed(tt.usedAt); got != tt.want {
				t.Errorf("IsTokenUsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildRoleClaim(t *testing.T) {
	titleEN := "Admin"
	titleLocal := "بەڕێوەبەر"
	scopeID := uuid.New()

	role := &RoleData{ID: uuid.New(), TitleEN: &titleEN, TitleLocal: &titleLocal, Permission: "admin", ScopeType: "university", ScopeID: &scopeID}

	claim := BuildRoleClaim(role)

	if claim == nil {
		t.Fatal("claim should not be nil")
	}

	if claim["permission"] != "admin" {
		t.Error("claim permission should be admin")
	}

	if claim["title_en"] != "Admin" {
		t.Error("claim should have title_en")
	}

	if claim["title_local"] != "بەڕێوەبەر" {
		t.Error("claim should have title_local")
	}

	if claim["scope_id"] != scopeID.String() {
		t.Error("claim should have scope_id")
	}
}

func TestBuildRoleClaim_NilRole(t *testing.T) {
	claim := BuildRoleClaim(nil)
	if claim != nil {
		t.Error("BuildRoleClaim(nil) should return nil")
	}
}

func TestStrPtr(t *testing.T) {
	tests := []struct {
		name  string
		input string
		isNil bool
	}{
		{"empty string", "", true},
		{"non-empty string", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strPtr(tt.input)
			if tt.isNil && result != nil {
				t.Error("expected nil")
			}
			if !tt.isNil && (result == nil || *result != tt.input) {
				t.Errorf("expected %q, got %v", tt.input, result)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"key1": "value1",
		"key2": 123,
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"string value", "key1", "value1"},
		{"non-string value", "key2", ""},
		{"missing key", "missing", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getString(m, tt.key); got != tt.want {
				t.Errorf("getString() = %q, want %q", got, tt.want)
			}
		})
	}
}
