package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/randdotdev/e-campus-server/internal/shared/config"
)

// ── Auth models ────────────────────────────────────────────────────────────

// RefreshToken is one rotation-chain entry of a login session. Tokens sharing
// a Family descend from one login; reuse of any member revokes the family.
// The json tags are the Redis store's serialisation contract (redis/tokens.go),
// not an HTTP shape — http/ has its own DTOs.
type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"token_hash"`
	Family    uuid.UUID  `json:"family"`
	Device    *string    `json:"device,omitempty"`
	IPAddress *string    `json:"ip_address,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// TokenPair is what a successful login or refresh hands the client: a short-
// lived access token and the next refresh token in the rotation chain.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// JWTClaims is the decoded payload of a verified access token.
type JWTClaims struct {
	UserID uuid.UUID  `json:"sub"`
	Email  string     `json:"email"`
	Role   *RoleClaim `json:"role"`
}

// RoleClaim is the institutional role as embedded in the access token. Level
// and ScopeType are raw strings here deliberately: they are authz's vocabulary,
// typed when that context migrates.
type RoleClaim struct {
	ID         uuid.UUID  `json:"id"`
	TitleEN    string     `json:"title_en,omitempty"`
	TitleLocal string     `json:"title_local,omitempty"`
	Level      string     `json:"level"`
	ScopeType  string     `json:"scope_type"`
	ScopeID    *uuid.UUID `json:"scope_id,omitempty"`
	Domain     string     `json:"domain,omitempty"`
}

// UserData is auth's projection of a user (credentials view).
type UserData struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	FullNameEN    string
	FullNameLocal *string
	AvatarURL     *string
	IsActive      bool
	IsVerified    bool
	CreatedAt     time.Time
}

// RoleData is auth's projection of a user's institutional role, used to build
// the role claim of an access token.
type RoleData struct {
	ID         uuid.UUID
	TitleEN    *string
	TitleLocal *string
	Level      string
	ScopeType  string
	ScopeID    *uuid.UUID
	Domain     *string
}

// ── Core (pure) ────────────────────────────────────────────────────────────

const bcryptCost = 12
const minPasswordLength = 8

// ValidatePassword checks the password policy: at least eight characters with
// an uppercase letter, a lowercase letter, and a digit. It returns
// ErrPasswordTooShort or ErrPasswordTooWeak on violation.
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

// HashPassword returns the bcrypt hash of a password.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(b), err
}

// CheckPassword reports whether a password matches its bcrypt hash.
func CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// HashToken returns the hex SHA-256 of a refresh token; only hashes are stored.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// GenerateRefreshToken returns a fresh opaque refresh token.
func GenerateRefreshToken() string { return uuid.New().String() }

// IsTokenExpired reports whether expiresAt has passed.
func IsTokenExpired(expiresAt time.Time) bool { return time.Now().After(expiresAt) }

