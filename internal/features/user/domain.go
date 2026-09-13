package user

import (
	"context"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/shared/session"
)

type Role = session.Role

const (
	RoleOwner     = session.RoleOwner
	RoleAdmin     = session.RoleAdmin
	RoleDeveloper = session.RoleDeveloper
	RoleMember    = session.RoleMember
)

type Status string

const (
	StatusActive    Status = "active"
	StatusSuspended Status = "suspended"
	StatusPending   Status = "pending"
)

type User struct {
	ID           string     `json:"id"`
	AccountID    string     `json:"account_id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Role         Role       `json:"role"`
	Status       Status     `json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (u *User) FullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Email
	}
	if u.FirstName == "" {
		return u.LastName
	}
	if u.LastName == "" {
		return u.FirstName
	}
	return u.FirstName + " " + u.LastName
}

func (u *User) HasRole(roles ...Role) bool {
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	return false
}

type TokenClaims = session.TokenClaims

type CreateRequest struct {
	AccountID string `json:"-"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      Role   `json:"role" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListByAccount(ctx context.Context, accountID string) ([]*User, error)
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	UpdateLastLogin(ctx context.Context, userID string, loginTime time.Time) error
	Delete(ctx context.Context, id string) error
}

type ServiceContract interface {
	CreateUser(ctx context.Context, req *CreateRequest) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsersByAccount(ctx context.Context, accountID string) ([]*User, error)
	UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*User, error)
	UpdateRole(ctx context.Context, userID string, role Role) (*User, error)
	UpdateStatus(ctx context.Context, userID string, status Status) (*User, error)
	DeleteUser(ctx context.Context, userID string) error
}
