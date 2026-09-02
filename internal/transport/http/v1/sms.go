package v1

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/internal/transport/http/middleware"
)

// SMSController handles HTTP requests for SMS messaging.
type SMSController struct {
	smsService domain.SMSService
}

// NewSMSController constructs a new SMSController.
func NewSMSController(smsService domain.SMSService) *SMSController {
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
	acc := middleware.GetAccount(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "Authentication required",
		})
		return
	}

	var req SendSMSRequestPayload
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Missing or invalid required fields: 'from', 'to', and 'body' are required",
		})
		return
	}

	domainReq := &domain.SendSMSRequest{
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
		case errors.Is(err, domain.ErrInsufficientFunds):
			c.JSON(http.StatusPaymentRequired, gin.H{
				"code":    "INSUFFICIENT_FUNDS",
				"message": "Account balance is too low to send this message",
			})
		case errors.Is(err, domain.ErrInvalidPhoneNumber):
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "INVALID_PHONE_NUMBER",
				"message": "Phone numbers must be in valid E.164 format (e.g. +1234567890)",
			})
		case errors.Is(err, domain.ErrInvalidSMSBody):
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "INVALID_BODY",
				"message": "Message body cannot be empty",
			})
		case errors.Is(err, domain.ErrCarrierTimeout):
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"code":    "CARRIER_TIMEOUT",
				"message": "Upstream telecom gateway timed out",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "An error occurred while processing your message",
			})
		}
		return
	}

	c.JSON(http.StatusCreated, msg)
}

// DeliveryReport handles carrier DLR callbacks for POST /v1/sms/dlr.
func (ctrl *SMSController) DeliveryReport(c *gin.Context) {
	var req struct {
		CarrierID    string           `json:"id" form:"id"`
		MessageID    string           `json:"message_id" form:"message_id"`
		Status       domain.SMSStatus `json:"status" form:"status"`
		MessageState domain.SMSStatus `json:"message_status" form:"message_status"`
		ErrorCode    string           `json:"error_code" form:"error_code"`
		ErrorMessage string           `json:"error_message" form:"error_message"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Invalid delivery report payload"})
		return
	}

	carrierID := firstNonEmpty(req.CarrierID, req.MessageID)
	status := req.Status
	if status == "" {
		status = req.MessageState
	}
	if carrierID == "" || status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "Carrier message id and status are required"})
		return
	}

	if err := ctrl.smsService.HandleDeliveryReport(c.Request.Context(), carrierID, normalizeDLRStatus(status), req.ErrorCode, req.ErrorMessage); err != nil {
		if errors.Is(err, domain.ErrSMSNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "RESOURCE_NOT_FOUND", "message": "SMS resource was not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "Failed to process delivery report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// GetSMS handles GET /v1/sms/:sid.
func (ctrl *SMSController) GetSMS(c *gin.Context) {
	acc := middleware.GetAccount(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authentication required"})
		return
	}

	sid := c.Param("sid")
	msg, err := ctrl.smsService.GetBySID(c.Request.Context(), acc.ID, sid)
	if err != nil {
		if errors.Is(err, domain.ErrSMSNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "RESOURCE_NOT_FOUND",
				"message": "The requested SMS resource was not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "Failed to retrieve SMS resource",
		})
		return
	}

	c.JSON(http.StatusOK, msg)
}

func normalizeDLRStatus(status domain.SMSStatus) domain.SMSStatus {
	switch domain.SMSStatus(strings.ToLower(string(status))) {
	case domain.SMSStatusQueued:
		return domain.SMSStatusQueued
	case domain.SMSStatusSending:
		return domain.SMSStatusSending
	case domain.SMSStatusSent:
		return domain.SMSStatusSent
	case domain.SMSStatusDelivered:
		return domain.SMSStatusDelivered
	case domain.SMSStatusUndelivered:
		return domain.SMSStatusUndelivered
	case domain.SMSStatusFailed:
		return domain.SMSStatusFailed
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
