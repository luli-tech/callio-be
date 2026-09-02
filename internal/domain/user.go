package domain

import (
	"context"
	"time"
)

// UserRole represents RBAC access tier.
type UserRole string

const (
	RoleOwner     UserRole = "owner"
	RoleAdmin     UserRole = "admin"
	RoleDeveloper UserRole = "developer"
	RoleMember    UserRole = "member"
)

// UserStatus represents user account lifecycle state.
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusPending   UserStatus = "pending"
)

// User represents a tenant portal user entity.
type User struct {
	ID           string     `json:"id" bson:"_id"`
	AccountID    string     `json:"account_id" bson:"account_id"`
	Email        string     `json:"email" bson:"email"`
	PasswordHash string     `json:"-" bson:"password_hash"`
	FirstName    string     `json:"first_name" bson:"first_name"`
	LastName     string     `json:"last_name" bson:"last_name"`
	Role         UserRole   `json:"role" bson:"role"`
	Status       UserStatus `json:"status" bson:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" bson:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" bson:"updated_at"`
}

// FullName returns the combined first and last name.
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

// HasRole returns true if the user's role satisfies any of the required roles.
func (u *User) HasRole(roles ...UserRole) bool {
	for _, r := range roles {
		if u.Role == r {
			return true
		}
	}
	return false
}

// TokenClaims represents decoded claims from a JWT session token.
type TokenClaims struct {
	UserID    string   `json:"user_id"`
	AccountID string   `json:"account_id"`
	Email     string   `json:"email"`
	Role      UserRole `json:"role"`
}

// RegisterRequest defines the payload for new tenant registration.
type RegisterRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

// LoginRequest defines credentials submitted for portal authentication.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest defines token renewal payload.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthResponse returns issued tokens and user info upon authentication.
type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	TokenType    string   `json:"token_type"`
	ExpiresIn    int64    `json:"expires_in"` // in seconds
	User         *User    `json:"user"`
	Account      *Account `json:"account,omitempty"`
}

// CreateUserRequest defines payload to invite/create a member inside a tenant account.
type CreateUserRequest struct {
	AccountID string   `json:"-"`
	Email     string   `json:"email" binding:"required,email"`
	Password  string   `json:"password" binding:"required,min=8"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Role      UserRole `json:"role" binding:"required"`
}

// UpdateProfileRequest defines profile field updates.
type UpdateProfileRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// UserRepository defines storage operations for users.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ListByAccount(ctx context.Context, accountID string) ([]*User, error)
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	UpdateLastLogin(ctx context.Context, userID string, loginTime time.Time) error
	Delete(ctx context.Context, id string) error
}

// UserService defines business logic for managing tenant users and profiles.
type UserService interface {
	CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	ListUsersByAccount(ctx context.Context, accountID string) ([]*User, error)
	UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*User, error)
	UpdateRole(ctx context.Context, userID string, role UserRole) (*User, error)
	UpdateStatus(ctx context.Context, userID string, status UserStatus) (*User, error)
	DeleteUser(ctx context.Context, userID string) error
}

// AuthService coordinates authentication, password verification, token issuance, and credential rotation.
type AuthService interface {
	Register(ctx context.Context, req *RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req *LoginRequest) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*AuthResponse, error)
	ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error)
	RotateAccountCredentials(ctx context.Context, accountID string) (*Account, error)
}
