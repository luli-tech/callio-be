package auth

import (
	"context"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/features/user"
	"github.com/luli-tech/twilio-Boss/internal/shared/session"
)

type RegisterRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type Response struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
	TokenType    string           `json:"token_type"`
	ExpiresIn    int64            `json:"expires_in"`
	User         *user.User       `json:"user"`
	Account      *account.Account `json:"account,omitempty"`
}

type TokenClaims = session.TokenClaims

type ServiceContract interface {
	Register(ctx context.Context, req *RegisterRequest) (*Response, error)
	Login(ctx context.Context, req *LoginRequest) (*Response, error)
	RefreshToken(ctx context.Context, refreshToken string) (*Response, error)
	ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error)
	RotateAccountCredentials(ctx context.Context, accountID string) (*account.Account, error)
}

type AccountStore interface {
	Create(ctx context.Context, account *account.Account) error
	GetByID(ctx context.Context, id string) (*account.Account, error)
	UpdateCredentials(ctx context.Context, accountID, sid, authToken string) error
}

type UserStore interface {
	Create(ctx context.Context, user *user.User) error
	GetByID(ctx context.Context, id string) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	UpdateLastLogin(ctx context.Context, userID string, loginTime time.Time) error
}
