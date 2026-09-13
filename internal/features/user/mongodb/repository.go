package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/features/user"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const usersCollection = "users"

type repository struct {
	users *mongo.Collection
}

func NewRepository(db *mongo.Database) user.Repository {
	if db == nil {
		return &repository{}
	}

	return &repository{users: db.Collection(usersCollection)}
}

func (r *repository) Create(ctx context.Context, usr *user.User) error {
	if r.users == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	usr.Email = normalizeEmail(usr.Email)
	now := time.Now().UTC()
	if usr.CreatedAt.IsZero() {
		usr.CreatedAt = now
	}
	usr.UpdatedAt = now

	if _, err := r.users.InsertOne(ctx, userModelFromDomain(usr)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, id string) (*user.User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	return r.findOne(ctx, bson.M{"email": normalizeEmail(email)})
}

func (r *repository) ListByAccount(ctx context.Context, accountID string) ([]*user.User, error) {
	if r.users == nil {
		return nil, apperrors.ErrDatabaseUnavailable
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.users.Find(ctx, bson.M{"account_id": accountID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*user.User
	for cursor.Next(ctx) {
		model := &userModel{}
		if err := cursor.Decode(model); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, model.toDomain())
	}

	return users, cursor.Err()
}

func (r *repository) Update(ctx context.Context, usr *user.User) error {
	if r.users == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	usr.Email = normalizeEmail(usr.Email)
	usr.UpdatedAt = time.Now().UTC()
	res, err := r.users.UpdateOne(
		ctx,
		bson.M{"_id": usr.ID},
		bson.M{"$set": bson.M{
			"account_id":    usr.AccountID,
			"email":         usr.Email,
			"password_hash": usr.PasswordHash,
			"first_name":    usr.FirstName,
			"last_name":     usr.LastName,
			"role":          usr.Role,
			"status":        usr.Status,
			"last_login_at": usr.LastLoginAt,
			"updated_at":    usr.UpdatedAt,
		}},
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

func (r *repository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return r.updateFields(ctx, userID, bson.M{"password_hash": passwordHash})
}

func (r *repository) UpdateLastLogin(ctx context.Context, userID string, loginTime time.Time) error {
	return r.updateFields(ctx, userID, bson.M{"last_login_at": loginTime})
}

func (r *repository) Delete(ctx context.Context, id string) error {
	if r.users == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	res, err := r.users.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if res.DeletedCount == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

func (r *repository) findOne(ctx context.Context, filter any) (*user.User, error) {
	if r.users == nil {
		return nil, apperrors.ErrDatabaseUnavailable
	}

	model := &userModel{}
	if err := r.users.FindOne(ctx, filter).Decode(model); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return model.toDomain(), nil
}

func (r *repository) updateFields(ctx context.Context, userID string, fields bson.M) error {
	if r.users == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	fields["updated_at"] = time.Now().UTC()
	res, err := r.users.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": fields})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
