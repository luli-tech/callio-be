package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/domain"
)

// JasminGateway dispatches outbound SMS messages through Jasmin's HTTP API.
type JasminGateway struct {
	baseURL    string
	username   string
	password   string
	dlrURL     string
	httpClient *http.Client
}

// NewJasminGateway creates a Jasmin HTTP SMS gateway adapter.
func NewJasminGateway(cfg config.SMSGatewayConfig) *JasminGateway {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &JasminGateway{
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		username: cfg.Username,
		password: cfg.Password,
		dlrURL:   cfg.DLRURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Dispatch sends a message via Jasmin and returns its accepted carrier ID.
func (g *JasminGateway) Dispatch(ctx context.Context, msg *domain.SMSMessage) (*domain.SMSDispatchResult, error) {
	if g.baseURL == "" || g.username == "" || g.password == "" {
		return nil, fmt.Errorf("%w: jasmin gateway is not configured", domain.ErrCarrierRejected)
	}

	endpoint, err := url.Parse(g.baseURL + "/secure/send")
	if err != nil {
		return nil, err
	}

	q := endpoint.Query()
	q.Set("username", g.username)
	q.Set("password", g.password)
	q.Set("from", msg.From)
	q.Set("to", msg.To)
	q.Set("content", msg.Body)
	if g.dlrURL != "" {
		q.Set("dlr-url", g.dlrURL)
		q.Set("dlr-level", "3")
		q.Set("dlr-method", http.MethodPost)
	}
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Callio-CPaaS-SMSGateway/1.0")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrCarrierTimeout, err)
	}
	defer resp.Body.Close()

	var jasminResp struct {
		Data  string `json:"data"`
		Error string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&jasminResp)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if jasminResp.Error == "" {
			jasminResp.Error = resp.Status
		}
		return &domain.SMSDispatchResult{
			Status:       domain.SMSStatusFailed,
			ErrorCode:    "JASMIN_REJECTED",
			ErrorMessage: jasminResp.Error,
		}, nil
	}

	return &domain.SMSDispatchResult{
		CarrierMessageID: jasminResp.Data,
		Status:           domain.SMSStatusSent,
	}, nil
}
