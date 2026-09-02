package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// CallRepo implements domain.CallRepository with MongoDB.
type CallRepo struct {
	calls *mongo.Collection
}

// NewCallRepo creates a new CallRepo instance.
func NewCallRepo(db *mongo.Database) *CallRepo {
	if db == nil {
		return &CallRepo{}
	}

	return &CallRepo{calls: db.Collection(callsCollection)}
}

// Create stores a new call session record.
func (r *CallRepo) Create(ctx context.Context, call *domain.CallSession) error {
	if r.calls == nil {
		return ErrDatabaseUnavailable
	}

	now := time.Now().UTC()
	if call.CreatedAt.IsZero() {
		call.CreatedAt = now
	}
	call.UpdatedAt = now

	if _, err := r.calls.InsertOne(ctx, call); err != nil {
		return fmt.Errorf("failed to insert call session: %w", err)
	}

	return nil
}

// GetByID fetches a call session by UUID.
func (r *CallRepo) GetByID(ctx context.Context, id string) (*domain.CallSession, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

// GetBySID fetches a call session by public SID (CAxxxxxxxx).
func (r *CallRepo) GetBySID(ctx context.Context, sid string) (*domain.CallSession, error) {
	return r.findOne(ctx, bson.M{"sid": sid})
}

// UpdateStatus updates the state of a call session.
func (r *CallRepo) UpdateStatus(ctx context.Context, sid string, status domain.CallStatus) error {
	if r.calls == nil {
		return ErrDatabaseUnavailable
	}

	res, err := r.calls.UpdateOne(
		ctx,
		bson.M{"sid": sid},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now().UTC()}},
	)
	if err != nil {
		return fmt.Errorf("failed to update call status: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrCallNotFound
	}

	return nil
}

// UpdateDurationAndPrice finalizes call duration, billing cost, and end time.
func (r *CallRepo) UpdateDurationAndPrice(ctx context.Context, sid string, duration int, price int64, endTime time.Time) error {
	if r.calls == nil {
		return ErrDatabaseUnavailable
	}

	res, err := r.calls.UpdateOne(
		ctx,
		bson.M{"sid": sid},
		bson.M{"$set": bson.M{
			"duration_seconds": duration,
			"price":            price,
			"end_time":         endTime,
			"status":           domain.CallStatusCompleted,
			"updated_at":       time.Now().UTC(),
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update call duration and price: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrCallNotFound
	}

	return nil
}

func (r *CallRepo) findOne(ctx context.Context, filter any) (*domain.CallSession, error) {
	if r.calls == nil {
		return nil, ErrDatabaseUnavailable
	}

	call := &domain.CallSession{}
	if err := r.calls.FindOne(ctx, filter).Decode(call); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrCallNotFound
		}
		return nil, fmt.Errorf("failed to query call session: %w", err)
	}

	return call, nil
}
