package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const (
	AccountsCollection     = "accounts"
	TransactionsCollection = "transactions"
)

// AccountRepo implements Repository via MongoDB.
type AccountRepo struct {
	accounts     *mongo.Collection
	transactions *mongo.Collection
}

// NewAccountRepo creates a new AccountRepo instance.
func NewAccountRepo(db *mongo.Database) *AccountRepo {
	if db == nil {
		return &AccountRepo{}
	}

	return &AccountRepo{
		accounts:     db.Collection(AccountsCollection),
		transactions: db.Collection(TransactionsCollection),
	}
}

func NewRepository(db *mongo.Database) account.Repository {
	return NewAccountRepo(db)
}

// GetByID retrieves an account by internal UUID.
func (r *AccountRepo) GetByID(ctx context.Context, id string) (*account.Account, error) {
	return r.findOne(ctx, bson.M{"_id": id})
}

// GetBySID retrieves an account by public SID (ACxxxxxxxx).
func (r *AccountRepo) GetBySID(ctx context.Context, sid string) (*account.Account, error) {
	return r.findOne(ctx, bson.M{"sid": sid})
}

// Create inserts a new account document into MongoDB.
func (r *AccountRepo) Create(ctx context.Context, acc *account.Account) error {
	if r.accounts == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	now := time.Now().UTC()
	if acc.CreatedAt.IsZero() {
		acc.CreatedAt = now
	}
	acc.UpdatedAt = now

	if _, err := r.accounts.InsertOne(ctx, accountModelFromDomain(acc)); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return apperrors.ErrEmailAlreadyExists
		}
		return fmt.Errorf("failed to insert account: %w", err)
	}

	return nil
}

// UpdateCredentials rotates the public Account SID and secret auth token.
func (r *AccountRepo) UpdateCredentials(ctx context.Context, accountID, sid, authToken string) error {
	if r.accounts == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	res, err := r.accounts.UpdateOne(
		ctx,
		bson.M{"_id": accountID},
		bson.M{"$set": bson.M{
			"sid":        sid,
			"auth_token": authToken,
			"updated_at": time.Now().UTC(),
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to update account credentials: %w", err)
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrAccountNotFound
	}

	return nil
}

// UpdateBalance updates the account balance and updated_at timestamp.
func (r *AccountRepo) UpdateBalance(ctx context.Context, accountID string, newBalance int64) error {
	if r.accounts == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	res, err := r.accounts.UpdateOne(
		ctx,
		bson.M{"_id": accountID},
		bson.M{"$set": bson.M{"balance": newBalance, "updated_at": time.Now().UTC()}},
	)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}
	if res.MatchedCount == 0 {
		return apperrors.ErrAccountNotFound
	}

	return nil
}

// RecordTransaction writes an immutable ledger entry.
func (r *AccountRepo) RecordTransaction(ctx context.Context, tx *account.Transaction) error {
	if r.transactions == nil {
		return apperrors.ErrDatabaseUnavailable
	}

	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now().UTC()
	}

	if _, err := r.transactions.InsertOne(ctx, transactionModelFromDomain(tx)); err != nil {
		return fmt.Errorf("failed to insert transaction ledger entry: %w", err)
	}

	return nil
}

func (r *AccountRepo) findOne(ctx context.Context, filter any) (*account.Account, error) {
	if r.accounts == nil {
		return nil, apperrors.ErrDatabaseUnavailable
	}

	var model accountModel
	if err := r.accounts.FindOne(ctx, filter).Decode(&model); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrAccountNotFound
		}
		return nil, fmt.Errorf("failed to query account: %w", err)
	}

	return model.toDomain(), nil
}
