package domain

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrUnauthorized       = errors.New("missing or invalid authorization token")
	ErrForbidden          = errors.New("forbidden")
	ErrConflict           = errors.New("conflict")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrValidation         = errors.New("validation failed")
	ErrVersionMismatch    = errors.New("task was modified by someone else")
	ErrNotTeamMember      = errors.New("user is not a member of the team")
)
