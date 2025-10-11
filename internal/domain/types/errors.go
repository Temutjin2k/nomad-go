package types

import "errors"

var (
	ErrUserNotFound           = errors.New("user not found")
	ErrSessionNotFound        = errors.New("session not found")
	ErrDriverRegistered       = errors.New("driver already registered")
	ErrDriverAlreadyOnline    = errors.New("driver already online")
	ErrDriverAlreadyOffline   = errors.New("driver already offline")
	ErrDriverMustBeAvailable  = errors.New("driver must be available")
	ErrDriverAlreadyOnRide    = errors.New("driver is already on a ride")
	ErrLicenseAlreadyExists   = errors.New("license already exist")
	ErrInvalidLicenseFormat   = errors.New("invalid license format: AA123123")
	ErrNoCoordinates          = errors.New("no coordinates found")
	ErrDriverLocationNotFound = errors.New("driver location not found")
	ErrRideNotFound           = errors.New("ride not found")
	ErrRideNotArrived         = errors.New("ride status is not 'arrived'")
	ErrRideDriverMismatch     = errors.New("ride does not belong to the driver")
)
