package user

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/internal/shared/session"
)

type Controller struct {
	userService ServiceContract
}

func NewController(userService ServiceContract) *Controller {
	return &Controller{userService: userService}
}

func (ctrl *Controller) Me(c *gin.Context) {
	claims, ok := requireClaims(c)
	if !ok {
		return
	}

	user, err := ctrl.userService.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *Controller) List(c *gin.Context) {
	claims, ok := requireRole(c, RoleOwner, RoleAdmin)
	if !ok {
		return
	}

	users, err := ctrl.userService.ListUsersByAccount(c.Request.Context(), claims.AccountID)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (ctrl *Controller) Create(c *gin.Context) {
	claims, ok := requireRole(c, RoleOwner, RoleAdmin)
	if !ok {
		return
	}

	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeInvalidRequest(c, "Valid email, password, and role are required")
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

func (ctrl *Controller) UpdateMe(c *gin.Context) {
	claims, ok := requireClaims(c)
	if !ok {
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeInvalidRequest(c, "Invalid profile payload")
		return
	}

	user, err := ctrl.userService.UpdateProfile(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *Controller) UpdateRole(c *gin.Context) {
	if _, ok := requireRole(c, RoleOwner, RoleAdmin); !ok {
		return
	}

	var req struct {
		Role Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeInvalidRequest(c, "Role is required")
		return
	}

	user, err := ctrl.userService.UpdateRole(c.Request.Context(), c.Param("id"), req.Role)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *Controller) UpdateStatus(c *gin.Context) {
	if _, ok := requireRole(c, RoleOwner, RoleAdmin); !ok {
		return
	}

	var req struct {
		Status Status `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeInvalidRequest(c, "Status is required")
		return
	}

	user, err := ctrl.userService.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		writeUserError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (ctrl *Controller) Delete(c *gin.Context) {
	if _, ok := requireRole(c, RoleOwner, RoleAdmin); !ok {
		return
	}

	if err := ctrl.userService.DeleteUser(c.Request.Context(), c.Param("id")); err != nil {
		writeUserError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func requireClaims(c *gin.Context) (*TokenClaims, bool) {
	val, exists := c.Get(session.ContextUserKey)
	claims, ok := val.(*TokenClaims)
	if !exists || !ok || claims == nil {
		writeUnauthorized(c, "JWT_REQUIRED", "Bearer token authentication is required")
		return nil, false
	}
	return claims, true
}

func requireRole(c *gin.Context, roles ...Role) (*TokenClaims, bool) {
	claims, ok := requireClaims(c)
	if !ok {
		return nil, false
	}
	if !hasAnyRole(claims, roles...) {
		writeError(c, "FORBIDDEN", "Owner or admin role required", http.StatusForbidden)
		return nil, false
	}
	return claims, true
}

func hasAnyRole(claims *TokenClaims, roles ...Role) bool {
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
	case errors.Is(err, apperrors.ErrUserNotFound):
		writeError(c, "USER_NOT_FOUND", "User was not found", http.StatusNotFound)
	case errors.Is(err, apperrors.ErrEmailAlreadyExists):
		writeError(c, "EMAIL_ALREADY_EXISTS", "Email address is already registered", http.StatusConflict)
	case errors.Is(err, apperrors.ErrPasswordTooWeak):
		writeError(c, "PASSWORD_TOO_WEAK", "Password must be at least 8 characters", http.StatusBadRequest)
	case errors.Is(err, apperrors.ErrForbidden):
		writeError(c, "INVALID_VALUE", "Invalid role or status", http.StatusBadRequest)
	default:
		writeError(c, "INTERNAL_ERROR", "User service failure", http.StatusInternalServerError)
	}
}

func writeInvalidRequest(c *gin.Context, message string) {
	writeError(c, "INVALID_REQUEST", message, http.StatusBadRequest)
}

func writeUnauthorized(c *gin.Context, code, message string) {
	writeError(c, code, message, http.StatusUnauthorized)
}

func writeError(c *gin.Context, code, message string, statusCode int) {
	c.JSON(statusCode, gin.H{
		"code":    code,
		"message": message,
	})
}
