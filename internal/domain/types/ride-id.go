package types

import "context"

type rideID struct{}

var rideIDKey = &rideID{}

func GetRideIDKey() *rideID {
	return rideIDKey
}

// Helper to set ride_id in context
func WithRideIDContext(ctx context.Context, rideID string) context.Context {
	return context.WithValue(ctx, GetRideIDKey(), rideID)
}
