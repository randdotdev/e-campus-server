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

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}

	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword should return false for wrong password")
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
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	if !IsTokenExpired(past) {
		t.Error("past time should be expired")
	}

	if IsTokenExpired(future) {
		t.Error("future time should not be expired")
	}
}

func TestIsTokenUsed(t *testing.T) {
	now := time.Now()

	if IsTokenUsed(nil) {
		t.Error("nil should mean not used")
	}

	if !IsTokenUsed(&now) {
		t.Error("non-nil time should mean used")
	}
}

func TestBuildRoleClaims(t *testing.T) {
	title := "Admin"
	scopeID := uuid.New()

	roles := []RoleData{
		{ID: uuid.New(), Permission: "admin", ScopeType: "university"},
		{ID: uuid.New(), Title: &title, Permission: "viewer", ScopeType: "college", ScopeID: &scopeID},
	}

	claims := BuildRoleClaims(roles)

	if len(claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(claims))
	}

	if claims[0]["permission"] != "admin" {
		t.Error("first claim permission should be admin")
	}

	if claims[1]["title"] != "Admin" {
		t.Error("second claim should have title")
	}

	if claims[1]["scope_id"] != scopeID.String() {
		t.Error("second claim should have scope_id")
	}
}

func TestStrPtr(t *testing.T) {
	result := strPtr("")
	if result != nil {
		t.Error("strPtr should return nil for empty string")
	}

	result = strPtr("test")
	if result == nil || *result != "test" {
		t.Error("strPtr should return pointer to non-empty string")
	}
}

func TestGetString(t *testing.T) {
	m := map[string]any{
		"key1": "value1",
		"key2": 123,
	}

	if getString(m, "key1") != "value1" {
		t.Error("getString should return string value")
	}

	if getString(m, "key2") != "" {
		t.Error("getString should return empty for non-string")
	}

	if getString(m, "missing") != "" {
		t.Error("getString should return empty for missing key")
	}
}
