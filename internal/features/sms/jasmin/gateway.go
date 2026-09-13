package jasmin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/features/sms"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
)

// JasminGateway dispatches outbound SMS messages through Jasmin's HTTP API.
type JasminGateway struct {
	baseURL    string
	username   string
	password   string
	dlrURL     string
	httpClient *http.Client
}

var jasminSuccessPattern = regexp.MustCompile(`(?i)^Success\s+"([^"]+)"`)

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
func (g *JasminGateway) Dispatch(ctx context.Context, msg *sms.Message) (*sms.DispatchResult, error) {
	if g.baseURL == "" || g.username == "" || g.password == "" {
		return nil, fmt.Errorf("%w: jasmin gateway is not configured", apperrors.ErrCarrierRejected)
	}

	endpoint, err := url.Parse(g.baseURL + "/send")
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
		return nil, fmt.Errorf("%w: %v", apperrors.ErrCarrierTimeout, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := strings.TrimSpace(string(bodyBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if body == "" {
			body = resp.Status
		}
		return &sms.DispatchResult{
			Status:       sms.StatusFailed,
			ErrorCode:    "JASMIN_REJECTED",
			ErrorMessage: body,
		}, nil
	}

	carrierMessageID := body
	if matches := jasminSuccessPattern.FindStringSubmatch(body); len(matches) == 2 {
		carrierMessageID = matches[1]
	}

	return &sms.DispatchResult{
		CarrierMessageID: carrierMessageID,
		Status:           sms.StatusSent,
	}, nil
}
