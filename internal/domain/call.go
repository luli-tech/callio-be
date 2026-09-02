package domain

import (
	"context"
	"time"
)

// CallDirection represents the direction of the voice call.
type CallDirection string

const (
	CallDirectionInbound  CallDirection = "inbound"
	CallDirectionOutbound CallDirection = "outbound-api"
)

// CallStatus represents the real-time telephony state of a call session.
type CallStatus string

const (
	CallStatusQueued     CallStatus = "queued"
	CallStatusInitiated  CallStatus = "initiated"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusInProgress CallStatus = "in-progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusBusy       CallStatus = "busy"
	CallStatusFailed     CallStatus = "failed"
	CallStatusNoAnswer   CallStatus = "no-answer"
	CallStatusCanceled   CallStatus = "canceled"
)

// CallSession represents an active or historical voice call entity.
type CallSession struct {
	ID              string        `json:"id" bson:"_id"`
	SID             string        `json:"sid" bson:"sid"` // Twilio-style SID: CAxxxxxxxx
	AccountID       string        `json:"account_id" bson:"account_id"`
	AccountSID      string        `json:"account_sid" bson:"account_sid"`
	From            string        `json:"from" bson:"from"`
	To              string        `json:"to" bson:"to"`
	Direction       CallDirection `json:"direction" bson:"direction"`
	Status          CallStatus    `json:"status" bson:"status"`
	DurationSeconds int           `json:"duration_seconds" bson:"duration_seconds"`
	RatePerMinute   int64         `json:"rate_per_minute" bson:"rate_per_minute"` // In micro-units
	Price           int64         `json:"price" bson:"price"`                     // Total billed price in micro-units
	PriceUnit       string        `json:"price_unit" bson:"price_unit"`
	StartTime       *time.Time    `json:"start_time,omitempty" bson:"start_time,omitempty"`
	EndTime         *time.Time    `json:"end_time,omitempty" bson:"end_time,omitempty"`
	RecordingURL    string        `json:"recording_url,omitempty" bson:"recording_url,omitempty"`
	CallbackURL     string        `json:"callback_url,omitempty" bson:"callback_url,omitempty"`
	CreatedAt       time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" bson:"updated_at"`
}

// InitiateCallRequest represents an API request to trigger an outbound phone call.
type InitiateCallRequest struct {
	AccountID   string `json:"-"`
	AccountSID  string `json:"-"`
	From        string `json:"from" binding:"required"`
	To          string `json:"to" binding:"required"`
	CallbackURL string `json:"callback_url"`
	AudioURL    string `json:"audio_url"`
}

// CallRepository defines storage operations for voice call sessions.
type CallRepository interface {
	Create(ctx context.Context, call *CallSession) error
	GetByID(ctx context.Context, id string) (*CallSession, error)
	GetBySID(ctx context.Context, sid string) (*CallSession, error)
	UpdateStatus(ctx context.Context, sid string, status CallStatus) error
	UpdateDurationAndPrice(ctx context.Context, sid string, duration int, price int64, endTime time.Time) error
}

// VoiceEngine defines interface for interacting with FreeSWITCH / Asterisk telephony servers via ESL.
type VoiceEngine interface {
	OriginateCall(ctx context.Context, call *CallSession) (string, error)
	HangupCall(ctx context.Context, callSID string, reason string) error
	PlayAudio(ctx context.Context, callSID string, audioURL string) error
}