// BuildRoleClaim converts a role projection to the claim map embedded in the
// access token; a nil role yields nil (no role claim).
func BuildRoleClaim(role *RoleData) map[string]any {
	if role == nil {
		return nil
	}
	claim := map[string]any{"id": role.ID.String(), "level": role.Level, "scope_type": role.ScopeType}
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

// GenerateAccessToken signs an HS256 access token for the user, optionally
// embedding a role claim built by BuildRoleClaim.
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
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseAccessToken verifies an access token's signature and expiry and returns
// its claims. It returns ErrTokenExpired for a stale token and ErrInvalidToken
// for anything else that fails verification.
func ParseAccessToken(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
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
	jc := &JWTClaims{UserID: userID, Email: email}
	if roleMap, ok := claims["role"].(map[string]any); ok {
		idStr, _ := roleMap["id"].(string)
		roleID, _ := uuid.Parse(idStr)
		role := RoleClaim{
			ID:         roleID,
			Level:      jwtString(roleMap, "level"),
			ScopeType:  jwtString(roleMap, "scope_type"),
			TitleEN:    jwtString(roleMap, "title_en"),
			TitleLocal: jwtString(roleMap, "title_local"),
		}
		if s := jwtString(roleMap, "scope_id"); s != "" {
			if parsed, err := uuid.Parse(s); err == nil {
				role.ScopeID = &parsed
			}
		}
		role.Domain = jwtString(roleMap, "domain")
		jc.Role = &role
	}
	return jc, nil
}

func jwtString(m map[string]any, key string) string {
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

// ── Auth ports ─────────────────────────────────────────────────────────────

// TokenRepository is the refresh-token store of record (Redis, with TTL
// semantics — losing an entry ends that session). Get and delete methods treat
// a missing token as ErrTokenNotFound and a no-op respectively.
type TokenRepository interface {
	CreateToken(ctx context.Context, token *RefreshToken) error
	GetTokenByHash(ctx context.Context, hash string) (*RefreshToken, error)
	// MarkTokenUsed atomically flags the token as consumed and reports whether
	// it had already been consumed — the reuse-detection compare-and-swap.
	MarkTokenUsed(ctx context.Context, hash string) (alreadyUsed bool, err error)
	// InvalidateFamily revokes every token descended from one login.
	InvalidateFamily(ctx context.Context, family uuid.UUID) error
	DeleteToken(ctx context.Context, hash string) error
	DeleteUserTokens(ctx context.Context, userID uuid.UUID) error
	GetUserSessions(ctx context.Context, userID uuid.UUID) ([]RefreshToken, error)
}

// AuthUserStore is auth's narrow view of the user store (the postgres user repo
// satisfies it). Create returns ErrEmailExists when the email is taken.
type AuthUserStore interface {
	Create(ctx context.Context, email, passwordHash, fullNameEN string, fullNameLocal *string) (*UserData, error)
	GetByEmail(ctx context.Context, email string) (*UserData, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserData, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	GetUserRole(ctx context.Context, userID uuid.UUID) (*RoleData, error)
}

// ── AuthService ────────────────────────────────────────────────────────────

// RegisterInput is what self-registration needs.
type RegisterInput struct {
	Email         string
	Password      string
	FullNameEN    string
	FullNameLocal *string
}

// AuthService is registration, login, token refresh, and logout.
type AuthService struct {
	tokens TokenRepository
	users  AuthUserStore
	jwt    *config.JWTConfig
	log    *slog.Logger
}

// NewAuthService wires the auth use cases.
func NewAuthService(tokens TokenRepository, users AuthUserStore, jwt *config.JWTConfig, log *slog.Logger) *AuthService {
	return &AuthService{tokens: tokens, users: users, jwt: jwt, log: log}
}

// Register creates a user account. It returns ErrEmailExists when the email is
// taken and the password-policy sentinels when the password is rejected.
func (s *AuthService) Register(ctx context.Context, req RegisterInput) (*UserData, error) {
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}
	// Friendly pre-check only; the users.email UNIQUE constraint decides races.
	exists, err := s.users.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	return s.users.Create(ctx, req.Email, hash, req.FullNameEN, req.FullNameLocal)
}

// Login verifies credentials and starts a new session (a fresh token family).
// It returns ErrInvalidCredentials on failure and ErrUserInactive for a
// deactivated account.
func (s *AuthService) Login(ctx context.Context, email, password, device, ip string) (*TokenPair, *UserData, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, err
	}
	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}
	if !CheckPassword(password, user.PasswordHash) {
		return nil, nil, ErrInvalidCredentials
	}
	tokens, err := s.createTokenPair(ctx, user, uuid.New(), device, ip)
	if err != nil {
		return nil, nil, err
	}
	return tokens, user, nil
}

// Refresh rotates a refresh token: the presented token is consumed exactly once
// and a new pair is issued in the same family. A second presentation of the
// same token means it leaked; the whole family is revoked and ErrTokenReused
// is returned.
func (s *AuthService) Refresh(ctx context.Context, refreshToken, device, ip string) (*TokenPair, error) {
	hash := HashToken(refreshToken)
	stored, err := s.tokens.GetTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if IsTokenExpired(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}
	alreadyUsed, err := s.tokens.MarkTokenUsed(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if alreadyUsed {
		s.log.ErrorContext(ctx, "refresh token reuse detected; revoking token family",
			"user_id", stored.UserID, "family", stored.Family)
		if err := s.tokens.InvalidateFamily(ctx, stored.Family); err != nil {
			// The compromised family is still live — this failure must be seen.
			s.log.ErrorContext(ctx, "revoking reused token family failed",
				"user_id", stored.UserID, "family", stored.Family, "error", err)
		}
		return nil, ErrTokenReused
	}
	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}
	return s.createTokenPair(ctx, user, stored.Family, device, ip)
}

// Logout revokes the presented refresh token; unknown tokens are a no-op.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.tokens.DeleteToken(ctx, HashToken(refreshToken))
}

// LogoutAll revokes every session of the user.
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.tokens.DeleteUserTokens(ctx, userID)
}

// ValidateAccessToken verifies an access token and returns its claims.
func (s *AuthService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	return ParseAccessToken(tokenString, s.jwt.Secret)
}

func (s *AuthService) createTokenPair(ctx context.Context, user *UserData, family uuid.UUID, device, ip string) (*TokenPair, error) {
	role, err := s.users.GetUserRole(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(s.jwt.AccessTTL)
	accessToken, err := GenerateAccessToken(user.ID, user.Email, BuildRoleClaim(role), expiresAt, s.jwt.Secret)
	if err != nil {
		return nil, err
	}
	refreshToken := GenerateRefreshToken()
	dbToken := &RefreshToken{
		UserID:    user.ID,
		TokenHash: HashToken(refreshToken),
		Family:    family,
		Device:    strPtr(device),
		IPAddress: strPtr(ip),
		ExpiresAt: time.Now().Add(s.jwt.RefreshTTL),
	}
	if err := s.tokens.CreateToken(ctx, dbToken); err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: expiresAt}, nil
}
