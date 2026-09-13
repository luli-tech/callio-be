package sms

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	accountFeature "github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/internal/shared/session"
	"github.com/luli-tech/twilio-Boss/internal/transport/httpapi"
)

// SMSController handles HTTP requests for SMS messaging.
type SMSController struct {
	smsService ServiceContract
}

// NewSMSController constructs a new SMSController.
func NewSMSController(smsService ServiceContract) *SMSController {
	return &SMSController{
		smsService: smsService,
	}
}

// SendSMSRequestPayload represents the incoming JSON / form body for POST /v1/sms/send.
type SendSMSRequestPayload struct {
	From        string `json:"from" form:"From" binding:"required"`
	To          string `json:"to" form:"To" binding:"required"`
	Body        string `json:"body" form:"Body" binding:"required"`
	CallbackURL string `json:"callback_url" form:"CallbackUrl"`
}

// SendSMS handles POST /v1/sms/send.
func (ctrl *SMSController) SendSMS(c *gin.Context) {
	acc, ok := requireAccount(c)
	if !ok {
		return
	}

	var req SendSMSRequestPayload
	if err := c.ShouldBind(&req); err != nil {
		httpapi.InvalidRequest(c, "Missing or invalid required fields: 'from', 'to', and 'body' are required")
		return
	}

	domainReq := &SendRequest{
		AccountID:   acc.ID,
		AccountSID:  acc.SID,
		From:        req.From,
		To:          req.To,
		Body:        req.Body,
		CallbackURL: req.CallbackURL,
	}

	msg, err := ctrl.smsService.Send(c.Request.Context(), domainReq)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrInsufficientFunds):
			httpapi.WriteError(c, httpapi.Error{"INSUFFICIENT_FUNDS", "Account balance is too low to send this message", http.StatusPaymentRequired})
		case errors.Is(err, apperrors.ErrInvalidPhoneNumber):
			httpapi.WriteError(c, httpapi.Error{"INVALID_PHONE_NUMBER", "Phone numbers must be in valid E.164 format (e.g. +1234567890)", http.StatusBadRequest})
		case errors.Is(err, apperrors.ErrInvalidSMSBody):
			httpapi.WriteError(c, httpapi.Error{"INVALID_BODY", "Message body cannot be empty", http.StatusBadRequest})
		case errors.Is(err, apperrors.ErrCarrierTimeout):
			httpapi.WriteError(c, httpapi.Error{"CARRIER_TIMEOUT", "Upstream telecom gateway timed out", http.StatusGatewayTimeout})
		default:
			httpapi.InternalError(c, "An error occurred while processing your message")
		}
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// DeliveryReport handles carrier DLR callbacks for POST /v1/sms/dlr.
func (ctrl *SMSController) DeliveryReport(c *gin.Context) {
	var req struct {
		CarrierID    string `json:"id" form:"id"`
		MessageID    string `json:"message_id" form:"message_id"`
		Status       Status `json:"status" form:"status"`
		MessageState Status `json:"message_status" form:"message_status"`
		ErrorCode    string `json:"error_code" form:"error_code"`
		ErrorMessage string `json:"error_message" form:"error_message"`
	}
	if err := c.ShouldBind(&req); err != nil {
		httpapi.InvalidRequest(c, "Invalid delivery report payload")
		return
	}

	carrierID := firstNonEmpty(req.CarrierID, req.MessageID)
	status := req.Status
	if status == "" {
		status = req.MessageState
	}
	if carrierID == "" || status == "" {
		httpapi.InvalidRequest(c, "Carrier message id and status are required")
		return
	}

	if err := ctrl.smsService.HandleDeliveryReport(c.Request.Context(), carrierID, normalizeDLRStatus(status), req.ErrorCode, req.ErrorMessage); err != nil {
		if errors.Is(err, apperrors.ErrSMSNotFound) {
			httpapi.NotFound(c, "RESOURCE_NOT_FOUND", "SMS resource was not found")
			return
		}
		httpapi.InternalError(c, "Failed to process delivery report")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// GetSMS handles GET /v1/sms/:sid.
func (ctrl *SMSController) GetSMS(c *gin.Context) {
	acc, ok := requireAccount(c)
	if !ok {
		return
	}

	sid := c.Param("sid")
	msg, err := ctrl.smsService.GetBySID(c.Request.Context(), acc.ID, sid)
	if err != nil {
		if errors.Is(err, apperrors.ErrSMSNotFound) {
			httpapi.NotFound(c, "RESOURCE_NOT_FOUND", "The requested SMS resource was not found")
			return
		}
		httpapi.InternalError(c, "Failed to retrieve SMS resource")
		return
	}

	c.JSON(http.StatusOK, msg)
}

func requireAccount(c *gin.Context) (*accountFeature.Account, bool) {
	val, exists := c.Get(session.ContextAccountKey)
	acc, ok := val.(*accountFeature.Account)
	if !exists || !ok || acc == nil {
		httpapi.Unauthorized(c, "UNAUTHORIZED", "Authentication required")
		return nil, false
	}
	return acc, true
}

func normalizeDLRStatus(status Status) Status {
	switch Status(strings.ToLower(string(status))) {
	case StatusQueued:
		return StatusQueued
	case StatusSending:
		return StatusSending
	case StatusSent:
		return StatusSent
	case StatusDelivered:
		return StatusDelivered
	case StatusUndelivered:
		return StatusUndelivered
	case StatusFailed:
		return StatusFailed
	default:
		return status
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
