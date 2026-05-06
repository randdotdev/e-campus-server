package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service      *Service
	log          *zap.Logger
	secureCookie bool
}

func NewHandler(service *Service, log *zap.Logger, secureCookie bool) *Handler {
	return &Handler{
		service:      service,
		log:          log,
		secureCookie: secureCookie,
	}
}

// Auth handlers

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			response.Conflict(c, "email already exists")
			return
		}
		if errors.Is(err, ErrPasswordTooShort) || errors.Is(err, ErrPasswordTooWeak) {
			response.BadRequest(c, err.Error())
			return
		}
		h.log.Error("register failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToUserResponse(user))
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	device := c.GetHeader("User-Agent")
	ip := c.ClientIP()

	tokens, user, err := h.service.Login(c.Request.Context(), req, device, ip)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Unauthorized(c, "invalid email or password")
			return
		}
		if errors.Is(err, ErrUserInactive) {
			response.Forbidden(c, "account is deactivated")
			return
		}
		h.log.Error("login failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)

	response.OK(c, LoginResponse{
		AccessToken: tokens.AccessToken,
		ExpiresAt:   tokens.ExpiresAt,
		User:        ToUserResponse(user),
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		response.Unauthorized(c, "refresh token required")
		return
	}

	device := c.GetHeader("User-Agent")
	ip := c.ClientIP()

	tokens, err := h.service.Refresh(c.Request.Context(), refreshToken, device, ip)
	if err != nil {
		h.clearRefreshCookie(c)

		if errors.Is(err, ErrTokenExpired) {
			response.Err(c, http.StatusUnauthorized, "REFRESH_TOKEN_EXPIRED", "refresh token expired")
			return
		}
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrTokenReused) {
			response.Unauthorized(c, "invalid refresh token")
			return
		}
		if errors.Is(err, ErrUserInactive) {
			response.Forbidden(c, "account is deactivated")
			return
		}
		h.log.Error("refresh failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	h.setRefreshCookie(c, tokens.RefreshToken)

	response.OK(c, RefreshResponse{
		AccessToken: tokens.AccessToken,
		ExpiresAt:   tokens.ExpiresAt,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		if err := h.service.Logout(c.Request.Context(), refreshToken); err != nil {
			h.log.Warn("logout failed", zap.Error(err))
		}
	}

	h.clearRefreshCookie(c)
	response.NoContent(c)
}

// Helper functions

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		token,
		int(7*24*time.Hour.Seconds()),
		"/api/v1/auth",
		"",
		h.secureCookie,
		true,
	)
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/v1/auth",
		"",
		h.secureCookie,
		true,
	)
}

func ExtractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		// Browser WebSocket API cannot set headers; fall back to query param.
		return c.Query("access_token")
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}
