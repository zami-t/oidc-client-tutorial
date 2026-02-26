package model

import "fmt"

// ErrCode represents application-level error codes.
type ErrCode string

const (
	ErrCodeStateMismatch           ErrCode = "STATE_MISMATCH"
	ErrCodeTokenVerificationFailed ErrCode = "TOKEN_VERIFICATION_FAILED"
	ErrCodeUnknownIdp              ErrCode = "UNKNOWN_IDP"
	ErrCodeSessionNotFound         ErrCode = "SESSION_NOT_FOUND"
	ErrCodeServerError             ErrCode = "SERVER_ERROR"
	ErrCodeAuthorizationError      ErrCode = "AUTHORIZATION_ERROR"
)

// AppError is an application-level error with a code and message.
type AppError struct {
	Code    ErrCode
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new AppError.
func NewAppError(code ErrCode, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
