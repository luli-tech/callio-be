package sms

import (
	"context"
	"time"
)

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound-api"
)

type Status string

const (
	StatusQueued      Status = "queued"
	StatusSending     Status = "sending"
	StatusSent        Status = "sent"
	StatusDelivered   Status = "delivered"
	StatusUndelivered Status = "undelivered"
	StatusFailed      Status = "failed"
)

type Message struct {
	ID           string    `json:"id"`
	SID          string    `json:"sid"`
	CarrierID    string    `json:"carrier_id,omitempty"`
	AccountID    string    `json:"account_id"`
	AccountSID   string    `json:"account_sid"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Body         string    `json:"body"`
	NumSegments  int       `json:"num_segments"`
	Direction    Direction `json:"direction"`
	Status       Status    `json:"status"`
	Price        int64     `json:"price"`
	PriceUnit    string    `json:"price_unit"`
	ErrorCode    string    `json:"error_code,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CallbackURL  string    `json:"callback_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SendRequest struct {
	AccountID   string `json:"-"`
	AccountSID  string `json:"-"`
	From        string `json:"from" binding:"required"`
	To          string `json:"to" binding:"required"`
	Body        string `json:"body" binding:"required"`
	CallbackURL string `json:"callback_url"`
}

type DispatchResult struct {
	CarrierMessageID string
	Status           Status
	ErrorCode        string
	ErrorMessage     string
}

type Repository interface {
	Create(ctx context.Context, msg *Message) error
	GetByID(ctx context.Context, id string) (*Message, error)
	GetBySID(ctx context.Context, sid string) (*Message, error)
	GetByCarrierID(ctx context.Context, carrierID string) (*Message, error)
	UpdateStatus(ctx context.Context, sid string, status Status, errCode, errMsg string) error
	UpdateCarrierID(ctx context.Context, sid, carrierID string) error
	ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*Message, error)
}

type Gateway interface {
	Dispatch(ctx context.Context, msg *Message) (*DispatchResult, error)
}

type ServiceContract interface {
	Send(ctx context.Context, req *SendRequest) (*Message, error)
	GetBySID(ctx context.Context, accountID, sid string) (*Message, error)
	HandleDeliveryReport(ctx context.Context, carrierMsgID string, status Status, errCode, errMsg string) error
}
