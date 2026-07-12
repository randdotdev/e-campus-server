package http

import (
	"errors"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/identity"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type RegisterRequest struct {
	Email         string  `json:"email" binding:"required,email"`
	Password      string  `json:"password" binding:"required,min=8"`
	FullNameEN    string  `json:"full_name_en" binding:"required,min=2,max=255"`
	FullNameLocal *string `json:"full_name_local" binding:"omitempty,max=255"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthUserResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	FullNameEN    string    `json:"full_name_en"`
	FullNameLocal *string   `json:"full_name_local,omitempty"`
	AvatarURL     *string   `json:"avatar_url,omitempty"`
	IsVerified    bool      `json:"is_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

type LoginResponse struct {
	AccessToken string           `json:"access_token"`
	ExpiresAt   time.Time        `json:"expires_at"`
	User        AuthUserResponse `json:"user"`
}

type RefreshResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func toAuthUserResponse(u *identity.UserData) AuthUserResponse {
	return AuthUserResponse{
		ID: u.ID, Email: u.Email, FullNameEN: u.FullNameEN, FullNameLocal: u.FullNameLocal,
		AvatarURL: u.AvatarURL, IsVerified: u.IsVerified, CreatedAt: u.CreatedAt,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	user, err := h.auth.Register(c.Request.Context(), identity.RegisterInput{
		Email: req.Email, Password: req.Password, FullNameEN: req.FullNameEN, FullNameLocal: req.FullNameLocal,
	})
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrEmailExists):
			response.Conflict(c, "email already exists")
		case errors.Is(err, identity.ErrPasswordTooShort), errors.Is(err, identity.ErrPasswordTooWeak):
			response.BadRequest(c, err.Error())
		default:
			h.log.Error("register failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}
	response.Created(c, toAuthUserResponse(user))
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	tokens, user, err := h.auth.Login(c.Request.Context(), req.Email, req.Password, c.GetHeader("User-Agent"), c.ClientIP())
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrInvalidCredentials):
			response.Unauthorized(c, "invalid email or password")
		case errors.Is(err, identity.ErrUserInactive):
			response.Forbidden(c, "account is deactivated")
		default:
			h.log.Error("login failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	response.OK(c, LoginResponse{AccessToken: tokens.AccessToken, ExpiresAt: tokens.ExpiresAt, User: toAuthUserResponse(user)})
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		response.Unauthorized(c, "refresh token required")
		return
	}
	tokens, err := h.auth.Refresh(c.Request.Context(), refreshToken, c.GetHeader("User-Agent"), c.ClientIP())
	if err != nil {
		h.clearRefreshCookie(c)
		switch {
		case errors.Is(err, identity.ErrTokenExpired):
			response.Err(c, nethttp.StatusUnauthorized, "REFRESH_TOKEN_EXPIRED", "refresh token expired")
		case errors.Is(err, identity.ErrInvalidToken), errors.Is(err, identity.ErrTokenReused):
			response.Unauthorized(c, "invalid refresh token")
		case errors.Is(err, identity.ErrUserInactive):
			response.Forbidden(c, "account is deactivated")
		default:
			h.log.Error("refresh failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}
	h.setRefreshCookie(c, tokens.RefreshToken)
	response.OK(c, RefreshResponse{AccessToken: tokens.AccessToken, ExpiresAt: tokens.ExpiresAt})
}

func (h *Handler) Logout(c *gin.Context) {
	if refreshToken, err := c.Cookie("refresh_token"); err == nil && refreshToken != "" {
		if err := h.auth.Logout(c.Request.Context(), refreshToken); err != nil {
			h.log.Warn("logout failed", zap.Error(err))
		}
	}
	h.clearRefreshCookie(c)
	response.NoContent(c)
}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(nethttp.SameSiteStrictMode)
	c.SetCookie("refresh_token", token, int(7*24*time.Hour.Seconds()), "/api/v1/auth", "", h.secureCookie, true)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(nethttp.SameSiteStrictMode)
	c.SetCookie("refresh_token", "", -1, "/api/v1/auth", "", h.secureCookie, true)
}
