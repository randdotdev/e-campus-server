package notification

import (
	"net/http"
	"testing"
)

func TestCheckOrigin_NoRestrictions(t *testing.T) {
	h := NewHandler(nil, nil, nil)

	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://evil.com")

	if !h.checkOrigin(req) {
		t.Error("should allow all origins when no restrictions")
	}
}

func TestCheckOrigin_Wildcard(t *testing.T) {
	h := NewHandler(nil, nil, nil, "*")

	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://anything.com")

	if !h.checkOrigin(req) {
		t.Error("should allow all origins with wildcard")
	}
}

func TestCheckOrigin_SpecificOrigin(t *testing.T) {
	h := NewHandler(nil, nil, nil, "https://example.com", "https://app.example.com")

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"https://example.com", true},
		{"https://app.example.com", true},
		{"https://evil.com", false},
		{"https://example.com.evil.com", false},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", tt.origin)

		got := h.checkOrigin(req)
		if got != tt.allowed {
			t.Errorf("checkOrigin(%q) = %v, want %v", tt.origin, got, tt.allowed)
		}
	}
}

func TestCheckOrigin_EmptyOriginHeader(t *testing.T) {
	h := NewHandler(nil, nil, nil, "https://example.com")

	req, _ := http.NewRequest("GET", "/ws", nil)
	// No Origin header

	if h.checkOrigin(req) {
		t.Error("should reject when Origin header is missing and restrictions are set")
	}
}
