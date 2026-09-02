package domain

import (
	"context"
	"time"
)

// SMSDirection indicates if message is inbound or outbound.
type SMSDirection string

const (
	SMSDirectionInbound  SMSDirection = "inbound"
	SMSDirectionOutbound SMSDirection = "outbound-api"
)

// SMSStatus represents the delivery lifecycle states of an SMS.
type SMSStatus string

const (
	SMSStatusQueued      SMSStatus = "queued"
	SMSStatusSending     SMSStatus = "sending"
	SMSStatusSent        SMSStatus = "sent"
	SMSStatusDelivered   SMSStatus = "delivered"
	SMSStatusUndelivered SMSStatus = "undelivered"
	SMSStatusFailed      SMSStatus = "failed"
)

// SMSMessage represents an SMS communication record.
type SMSMessage struct {
	ID           string       `json:"id" bson:"_id"`
	SID          string       `json:"sid" bson:"sid"` // Twilio-style SID: SMxxxxxxxx
	CarrierID    string       `json:"carrier_id,omitempty" bson:"carrier_id,omitempty"`
	AccountID    string       `json:"account_id" bson:"account_id"`
	AccountSID   string       `json:"account_sid" bson:"account_sid"`
	From         string       `json:"from" bson:"from"` // E.164 phone number
	To           string       `json:"to" bson:"to"`     // E.164 phone number
	Body         string       `json:"body" bson:"body"`
	NumSegments  int          `json:"num_segments" bson:"num_segments"`
	Direction    SMSDirection `json:"direction" bson:"direction"`
	Status       SMSStatus    `json:"status" bson:"status"`
	Price        int64        `json:"price" bson:"price"`           // In micro-units
	PriceUnit    string       `json:"price_unit" bson:"price_unit"` // USD
	ErrorCode    string       `json:"error_code,omitempty" bson:"error_code,omitempty"`
	ErrorMessage string       `json:"error_message,omitempty" bson:"error_message,omitempty"`
	CallbackURL  string       `json:"callback_url,omitempty" bson:"callback_url,omitempty"`
	CreatedAt    time.Time    `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at" bson:"updated_at"`
}

// SendSMSRequest is the payload submitted by an API client to dispatch an SMS.
type SendSMSRequest struct {
	AccountID   string `json:"-"`
	AccountSID  string `json:"-"`
	From        string `json:"from" binding:"required"`
	To          string `json:"to" binding:"required"`
	Body        string `json:"body" binding:"required"`
	CallbackURL string `json:"callback_url"`
}

// SMSDispatchResult represents the upstream carrier response.
type SMSDispatchResult struct {
	CarrierMessageID string
	Status           SMSStatus
	ErrorCode        string
	ErrorMessage     string
}

// SMSRepository defines persistence operations for SMS records.
type SMSRepository interface {
	Create(ctx context.Context, msg *SMSMessage) error
	GetByID(ctx context.Context, id string) (*SMSMessage, error)
	GetBySID(ctx context.Context, sid string) (*SMSMessage, error)
	GetByCarrierID(ctx context.Context, carrierID string) (*SMSMessage, error)
	UpdateStatus(ctx context.Context, sid string, status SMSStatus, errCode, errMsg string) error
	UpdateCarrierID(ctx context.Context, sid, carrierID string) error
	ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*SMSMessage, error)
}

// SMSGateway defines the contract for communicating with upstream telecommunication carriers (SMPP/HTTP/Jasmin).
type SMSGateway interface {
	Dispatch(ctx context.Context, msg *SMSMessage) (*SMSDispatchResult, error)
}

// SMSService defines the business workflow orchestration for SMS.
type SMSService interface {
	Send(ctx context.Context, req *SendSMSRequest) (*SMSMessage, error)
	GetBySID(ctx context.Context, accountID, sid string) (*SMSMessage, error)
	HandleDeliveryReport(ctx context.Context, carrierMsgID string, status SMSStatus, errCode, errMsg string) error
}
