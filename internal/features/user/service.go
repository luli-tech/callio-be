package user

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
	"golang.org/x/crypto/bcrypt"
)

type service struct {
	userRepo Repository
}

func NewService(userRepo Repository) ServiceContract {
	return &service{userRepo: userRepo}
}

func (s *service) CreateUser(ctx context.Context, req *CreateRequest) (*User, error) {
	if len(req.Password) < 8 {
		return nil, apperrors.ErrPasswordTooWeak
	}
	if !isValidRole(req.Role) {
		return nil, apperrors.ErrForbidden
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := &User{
		ID:           uuid.New().String(),
		AccountID:    req.AccountID,
		Email:        normalizeEmail(req.Email),
		PasswordHash: string(hash),
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		Role:         req.Role,
		Status:       StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetByEmail(ctx, normalizeEmail(email))
}

func (s *service) ListUsersByAccount(ctx context.Context, accountID string) ([]*User, error) {
	return s.userRepo.ListByAccount(ctx, accountID)
}

func (s *service) UpdateProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*User, error) {
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

func (s *service) UpdateRole(ctx context.Context, userID string, role Role) (*User, error) {
	if !isValidRole(role) {
		return nil, apperrors.ErrForbidden
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

func (s *service) UpdateStatus(ctx context.Context, userID string, status Status) (*User, error) {
	if !isValidStatus(status) {
		return nil, apperrors.ErrForbidden
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

func (s *service) DeleteUser(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}

func isValidRole(role Role) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleDeveloper, RoleMember:
		return true
	default:
		return false
	}
}

func isValidStatus(status Status) bool {
	switch status {
	case StatusActive, StatusSuspended, StatusPending:
		return true
	default:
		return false
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
