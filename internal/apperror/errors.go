// Package apperror defines sentinel errors shared across bounded contexts.
// Domain services wrap these with fmt.Errorf("...: %w", apperror.Err...).
// Transport layers map them to HTTP statuses via httpapi.RespondError.
package apperror

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrForbidden          = errors.New("forbidden")
	ErrValidation         = errors.New("validation error")
)
