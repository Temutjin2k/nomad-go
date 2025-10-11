package types

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrDriverRegistered     = errors.New("driver already registered")
	ErrDriverAlreadyOnline  = errors.New("driver already online")
	ErrLicenseAlreadyExists = errors.New("license already exist")
	ErrInvalidLicenseFormat = errors.New("invalid license format: AA123123")

	ErrRideCannotBeCancelled = errors.New("ride cannot be cancelled")
	ErrRideNotFound = errors.New("ride not found")
	ErrNotFound = errors.New("requested item not found")
)
