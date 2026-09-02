package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/internal/transport/http/middleware"
)

type mockSMSSvc struct {
	sendFn func(ctx context.Context, req *domain.SendSMSRequest) (*domain.SMSMessage, error)
}

func (m *mockSMSSvc) Send(ctx context.Context, req *domain.SendSMSRequest) (*domain.SMSMessage, error) {
	if m.sendFn != nil {
		return m.sendFn(ctx, req)
	}
	return nil, nil
}

func (m *mockSMSSvc) GetBySID(ctx context.Context, accountID, sid string) (*domain.SMSMessage, error) {
	return nil, domain.ErrSMSNotFound
}

func (m *mockSMSSvc) HandleDeliveryReport(ctx context.Context, carrierMsgID string, status domain.SMSStatus, errCode, errMsg string) error {
	return nil
}

func setupTestRouter(smsSvc domain.SMSService, mockAcc *domain.Account) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	ctrl := NewSMSController(smsSvc)

	// Auth stub
	r.Use(func(c *gin.Context) {
		if mockAcc != nil {
			c.Set(middleware.ContextAccountKey, mockAcc)
		}
		c.Next()
	})

	r.POST("/v1/sms/send", ctrl.SendSMS)
	r.GET("/v1/sms/:sid", ctrl.GetSMS)

	return r
}

func TestSMSController_SendSMS_Success(t *testing.T) {
	mockAcc := &domain.Account{
		ID:     "acc-123",
		SID:    "AC1234567890",
		Status: domain.AccountStatusActive,
	}

	mockSvc := &mockSMSSvc{
		sendFn: func(ctx context.Context, req *domain.SendSMSRequest) (*domain.SMSMessage, error) {
			return &domain.SMSMessage{
				ID:        "msg-1",
				SID:       "SM999999",
				AccountID: req.AccountID,
				From:      req.From,
				To:        req.To,
				Body:      req.Body,
				Status:    domain.SMSStatusSent,
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

	var resp domain.SMSMessage
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.SID != "SM999999" {
		t.Errorf("expected SID SM999999, got %s", resp.SID)
	}
}

func TestSMSController_SendSMS_InsufficientFunds(t *testing.T) {
	mockAcc := &domain.Account{
		ID:     "acc-123",
		SID:    "AC1234567890",
		Status: domain.AccountStatusActive,
	}

	mockSvc := &mockSMSSvc{
		sendFn: func(ctx context.Context, req *domain.SendSMSRequest) (*domain.SMSMessage, error) {
			return nil, domain.ErrInsufficientFunds
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

