package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrDatabaseUnavailable = errors.New("database is unavailable")
	ErrAccountNotFound     = errors.New("account not found")
	ErrAccountSuspended    = errors.New("account is suspended")
	ErrInvalidCredentials  = errors.New("invalid API key, auth token, or password")
	ErrUnauthorized        = errors.New("unauthorized request")
	ErrForbidden           = errors.New("forbidden: insufficient permissions")

	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email address is already registered")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrPasswordTooWeak    = errors.New("password must be at least 8 characters")
	ErrUserSuspended      = errors.New("user account is suspended")
	ErrInvalidToken       = errors.New("invalid or malformed token")
	ErrTokenExpired       = errors.New("token has expired")

	ErrInsufficientFunds = errors.New("insufficient account balance")
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrNegativeBalance   = errors.New("balance cannot be negative")

	ErrIdempotencyConflict = errors.New("concurrent request in progress with the same idempotency key")
	ErrIdempotencyKeyEmpty = errors.New("idempotency key cannot be empty")

	ErrRateLimitExceeded = errors.New("rate limit exceeded, please retry later")

	ErrInvalidPhoneNumber = errors.New("invalid E.164 phone number format")
	ErrInvalidSMSBody     = errors.New("sms message body cannot be empty")
	ErrSMSNotFound        = errors.New("sms message record not found")
	ErrCarrierTimeout     = errors.New("upstream carrier timeout")
	ErrCarrierRejected    = errors.New("upstream carrier rejected message")
)

type AppError struct {
	Err        error  `json:"-"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(err error, code string, message string, statusCode int) *AppError {
	return &AppError{
		Err:        err,
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}
