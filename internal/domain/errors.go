package domain

import (
	"errors"
	"fmt"
)

// Domain sentinel errors.
var (
	// Account errors
	ErrAccountNotFound      = errors.New("account not found")
	ErrAccountSuspended     = errors.New("account is suspended")
	ErrInvalidCredentials   = errors.New("invalid API key, auth token, or password")
	ErrUnauthorized         = errors.New("unauthorized request")
	ErrForbidden            = errors.New("forbidden: insufficient permissions")

	// User & Auth errors
	ErrUserNotFound         = errors.New("user not found")
	ErrEmailAlreadyExists   = errors.New("email address is already registered")
	ErrInvalidPassword      = errors.New("invalid password")
	ErrPasswordTooWeak      = errors.New("password must be at least 8 characters")
	ErrUserSuspended        = errors.New("user account is suspended")
	ErrInvalidToken         = errors.New("invalid or malformed token")
	ErrTokenExpired         = errors.New("token has expired")

	// Billing errors
	ErrInsufficientFunds    = errors.New("insufficient account balance")
	ErrInvalidAmount        = errors.New("amount must be greater than zero")
	ErrNegativeBalance      = errors.New("balance cannot be negative")

	// Idempotency errors
	ErrIdempotencyConflict  = errors.New("concurrent request in progress with the same idempotency key")
	ErrIdempotencyKeyEmpty  = errors.New("idempotency key cannot be empty")

	// Rate limiting errors
	ErrRateLimitExceeded    = errors.New("rate limit exceeded, please retry later")

	// SMS errors
	ErrInvalidPhoneNumber   = errors.New("invalid E.164 phone number format")
	ErrInvalidSMSBody       = errors.New("sms message body cannot be empty")
	ErrSMSNotFound          = errors.New("sms message record not found")
	ErrCarrierTimeout       = errors.New("upstream carrier timeout")
	ErrCarrierRejected      = errors.New("upstream carrier rejected message")

	// Call / Voice errors
	ErrCallNotFound         = errors.New("call session not found")
	ErrInvalidCallState     = errors.New("invalid call state transition")
	ErrESLConnectionFailed  = errors.New("failed to connect to FreeSWITCH event socket")
	ErrCallAlreadyEnded     = errors.New("call is already terminated")
)

// AppError represents a typed domain error with HTTP status context and code.
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

// NewAppError constructs a new AppError wrapper.
func NewAppError(err error, code string, message string, statusCode int) *AppError {
	return &AppError{
		Err:        err,
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}
