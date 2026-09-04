package identity

import "errors"

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrConflict           = errors.New("account registration unavailable")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("authentication required")
	ErrNotFound           = errors.New("not found")
)
