package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// Service implements domain.AuthService.
type Service struct {
	accountRepo domain.AccountRepository
	userRepo    domain.UserRepository
	jwtConfig   config.JWTConfig
}

type jwtClaims struct {
	UserID    string          `json:"user_id"`
	AccountID string          `json:"account_id"`
	Email     string          `json:"email"`
	Role      domain.UserRole `json:"role"`
	jwt.RegisteredClaims
}

// NewService constructs an authentication service.
func NewService(accountRepo domain.AccountRepository, userRepo domain.UserRepository, jwtConfig config.JWTConfig) *Service {
	return &Service{
		accountRepo: accountRepo,
		userRepo:    userRepo,
		jwtConfig:   jwtConfig,
	}
}

// Register creates a tenant account, owner user, and first JWT token pair.
func (s *Service) Register(ctx context.Context, req *domain.RegisterRequest) (*domain.AuthResponse, error) {
	if len(req.Password) < 8 {
		return nil, domain.ErrPasswordTooWeak
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	accountSID, authToken, err := generateAccountCredentials()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	account := &domain.Account{
		ID:        uuid.New().String(),
		SID:       accountSID,
		AuthToken: authToken,
		Name:      strings.TrimSpace(req.CompanyName),
		Email:     normalizeEmail(req.Email),
		Balance:   0,
		Currency:  "USD",
		Status:    domain.AccountStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		AccountID:    account.ID,
		Email:        normalizeEmail(req.Email),
		PasswordHash: string(passwordHash),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Role:         domain.RoleOwner,
		Status:       domain.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.buildAuthResponse(user, account)
}

// Login validates user credentials and returns a fresh token pair.
func (s *Service) Login(ctx context.Context, req *domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, normalizeEmail(req.Email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrUserSuspended
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	account, err := s.accountRepo.GetByID(ctx, user.AccountID)
	if err != nil {
		return nil, err
	}
	if !account.IsActive() {
		return nil, domain.ErrAccountSuspended
	}

	loginTime := time.Now().UTC()
	user.LastLoginAt = &loginTime
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID, loginTime)

	return s.buildAuthResponse(user, account)
}

// RefreshToken validates a refresh token and returns a new token pair.
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*domain.AuthResponse, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrUserSuspended
	}

	account, err := s.accountRepo.GetByID(ctx, user.AccountID)
	if err != nil {
		return nil, err
	}
	if !account.IsActive() {
		return nil, domain.ErrAccountSuspended
	}

	return s.buildAuthResponse(user, account)
}

// ValidateToken verifies a JWT and returns its domain claims.
func (s *Service) ValidateToken(ctx context.Context, tokenString string) (*domain.TokenClaims, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &domain.TokenClaims{
		UserID:    claims.UserID,
		AccountID: claims.AccountID,
		Email:     claims.Email,
		Role:      claims.Role,
	}, nil
}

// RotateAccountCredentials issues a new Account SID and Auth Token for API basic auth.
func (s *Service) RotateAccountCredentials(ctx context.Context, accountID string) (*domain.Account, error) {
	accountSID, authToken, err := generateAccountCredentials()
	if err != nil {
		return nil, err
	}

	if err := s.accountRepo.UpdateCredentials(ctx, accountID, accountSID, authToken); err != nil {
		return nil, err
	}

	return s.accountRepo.GetByID(ctx, accountID)
}

func (s *Service) buildAuthResponse(user *domain.User, account *domain.Account) (*domain.AuthResponse, error) {
	accessToken, err := s.issueToken(user, s.jwtConfig.AccessTokenTTL)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.issueToken(user, s.jwtConfig.RefreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtConfig.AccessTokenTTL.Seconds()),
		User:         user,
		Account:      account,
	}, nil
}

func (s *Service) issueToken(user *domain.User, ttl time.Duration) (string, error) {
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
			return nil, domain.ErrTokenExpired
		}
		return nil, domain.ErrInvalidToken
	}
	if token == nil || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}

func generateAccountCredentials() (string, string, error) {
	accountSID := fmt.Sprintf("AC%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:32])

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}

	return accountSID, hex.EncodeToString(tokenBytes), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
