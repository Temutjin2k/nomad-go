package types

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrDriverRegistered     = errors.New("driver already registered")
	ErrLicenseAlreadyExists = errors.New("license already exist")
	ErrInvalidLicenseFormat = errors.New("invalid license format: AA123123")
)
