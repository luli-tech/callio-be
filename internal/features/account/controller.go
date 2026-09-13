package account

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/shared/session"
	"github.com/luli-tech/twilio-Boss/internal/transport/httpapi"
)

// AccountController handles account inspection and management.
type AccountController struct {
	accountService ServiceContract
}

// NewAccountController constructs a new AccountController.
func NewAccountController(accountService ServiceContract) *AccountController {
	return &AccountController{accountService: accountService}
}

// GetCurrentAccount returns the authenticated account's details and live balance.
func (ctrl *AccountController) GetCurrentAccount(c *gin.Context) {
	acc, ok := requireAccount(c)
	if !ok {
		return
	}

	current, err := ctrl.accountService.GetCurrentAccount(c.Request.Context(), acc.ID)
	if err != nil {
		writeAccountError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         current.ID,
		"sid":        current.SID,
		"name":       current.Name,
		"email":      current.Email,
		"balance":    current.Balance,
		"currency":   current.Currency,
		"status":     current.Status,
		"created_at": current.CreatedAt,
		"updated_at": current.UpdatedAt,
	})
}

// CreateAccount registers a new customer tenant account.
func (ctrl *AccountController) CreateAccount(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpapi.InvalidRequest(c, "Valid name and email are required")
		return
	}

	acc, err := ctrl.accountService.CreateAccount(c.Request.Context(), &req)
	if err != nil {
		writeAccountError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         acc.ID,
		"sid":        acc.SID,
		"auth_token": acc.AuthToken,
		"name":       acc.Name,
		"email":      acc.Email,
		"balance":    acc.Balance,
		"currency":   acc.Currency,
		"status":     acc.Status,
	})
}

func requireAccount(c *gin.Context) (*Account, bool) {
	val, exists := c.Get(session.ContextAccountKey)
	acc, ok := val.(*Account)
	if !exists || !ok || acc == nil {
		httpapi.Unauthorized(c, "UNAUTHORIZED", "Authentication required")
		return nil, false
	}
	return acc, true
}

func writeAccountError(c *gin.Context, err error) {
	httpapi.WriteError(c, httpapi.Error{
		Code:       "ACCOUNT_SERVICE_ERROR",
		Message:    "Account service failure",
		StatusCode: http.StatusInternalServerError,
	})
}
