package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12
const minPasswordLength = 8

// ValidatePassword checks password strength.
// Returns nil if valid, error message if invalid.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return ErrPasswordTooWeak
	}

	return nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func GenerateRefreshToken() string {
	return uuid.New().String()
}

func IsTokenExpired(expiresAt time.Time) bool {
	return time.Now().After(expiresAt)
}

func IsTokenUsed(usedAt *time.Time) bool {
	return usedAt != nil
}

func BuildRoleClaim(role *RoleData) map[string]any {
	if role == nil {
		return nil
	}
	claim := map[string]any{
		"id":         role.ID.String(),
		"level":      role.Level,
		"scope_type": role.ScopeType,
	}
	if role.TitleEN != nil {
		claim["title_en"] = *role.TitleEN
	}
	if role.TitleLocal != nil {
		claim["title_local"] = *role.TitleLocal
	}
	if role.ScopeID != nil {
		claim["scope_id"] = role.ScopeID.String()
	}
	if role.Domain != nil {
		claim["domain"] = *role.Domain
	}
	return claim
}

func GenerateAccessToken(userID uuid.UUID, email string, roleClaim map[string]any, expiresAt time.Time, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   expiresAt.Unix(),
	}
	if roleClaim != nil {
		claims["role"] = roleClaim
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseAccessToken(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return parseClaims(claims)
}

func parseClaims(claims jwt.MapClaims) (*JWTClaims, error) {
	sub, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(sub)
	if err != nil {
		return nil, ErrInvalidToken
	}

	email, _ := claims["email"].(string)

	jwtClaims := &JWTClaims{
		UserID: userID,
		Email:  email,
	}

	if roleMap, ok := claims["role"].(map[string]any); ok {
		idStr, _ := roleMap["id"].(string)
		roleID, _ := uuid.Parse(idStr)

		role := RoleClaim{
			ID:         roleID,
			Level:      getString(roleMap, "level"),
			ScopeType:  getString(roleMap, "scope_type"),
			TitleEN:    getString(roleMap, "title_en"),
			TitleLocal: getString(roleMap, "title_local"),
		}

		if scopeIDStr := getString(roleMap, "scope_id"); scopeIDStr != "" {
			if parsed, err := uuid.Parse(scopeIDStr); err == nil {
				role.ScopeID = &parsed
			}
		}
		role.Domain = getString(roleMap, "domain")

		jwtClaims.Role = &role
	}

	return jwtClaims, nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
