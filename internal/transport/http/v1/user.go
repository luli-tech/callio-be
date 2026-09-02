package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/internal/transport/http/middleware"
)

// UserController handles tenant user management endpoints.
type UserController struct {
	userService domain.UserService
}

// NewUserController constructs a new UserController.
func NewUserController(userService domain.UserService) *UserController {
	return &UserController{userService: userService}
}

// Me handles GET /v1/users/me.
func (ctrl *UserController) Me(c *gin.Context) {
	claims := middleware.GetTokenClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "JWT_REQUIRED", "message": "Bearer token authentication is required"})
		return
	}

	user, err := ctrl.userService.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// List handles GET /v1/users.
func (ctrl *UserController) List(c *gin.Context) {
	claims := middleware.GetTokenClaims(c)
	if !hasAnyRole(claims, domain.RoleOwner, domain.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Owner or admin role required"})
		return
	}

	users, err := ctrl.userService.ListUsersByAccount(c.Request.Context(), claims.AccountID)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// Create handles POST /v1/users.
func (ctrl *UserController) Create(c *gin.Context) {
	claims := middleware.GetTokenClaims(c)
	if !hasAnyRole(claims, domain.RoleOwner, domain.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Owner or admin role required"})
		return
	}

	var req domain.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Valid email, password, and role are required"})
		return
	}
	req.AccountID = claims.AccountID

	user, err := ctrl.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

// UpdateMe handles PATCH /v1/users/me.
func (ctrl *UserController) UpdateMe(c *gin.Context) {
	claims := middleware.GetTokenClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "JWT_REQUIRED", "message": "Bearer token authentication is required"})
		return
	}

	var req domain.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Invalid profile payload"})
		return
	}

	user, err := ctrl.userService.UpdateProfile(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateRole handles PATCH /v1/users/:id/role.
func (ctrl *UserController) UpdateRole(c *gin.Context) {
	claims := middleware.GetTokenClaims(c)
	if !hasAnyRole(claims, domain.RoleOwner, domain.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Owner or admin role required"})
		return
	}

	var req struct {
		Role domain.UserRole `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Role is required"})
		return
	}

	user, err := ctrl.userService.UpdateRole(c.Request.Context(), c.Param("id"), req.Role)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateStatus handles PATCH /v1/users/:id/status.
func (ctrl *UserController) UpdateStatus(c *gin.Context) {
	claims := middleware.GetTokenClaims(c)
	if !hasAnyRole(claims, domain.RoleOwner, domain.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Owner or admin role required"})
		return
	}

	var req struct {
		Status domain.UserStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Status is required"})
		return
	}

	user, err := ctrl.userService.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// Delete handles DELETE /v1/users/:id.
func (ctrl *UserController) Delete(c *gin.Context) {
	claims := middleware.GetTokenClaims(c)
	if !hasAnyRole(claims, domain.RoleOwner, domain.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "Owner or admin role required"})
		return
	}

	if err := ctrl.userService.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		writeUserError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func hasAnyRole(claims *domain.TokenClaims, roles ...domain.UserRole) bool {
	if claims == nil {
		return false
	}
	for _, role := range roles {
		if claims.Role == role {
			return true
		}
	}
	return false
}

func writeUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"code": "USER_NOT_FOUND", "message": "User was not found"})
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"code": "EMAIL_ALREADY_EXISTS", "message": "Email address is already registered"})
	case errors.Is(err, domain.ErrPasswordTooWeak):
		c.JSON(http.StatusBadRequest, gin.H{"code": "PASSWORD_TOO_WEAK", "message": "Password must be at least 8 characters"})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_VALUE", "message": "Invalid role or status"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "User service failure"})
	}
}
