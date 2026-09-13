package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/features/billing"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

var e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// Service coordinates SMS business logic, billing deductions, and carrier routing.
type Service struct {
	smsRepo        Repository
	billingService billing.Service
	gateway        Gateway
	cfg            *config.BillingConfig
	logger         *logger.Logger
	httpClient     *http.Client
	workerWg       sync.WaitGroup
	callbackChan   chan *Message
	closeOnce      sync.Once
	closed         chan struct{}
}

// NewService constructs a new enterprise SMS business logic service.
func NewService(
	smsRepo Repository,
	billingService billing.Service,
	gateway Gateway,
	cfg *config.BillingConfig,
	log *logger.Logger,
) *Service {
	if log == nil {
		log = logger.Default()
	}

	s := &Service{
		smsRepo:        smsRepo,
		billingService: billingService,
		gateway:        gateway,
		cfg:            cfg,
		logger:         log,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		callbackChan: make(chan *Message, 500),
		closed:       make(chan struct{}),
	}

	// Start 5 concurrent webhook callback worker routines
	for i := 0; i < 5; i++ {
		s.workerWg.Add(1)
		go s.webhookWorker(i)
	}

	return s
}

// Send validates, bills, stores, and dispatches an SMS message.
func (s *Service) Send(ctx context.Context, req *SendRequest) (*Message, error) {
	// Validate phone numbers (E.164)
	if !e164Regex.MatchString(req.From) || !e164Regex.MatchString(req.To) {
		return nil, apperrors.ErrInvalidPhoneNumber
	}

	if strings.TrimSpace(req.Body) == "" {
		return nil, apperrors.ErrInvalidSMSBody
	}

	// Calculate segments and total cost
	segments := calculateSegments(req.Body)
	totalCost := int64(segments) * s.cfg.DefaultRatePerSMS

	smsSID := fmt.Sprintf("SM%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:32])
	smsID := uuid.New().String()

	// 1. Atomically deduct balance
	desc := fmt.Sprintf("Outbound SMS to %s (%d segment/s)", req.To, segments)
	_, err := s.billingService.DeductBalance(ctx, req.AccountID, totalCost, smsSID, desc)
	if err != nil {
		return nil, err
	}

	msg := &Message{
		ID:          smsID,
		SID:         smsSID,
		AccountID:   req.AccountID,
		AccountSID:  req.AccountSID,
		From:        req.From,
		To:          req.To,
		Body:        req.Body,
		NumSegments: segments,
		Direction:   DirectionOutbound,
		Status:      StatusQueued,
		Price:       totalCost,
		PriceUnit:   s.cfg.Currency,
		CallbackURL: req.CallbackURL,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	// 2. Persist initial queued state
	if err := s.smsRepo.Create(ctx, msg); err != nil {
		// Refund if DB persistence fails
		_, _ = s.billingService.RefundBalance(ctx, req.AccountID, totalCost, smsSID, "Refund: DB write failure")
		return nil, fmt.Errorf("failed to save sms message: %w", err)
	}

	// 3. Dispatch to upstream carrier
	dispatchResult, err := s.gateway.Dispatch(ctx, msg)
	if err != nil {
		// Immediate network/gateway failure: refund and mark failed
		msg.Status = StatusFailed
		msg.ErrorCode = "30001"
		msg.ErrorMessage = err.Error()
		_ = s.smsRepo.UpdateStatus(ctx, msg.SID, msg.Status, msg.ErrorCode, msg.ErrorMessage)
		_, _ = s.billingService.RefundBalance(ctx, req.AccountID, totalCost, smsSID, "Refund: Gateway dispatch failure")
		return msg, nil
	}

	// 4. Update status according to carrier dispatch result
	msg.Status = dispatchResult.Status
	msg.CarrierID = dispatchResult.CarrierMessageID
	msg.ErrorCode = dispatchResult.ErrorCode
	msg.ErrorMessage = dispatchResult.ErrorMessage
	if msg.Status == StatusFailed {
		_, _ = s.billingService.RefundBalance(ctx, req.AccountID, totalCost, smsSID, "Refund: Carrier rejected")
	}

	_ = s.smsRepo.UpdateStatus(ctx, msg.SID, msg.Status, msg.ErrorCode, msg.ErrorMessage)
	if msg.CarrierID != "" {
		_ = s.smsRepo.UpdateCarrierID(ctx, msg.SID, msg.CarrierID)
	}

	// 5. Enqueue status callback webhook if configured
	if msg.CallbackURL != "" {
		select {
		case s.callbackChan <- msg:
		default:
			s.logger.Warn("callback queue is full, dropping callback", "sms_sid", msg.SID)
		}
	}

	return msg, nil
}

// GetBySID retrieves an SMS message ensuring tenant isolation.
func (s *Service) GetBySID(ctx context.Context, accountID, sid string) (*Message, error) {
	msg, err := s.smsRepo.GetBySID(ctx, sid)
	if err != nil {
		return nil, err
	}

	if msg.AccountID != accountID {
		return nil, apperrors.ErrSMSNotFound
	}

	return msg, nil
}

// HandleDeliveryReport processes asynchronous Delivery Receipts (DLR) from upstream carriers.
func (s *Service) HandleDeliveryReport(ctx context.Context, carrierMsgID string, status Status, errCode, errMsg string) error {
	s.logger.Info("received carrier DLR", "carrier_msg_id", carrierMsgID, "status", status)

	msg, err := s.smsRepo.GetByCarrierID(ctx, carrierMsgID)
	if err != nil {
		return err
	}

	if err := s.smsRepo.UpdateStatus(ctx, msg.SID, status, errCode, errMsg); err != nil {
		return err
	}

	msg.Status = status
	msg.ErrorCode = errCode
	msg.ErrorMessage = errMsg
	msg.UpdatedAt = time.Now().UTC()

	if msg.CallbackURL != "" {
		select {
		case s.callbackChan <- msg:
		default:
			s.logger.Warn("callback queue is full, dropping delivery report callback", "sms_sid", msg.SID)
		}
	}

	return nil
}

// Close initiates graceful worker pool shutdown.
func (s *Service) Close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		close(s.callbackChan)
		s.workerWg.Wait()
	})
}

func (s *Service) webhookWorker(workerID int) {
	defer s.workerWg.Done()

	for msg := range s.callbackChan {
		if msg.CallbackURL == "" {
			continue
		}

		payload, err := json.Marshal(msg)
		if err != nil {
			continue
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, msg.CallbackURL, bytes.NewReader(payload))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Callio-CPaaS-Webhook/1.0")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			s.logger.Warn("failed to deliver status callback", "worker", workerID, "sms_sid", msg.SID, "url", msg.CallbackURL, "error", err)
			continue
		}
		_ = resp.Body.Close()
	}
}

// calculateSegments computes GSM-7 / Unicode SMS segment count.
func calculateSegments(body string) int {
	isUnicode := false
	for _, r := range body {
		if r > 127 {
			isUnicode = true
			break
		}
	}

	charCount := utf8.RuneCountInString(body)
	if charCount == 0 {
		return 1
	}

	if isUnicode {
		if charCount <= 70 {
			return 1
		}
		return (charCount + 66) / 67 // Concatenated Unicode segments: 67 chars each
	}

	if charCount <= 160 {
		return 1
	}
	return (charCount + 152) / 153 // Concatenated GSM-7 segments: 153 chars each
}
