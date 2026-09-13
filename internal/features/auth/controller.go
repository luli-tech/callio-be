package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	accountFeature "github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/internal/shared/session"
	"github.com/luli-tech/twilio-Boss/internal/transport/httpapi"
)

// AuthController handles tenant portal authentication endpoints.
type AuthController struct {
	authService ServiceContract
}

// NewAuthController constructs a new AuthController.
func NewAuthController(authService ServiceContract) *AuthController {
	return &AuthController{authService: authService}
}

// Register handles POST /v1/auth/register.
func (ctrl *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.InvalidRequest(c, "Valid company, email, and password are required")
		return
	}

	resp, err := ctrl.authService.Register(c.Request.Context(), &req)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login handles POST /v1/auth/login.
func (ctrl *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.InvalidRequest(c, "Valid email and password are required")
		return
	}

	resp, err := ctrl.authService.Login(c.Request.Context(), &req)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Refresh handles POST /v1/auth/refresh.
func (ctrl *AuthController) Refresh(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.InvalidRequest(c, "Refresh token is required")
		return
	}

	resp, err := ctrl.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RotateCredentials handles POST /v1/auth/account/credentials/rotate.
func (ctrl *AuthController) RotateCredentials(c *gin.Context) {
	acc, ok := requireAccount(c)
	if !ok {
		return
	}

	updated, err := ctrl.authService.RotateAccountCredentials(c.Request.Context(), acc.ID)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"sid":        updated.SID,
		"auth_token": updated.AuthToken,
	})
}

func requireAccount(c *gin.Context) (*accountFeature.Account, bool) {
	val, exists := c.Get(session.ContextAccountKey)
	acc, ok := val.(*accountFeature.Account)
	if !exists || !ok || acc == nil {
		httpapi.Unauthorized(c, "UNAUTHORIZED", "Authentication required")
		return nil, false
	}
	return acc, true
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apperrors.ErrEmailAlreadyExists):
		httpapi.WriteError(c, httpapi.Error{"EMAIL_ALREADY_EXISTS", "Email address is already registered", http.StatusConflict})
	case errors.Is(err, apperrors.ErrInvalidCredentials), errors.Is(err, apperrors.ErrInvalidPassword):
		httpapi.Unauthorized(c, "INVALID_CREDENTIALS", "Invalid email or password")
	case errors.Is(err, apperrors.ErrPasswordTooWeak):
		httpapi.WriteError(c, httpapi.Error{"PASSWORD_TOO_WEAK", "Password must be at least 8 characters", http.StatusBadRequest})
	case errors.Is(err, apperrors.ErrTokenExpired):
		httpapi.Unauthorized(c, "TOKEN_EXPIRED", "Token has expired")
	case errors.Is(err, apperrors.ErrInvalidToken):
		httpapi.Unauthorized(c, "INVALID_TOKEN", "Token is invalid or malformed")
	case errors.Is(err, apperrors.ErrUserSuspended), errors.Is(err, apperrors.ErrAccountSuspended):
		httpapi.WriteError(c, httpapi.Error{"ACCOUNT_DISABLED", "Account or user is not active", http.StatusForbidden})
	default:
		httpapi.InternalError(c, "Authentication service failure")
	}
}
