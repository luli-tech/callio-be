package user

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// Service implements domain.UserService.
type Service struct {
	userRepo domain.UserRepository
}

// NewService constructs a tenant user management service.
func NewService(userRepo domain.UserRepository) *Service {
	return &Service{userRepo: userRepo}
}

// CreateUser creates a tenant user with a hashed password.
func (s *Service) CreateUser(ctx context.Context, req *domain.CreateUserRequest) (*domain.User, error) {
	if len(req.Password) < 8 {
		return nil, domain.ErrPasswordTooWeak
	}
	if !isValidRole(req.Role) {
		return nil, domain.ErrForbidden
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New().String(),
		AccountID:    req.AccountID,
		Email:        normalizeEmail(req.Email),
		PasswordHash: string(hash),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Role:         req.Role,
		Status:       domain.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID fetches a user by ID.
func (s *Service) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetUserByEmail fetches a user by email.
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	return s.userRepo.GetByEmail(ctx, normalizeEmail(email))
}

// ListUsersByAccount lists all users within a tenant account.
func (s *Service) ListUsersByAccount(ctx context.Context, accountID string) ([]*domain.User, error) {
	return s.userRepo.ListByAccount(ctx, accountID)
}

// UpdateProfile updates mutable profile fields.
func (s *Service) UpdateProfile(ctx context.Context, userID string, req *domain.UpdateProfileRequest) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.FirstName = strings.TrimSpace(req.FirstName)
	user.LastName = strings.TrimSpace(req.LastName)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateRole changes a user's RBAC role.
func (s *Service) UpdateRole(ctx context.Context, userID string, role domain.UserRole) (*domain.User, error) {
	if !isValidRole(role) {
		return nil, domain.ErrForbidden
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Role = role
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateStatus changes a user's lifecycle status.
func (s *Service) UpdateStatus(ctx context.Context, userID string, status domain.UserStatus) (*domain.User, error) {
	if !isValidStatus(status) {
		return nil, domain.ErrForbidden
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.Status = status
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// DeleteUser deletes a tenant user.
func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}

func isValidRole(role domain.UserRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleDeveloper, domain.RoleMember:
		return true
	default:
		return false
	}
}

func isValidStatus(status domain.UserStatus) bool {
	switch status {
	case domain.UserStatusActive, domain.UserStatusSuspended, domain.UserStatusPending:
		return true
	default:
		return false
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
