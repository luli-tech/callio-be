package mongodb

import (
	"time"

	"github.com/luli-tech/twilio-Boss/internal/features/sms"
)

type messageModel struct {
	ID           string        `bson:"_id"`
	SID          string        `bson:"sid"`
	CarrierID    string        `bson:"carrier_id,omitempty"`
	AccountID    string        `bson:"account_id"`
	AccountSID   string        `bson:"account_sid"`
	From         string        `bson:"from"`
	To           string        `bson:"to"`
	Body         string        `bson:"body"`
	NumSegments  int           `bson:"num_segments"`
	Direction    sms.Direction `bson:"direction"`
	Status       sms.Status    `bson:"status"`
	Price        int64         `bson:"price"`
	PriceUnit    string        `bson:"price_unit"`
	ErrorCode    string        `bson:"error_code,omitempty"`
	ErrorMessage string        `bson:"error_message,omitempty"`
	CallbackURL  string        `bson:"callback_url,omitempty"`
	CreatedAt    time.Time     `bson:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at"`
}

func messageModelFromDomain(msg *sms.Message) *messageModel {
	if msg == nil {
		return nil
	}
	return &messageModel{
		ID:           msg.ID,
		SID:          msg.SID,
		CarrierID:    msg.CarrierID,
		AccountID:    msg.AccountID,
		AccountSID:   msg.AccountSID,
		From:         msg.From,
		To:           msg.To,
		Body:         msg.Body,
		NumSegments:  msg.NumSegments,
		Direction:    msg.Direction,
		Status:       msg.Status,
		Price:        msg.Price,
		PriceUnit:    msg.PriceUnit,
		ErrorCode:    msg.ErrorCode,
		ErrorMessage: msg.ErrorMessage,
		CallbackURL:  msg.CallbackURL,
		CreatedAt:    msg.CreatedAt,
		UpdatedAt:    msg.UpdatedAt,
	}
}

func (m *messageModel) toDomain() *sms.Message {
	if m == nil {
		return nil
	}
	return &sms.Message{
		ID:           m.ID,
		SID:          m.SID,
		CarrierID:    m.CarrierID,
		AccountID:    m.AccountID,
		AccountSID:   m.AccountSID,
		From:         m.From,
		To:           m.To,
		Body:         m.Body,
		NumSegments:  m.NumSegments,
		Direction:    m.Direction,
		Status:       m.Status,
		Price:        m.Price,
		PriceUnit:    m.PriceUnit,
		ErrorCode:    m.ErrorCode,
		ErrorMessage: m.ErrorMessage,
		CallbackURL:  m.CallbackURL,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
