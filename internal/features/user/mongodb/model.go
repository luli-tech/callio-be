package mongodb

import (
	"time"

	"github.com/luli-tech/twilio-Boss/internal/features/user"
)

type userModel struct {
	ID           string      `bson:"_id"`
	AccountID    string      `bson:"account_id"`
	Email        string      `bson:"email"`
	PasswordHash string      `bson:"password_hash"`
	FirstName    string      `bson:"first_name"`
	LastName     string      `bson:"last_name"`
	Role         user.Role   `bson:"role"`
	Status       user.Status `bson:"status"`
	LastLoginAt  *time.Time  `bson:"last_login_at,omitempty"`
	CreatedAt    time.Time   `bson:"created_at"`
	UpdatedAt    time.Time   `bson:"updated_at"`
}

func userModelFromDomain(usr *user.User) *userModel {
	if usr == nil {
		return nil
	}
	return &userModel{
		ID:           usr.ID,
		AccountID:    usr.AccountID,
		Email:        usr.Email,
		PasswordHash: usr.PasswordHash,
		FirstName:    usr.FirstName,
		LastName:     usr.LastName,
		Role:         usr.Role,
		Status:       usr.Status,
		LastLoginAt:  usr.LastLoginAt,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
	}
}

func (m *userModel) toDomain() *user.User {
	if m == nil {
		return nil
	}
	return &user.User{
		ID:           m.ID,
		AccountID:    m.AccountID,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		FirstName:    m.FirstName,
		LastName:     m.LastName,
		Role:         m.Role,
		Status:       m.Status,
		LastLoginAt:  m.LastLoginAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}
