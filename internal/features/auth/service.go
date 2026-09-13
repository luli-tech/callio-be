package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/config"
	accountFeature "github.com/luli-tech/twilio-Boss/internal/features/account"
	userFeature "github.com/luli-tech/twilio-Boss/internal/features/user"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"github.com/luli-tech/twilio-Boss/internal/shared/session"
	"github.com/luli-tech/twilio-Boss/pkg/credentials"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	accountStore AccountStore
	userStore    UserStore
	jwtConfig    config.JWTConfig
}

type jwtClaims struct {
	UserID    string       `json:"user_id"`
	AccountID string       `json:"account_id"`
	Email     string       `json:"email"`
	Role      session.Role `json:"role"`
	jwt.RegisteredClaims
}

// NewService constructs an authentication service.
func NewService(accountStore AccountStore, userStore UserStore, jwtConfig config.JWTConfig) *Service {
	return &Service{
		accountStore: accountStore,
		userStore:    userStore,
		jwtConfig:    jwtConfig,
	}
}

// Register creates a tenant account, owner user, and first JWT token pair.
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*Response, error) {
	if len(req.Password) < 8 {
		return nil, apperrors.ErrPasswordTooWeak
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	accountSID, authToken, err := credentials.GenerateAccountCredentials()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	account := &accountFeature.Account{
		ID:        uuid.New().String(),
		SID:       accountSID,
		AuthToken: authToken,
		Name:      strings.TrimSpace(req.CompanyName),
		Email:     normalizeEmail(req.Email),
		Balance:   0,
		Currency:  "USD",
		Status:    accountFeature.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.accountStore.Create(ctx, account); err != nil {
		return nil, err
	}

	user := &userFeature.User{
		ID:           uuid.New().String(),
		AccountID:    account.ID,
		Email:        normalizeEmail(req.Email),
		PasswordHash: string(passwordHash),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Role:         userFeature.RoleOwner,
		Status:       userFeature.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userStore.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.buildAuthResponse(user, account)
}

// Login validates user credentials and returns a fresh token pair.
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*Response, error) {
	user, err := s.userStore.GetByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			return nil, apperrors.ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status != userFeature.StatusActive {
		return nil, apperrors.ErrUserSuspended
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	account, err := s.accountStore.GetByID(ctx, user.AccountID)
	if err != nil {
		return nil, err
	}
	if !account.IsActive() {
		return nil, apperrors.ErrAccountSuspended
	}

	loginTime := time.Now().UTC()
	user.LastLoginAt = &loginTime
	_ = s.userStore.UpdateLastLogin(ctx, user.ID, loginTime)

	return s.buildAuthResponse(user, account)
}

// RefreshToken validates a refresh token and returns a new token pair.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*Response, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userStore.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user.Status != userFeature.StatusActive {
		return nil, apperrors.ErrUserSuspended
	}

	account, err := s.accountStore.GetByID(ctx, user.AccountID)
	if err != nil {
		return nil, err
	}
	if !account.IsActive() {
		return nil, apperrors.ErrAccountSuspended
	}

	return s.buildAuthResponse(user, account)
}

// ValidateToken verifies a JWT and returns its domain claims.
func (s *Service) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &TokenClaims{
		UserID:    claims.UserID,
		AccountID: claims.AccountID,
		Email:     claims.Email,
		Role:      claims.Role,
	}, nil
}

// RotateAccountCredentials issues a new Account SID and Auth Token for API basic auth.
func (s *Service) RotateAccountCredentials(ctx context.Context, accountID string) (*accountFeature.Account, error) {
	accountSID, authToken, err := credentials.GenerateAccountCredentials()
	if err != nil {
		return nil, err
	}

	if err := s.accountStore.UpdateCredentials(ctx, accountID, accountSID, authToken); err != nil {
		return nil, err
	}

	return s.accountStore.GetByID(ctx, accountID)
}

func (s *Service) buildAuthResponse(user *userFeature.User, account *accountFeature.Account) (*Response, error) {
	accessToken, err := s.issueToken(user, s.jwtConfig.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.issueToken(user, s.jwtConfig.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &Response{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtConfig.AccessTokenTTL.Seconds()),
		User:         user,
		Account:      account,
	}, nil
}

func (s *Service) issueToken(user *userFeature.User, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := jwtClaims{
		UserID:    user.ID,
		AccountID: user.AccountID,
		Email:     user.Email,
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.jwtConfig.Issuer,
			Subject:   user.ID,
			ID:        uuid.New().String(),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtConfig.Secret))
}

func (s *Service) parseToken(tokenString string) (*jwtClaims, error) {
	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
			}
			return []byte(s.jwtConfig.Secret), nil
		},
		jwt.WithIssuer(s.jwtConfig.Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}
	if token == nil || !token.Valid {
		return nil, apperrors.ErrInvalidToken
	}

	return claims, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
