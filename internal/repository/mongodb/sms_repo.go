package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SMSRepo implements domain.SMSRepository using MongoDB.
type SMSRepo struct {
	messages *mongo.Collection
}

// NewSMSRepo creates a new SMSRepo instance.
func NewSMSRepo(db *mongo.Database) *SMSRepo {
	if db == nil {
		return &SMSRepo{}
	}

	return &SMSRepo{messages: db.Collection(smsCollection)}
}

// Create inserts a new SMS message record.
func (r *SMSRepo) Create(ctx context.Context, msg *domain.SMSMessage) error {
	if r.messages == nil {
		return ErrDatabaseUnavailable
	}

	now := time.Now().UTC()
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = now
	}
	msg.UpdatedAt = now

	if _, err := r.messages.InsertOne(ctx, msg); err != nil {
		return fmt.Errorf("failed to insert sms message: %w", err)
	}

	return nil
}

// GetByID retrieves an SMS message by UUID.
func (r *SMSRepo) GetByID(ctx context.Context, id string) (*domain.SMSMessage, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

// GetBySID retrieves an SMS message by public SID (SMxxxxxxxx).
func (r *SMSRepo) GetBySID(ctx context.Context, sid string) (*domain.SMSMessage, error) {
	return r.findOne(ctx, bson.M{"sid": sid})
}

// GetByCarrierID retrieves an SMS message by upstream carrier identifier.
func (r *SMSRepo) GetByCarrierID(ctx context.Context, carrierID string) (*domain.SMSMessage, error) {
	return r.findOne(ctx, bson.M{"carrier_id": carrierID})
}

// UpdateStatus updates the delivery status and optional error details of an SMS.
func (r *SMSRepo) UpdateStatus(ctx context.Context, sid string, status domain.SMSStatus, errCode, errMsg string) error {
	if r.messages == nil {
		return ErrDatabaseUnavailable
	}

	res, err := r.messages.UpdateOne(
		ctx,
		bson.M{"sid": sid},
		bson.M{"$set": bson.M{
			"status":        status,
			"error_code":    errCode,
			"error_message": errMsg,
			"updated_at":    time.Now().UTC(),
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update sms status: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrSMSNotFound
	}

	return nil
}

// UpdateCarrierID stores the upstream carrier/Jasmin message identifier.
func (r *SMSRepo) UpdateCarrierID(ctx context.Context, sid, carrierID string) error {
	if r.messages == nil {
		return ErrDatabaseUnavailable
	}

	res, err := r.messages.UpdateOne(
		ctx,
		bson.M{"sid": sid},
		bson.M{"$set": bson.M{"carrier_id": carrierID, "updated_at": time.Now().UTC()}},
	)
	if err != nil {
		return fmt.Errorf("failed to update sms carrier id: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrSMSNotFound
	}

	return nil
}

// ListByAccount fetches paginated SMS records for an account.
func (r *SMSRepo) ListByAccount(ctx context.Context, accountID string, limit, offset int) ([]*domain.SMSMessage, error) {
	if r.messages == nil {
		return nil, ErrDatabaseUnavailable
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.messages.Find(ctx, bson.M{"account_id": accountID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list sms messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*domain.SMSMessage
	for cursor.Next(ctx) {
		msg := &domain.SMSMessage{}
		if err := cursor.Decode(msg); err != nil {
			return nil, fmt.Errorf("failed to decode sms message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, cursor.Err()
}

func (r *SMSRepo) findOne(ctx context.Context, filter any) (*domain.SMSMessage, error) {
	if r.messages == nil {
		return nil, ErrDatabaseUnavailable
	}

	msg := &domain.SMSMessage{}
	if err := r.messages.FindOne(ctx, filter).Decode(msg); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrSMSNotFound
		}
		return nil, fmt.Errorf("failed to query sms message: %w", err)
	}

	return msg, nil
}
