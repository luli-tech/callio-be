package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/domain"
)

type mockAccountRepo struct {
	accounts map[string]*domain.Account
}

func (m *mockAccountRepo) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	acc, ok := m.accounts[id]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	return acc, nil
}

func (m *mockAccountRepo) GetBySID(ctx context.Context, sid string) (*domain.Account, error) {
	for _, acc := range m.accounts {
		if acc.SID == sid {
			return acc, nil
		}
	}
	return nil, domain.ErrAccountNotFound
}

func (m *mockAccountRepo) Create(ctx context.Context, account *domain.Account) error {
	m.accounts[account.ID] = account
	return nil
}

func (m *mockAccountRepo) UpdateCredentials(ctx context.Context, accountID, sid, authToken string) error {
	acc, ok := m.accounts[accountID]
	if !ok {
		return domain.ErrAccountNotFound
	}
	acc.SID = sid
	acc.AuthToken = authToken
	return nil
}

func (m *mockAccountRepo) UpdateBalance(ctx context.Context, accountID string, newBalance int64) error {
	return nil
}

func (m *mockAccountRepo) RecordTransaction(ctx context.Context, tx *domain.Transaction) error {
	return nil
}

func TestAuth_BasicAuth_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAccountRepo{
		accounts: map[string]*domain.Account{
			"acc-1": {
				ID:        "acc-1",
				SID:       "AC12345",
				AuthToken: "secret_token_123",
				Status:    domain.AccountStatusActive,
			},
		},
	}

	router := gin.New()
	router.Use(Auth(repo))
	router.GET("/test", func(c *gin.Context) {
		acc := GetAccount(c)
		if acc == nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, gin.H{"account_id": acc.ID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetBasicAuth("AC12345", "secret_token_123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestAuth_BasicAuth_InvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAccountRepo{
		accounts: map[string]*domain.Account{
			"acc-1": {
				ID:        "acc-1",
				SID:       "AC12345",
				AuthToken: "secret_token_123",
				Status:    domain.AccountStatusActive,
			},
		},
	}

	router := gin.New()
	router.Use(Auth(repo))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetBasicAuth("AC12345", "wrong_token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestAuth_SuspendedAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &mockAccountRepo{
		accounts: map[string]*domain.Account{
			"acc-1": {
				ID:        "acc-1",
				SID:       "AC12345",
				AuthToken: "secret_token_123",
				Status:    domain.AccountStatusSuspended,
			},
		},
	}

	router := gin.New()
	router.Use(Auth(repo))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetBasicAuth("AC12345", "secret_token_123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d", w.Code)
	}
}
