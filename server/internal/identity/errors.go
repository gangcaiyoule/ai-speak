package identity

import "errors"

var (
	ErrInvalidRequest     = errors.New("identity: invalid request")
	ErrConflict           = errors.New("identity: user already exists")
	ErrInvalidCredentials = errors.New("identity: invalid credentials")
	ErrUnauthorized       = errors.New("identity: authentication required")
	ErrNotFound           = errors.New("identity: not found")
)
