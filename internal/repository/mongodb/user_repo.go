package mongodb

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// UserRepo implements domain.UserRepository using MongoDB.
type UserRepo struct {
	users *mongo.Collection
}

// NewUserRepo creates a new UserRepo instance.
func NewUserRepo(db *mongo.Database) *UserRepo {
	if db == nil {
		return &UserRepo{}
	}

	return &UserRepo{users: db.Collection(usersCollection)}
}

// Create inserts a new user document.
func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	if r.users == nil {
		return ErrDatabaseUnavailable
	}

	user.Email = normalizeEmail(user.Email)
	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	if _, err := r.users.InsertOne(ctx, user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by UUID.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

// GetByEmail retrieves a user by normalized email address.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.findOne(ctx, bson.M{"email": normalizeEmail(email)})
}

// ListByAccount returns all users for an account.
func (r *UserRepo) ListByAccount(ctx context.Context, accountID string) ([]*domain.User, error) {
	if r.users == nil {
		return nil, ErrDatabaseUnavailable
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.users.Find(ctx, bson.M{"account_id": accountID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*domain.User
	for cursor.Next(ctx) {
		user := &domain.User{}
		if err := cursor.Decode(user); err != nil {
			return nil, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, user)
	}

	return users, cursor.Err()
}

// Update persists profile, role, and status changes for an existing user.
func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	if r.users == nil {
		return ErrDatabaseUnavailable
	}

	user.Email = normalizeEmail(user.Email)
	user.UpdatedAt = time.Now().UTC()
	res, err := r.users.UpdateOne(
		ctx,
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"account_id":    user.AccountID,
			"email":         user.Email,
			"password_hash": user.PasswordHash,
			"first_name":    user.FirstName,
			"last_name":     user.LastName,
			"role":          user.Role,
			"status":        user.Status,
			"last_login_at": user.LastLoginAt,
			"updated_at":    user.UpdatedAt,
		}},
	)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to update user: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// UpdatePassword updates only the password hash.
func (r *UserRepo) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	return r.updateFields(ctx, userID, bson.M{"password_hash": passwordHash})
}

// UpdateLastLogin records the latest successful login time.
func (r *UserRepo) UpdateLastLogin(ctx context.Context, userID string, loginTime time.Time) error {
	return r.updateFields(ctx, userID, bson.M{"last_login_at": loginTime})
}

// Delete removes a user document.
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	if r.users == nil {
		return ErrDatabaseUnavailable
	}

	res, err := r.users.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if res.DeletedCount == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *UserRepo) findOne(ctx context.Context, filter any) (*domain.User, error) {
	if r.users == nil {
		return nil, ErrDatabaseUnavailable
	}

	user := &domain.User{}
	if err := r.users.FindOne(ctx, filter).Decode(user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return user, nil
}

func (r *UserRepo) updateFields(ctx context.Context, userID string, fields bson.M) error {
	if r.users == nil {
		return ErrDatabaseUnavailable
	}

	fields["updated_at"] = time.Now().UTC()
	res, err := r.users.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": fields})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	if res.MatchedCount == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
