package auth

import "errors"

var (
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrTokenGenerateFail     = errors.New("failed to generate token")
	ErrUnexpected            = errors.New("unexpected error")
	ErrNotUniqueEmail        = errors.New("user with this email already exists")
	ErrCannotCreateAdmin     = errors.New("cannot create admin via API")
	ErrInvalidToken          = errors.New("invalid token")
	ErrExpToken              = errors.New("expired token")
	ErrUserWithEmailNotFound = errors.New("user with this email not found")
	ErrActionForbidden       = errors.New("action forbidden")
)
