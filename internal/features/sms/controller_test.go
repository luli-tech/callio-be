package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/internal/shared/session"
)

type mockSMSSvc struct {
	sendFn func(ctx context.Context, req *SendRequest) (*Message, error)
}

func (m *mockSMSSvc) Send(ctx context.Context, req *SendRequest) (*Message, error) {
	if m.sendFn != nil {
		return m.sendFn(ctx, req)
	}
	return nil, nil
}

func (m *mockSMSSvc) GetBySID(ctx context.Context, accountID, sid string) (*Message, error) {
	return nil, apperrors.ErrSMSNotFound
}

func (m *mockSMSSvc) HandleDeliveryReport(ctx context.Context, carrierMsgID string, status Status, errCode, errMsg string) error {
	return nil
}

func setupTestRouter(smsSvc ServiceContract, mockAcc *account.Account) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ctrl := NewSMSController(smsSvc)

	// Auth stub
	r.Use(func(c *gin.Context) {
		if mockAcc != nil {
			c.Set(session.ContextAccountKey, mockAcc)
		}
		c.Next()
	})

	r.POST("/v1/sms/send", ctrl.SendSMS)
	r.GET("/v1/sms/:sid", ctrl.GetSMS)

	return r
}

func TestSMSController_SendSMS_Success(t *testing.T) {
	mockAcc := &account.Account{
		ID:     "acc-123",
		SID:    "AC1234567890",
		Status: account.StatusActive,
	}

	mockSvc := &mockSMSSvc{
		sendFn: func(ctx context.Context, req *SendRequest) (*Message, error) {
			return &Message{
				ID:        "msg-1",
				SID:       "SM999999",
				AccountID: req.AccountID,
				From:      req.From,
				To:        req.To,
				Body:      req.Body,
				Status:    StatusSent,
				Price:     7500,
			}, nil
		},
	}

	router := setupTestRouter(mockSvc, mockAcc)

	payload := map[string]string{
		"from": "+12025550100",
		"to":   "+12025550199",
		"body": "Test message payload",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/sms/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp Message
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.SID != "SM999999" {
		t.Errorf("expected SID SM999999, got %s", resp.SID)
	}
}

func TestSMSController_SendSMS_InsufficientFunds(t *testing.T) {
	mockAcc := &account.Account{
		ID:     "acc-123",
		SID:    "AC1234567890",
		Status: account.StatusActive,
	}

	mockSvc := &mockSMSSvc{
		sendFn: func(ctx context.Context, req *SendRequest) (*Message, error) {
			return nil, apperrors.ErrInsufficientFunds
		},
	}

	router := setupTestRouter(mockSvc, mockAcc)

	payload := map[string]string{
		"from": "+12025550100",
		"to":   "+12025550199",
		"body": "Test insufficient balance",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/sms/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status 402 Payment Required, got %d", w.Code)
	}
}
