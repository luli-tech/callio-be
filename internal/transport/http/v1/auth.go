package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/internal/transport/http/middleware"
)

// AuthController handles tenant portal authentication endpoints.
type AuthController struct {
	authService domain.AuthService
}

// NewAuthController constructs a new AuthController.
func NewAuthController(authService domain.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// Register handles POST /v1/auth/register.
func (ctrl *AuthController) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Valid company, email, and password are required"})
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
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Valid email and password are required"})
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
	var req domain.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Refresh token is required"})
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
	acc := middleware.GetAccount(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
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

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"code": "EMAIL_ALREADY_EXISTS", "message": "Email address is already registered"})
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrInvalidPassword):
		c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password"})
	case errors.Is(err, domain.ErrPasswordTooWeak):
		c.JSON(http.StatusBadRequest, gin.H{"code": "PASSWORD_TOO_WEAK", "message": "Password must be at least 8 characters"})
	case errors.Is(err, domain.ErrTokenExpired):
		c.JSON(http.StatusUnauthorized, gin.H{"code": "TOKEN_EXPIRED", "message": "Token has expired"})
	case errors.Is(err, domain.ErrInvalidToken):
		c.JSON(http.StatusUnauthorized, gin.H{"code": "INVALID_TOKEN", "message": "Token is invalid or malformed"})
	case errors.Is(err, domain.ErrUserSuspended), errors.Is(err, domain.ErrAccountSuspended):
		c.JSON(http.StatusForbidden, gin.H{"code": "ACCOUNT_DISABLED", "message": "Account or user is not active"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Authentication service failure"})
	}
}
