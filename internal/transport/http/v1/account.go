package v1

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/internal/transport/http/middleware"
)

// AccountController handles account inspection and management.
type AccountController struct {
	accountRepo domain.AccountRepository
	billingSvc  domain.BillingService
}

// NewAccountController constructs a new AccountController.
func NewAccountController(accountRepo domain.AccountRepository, billingSvc domain.BillingService) *AccountController {
	return &AccountController{
		accountRepo: accountRepo,
		billingSvc:  billingSvc,
	}
}

// GetCurrentAccount returns the authenticated account's details and live balance.
func (ctrl *AccountController) GetCurrentAccount(c *gin.Context) {
	acc := middleware.GetAccount(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	liveBalance, err := ctrl.billingSvc.GetBalance(c.Request.Context(), acc.ID)
	if err == nil {
		acc.Balance = liveBalance
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         acc.ID,
		"sid":        acc.SID,
		"name":       acc.Name,
		"email":      acc.Email,
		"balance":    acc.Balance,
		"currency":   acc.Currency,
		"status":     acc.Status,
		"created_at": acc.CreatedAt,
		"updated_at": acc.UpdatedAt,
	})
}

// CreateAccountRequestPayload defines tenant account creation payload.
type CreateAccountRequestPayload struct {
	Name           string `json:"name" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	InitialBalance int64  `json:"initial_balance"` // In micro-units
}

// CreateAccount registers a new customer tenant account.
func (ctrl *AccountController) CreateAccount(c *gin.Context) {
	var req CreateAccountRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Valid name and email are required",
		})
		return
	}

	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	authToken := hex.EncodeToString(tokenBytes)

	accountSID := fmt.Sprintf("AC%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:32])

	acc := &domain.Account{
		ID:        uuid.New().String(),
		SID:       accountSID,
		AuthToken: authToken,
		Name:      req.Name,
		Email:     req.Email,
		Balance:   req.InitialBalance,
		Currency:  "USD",
		Status:    domain.AccountStatusActive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := ctrl.accountRepo.Create(c.Request.Context(), acc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "ACCOUNT_CREATION_FAILED",
			"message": "Could not register account",
		})
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

