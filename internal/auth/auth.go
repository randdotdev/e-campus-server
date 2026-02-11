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

func BuildRoleClaims(roles []RoleData) []map[string]any {
	claims := make([]map[string]any, len(roles))
	for i, r := range roles {
		claim := map[string]any{
			"id":         r.ID.String(),
			"permission": r.Permission,
			"scope_type": r.ScopeType,
		}
		if r.Title != nil {
			claim["title"] = *r.Title
		}
		if r.ScopeID != nil {
			claim["scope_id"] = r.ScopeID.String()
		}
		claims[i] = claim
	}
	return claims
}

func GenerateAccessToken(userID uuid.UUID, email string, roleClaims []map[string]any, expiresAt time.Time, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   userID.String(),
		"email": email,
		"roles": roleClaims,
		"iat":   time.Now().Unix(),
		"exp":   expiresAt.Unix(),
	})
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

	if rolesData, ok := claims["roles"].([]any); ok {
		for _, r := range rolesData {
			roleMap, ok := r.(map[string]any)
			if !ok {
				continue
			}

			idStr, _ := roleMap["id"].(string)
			roleID, _ := uuid.Parse(idStr)

			role := RoleClaim{
				ID:         roleID,
				Permission: getString(roleMap, "permission"),
				ScopeType:  getString(roleMap, "scope_type"),
				Title:      getString(roleMap, "title"),
			}

			if scopeIDStr := getString(roleMap, "scope_id"); scopeIDStr != "" {
				if parsed, err := uuid.Parse(scopeIDStr); err == nil {
					role.ScopeID = &parsed
				}
			}

			jwtClaims.Roles = append(jwtClaims.Roles, role)
		}
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
