package wrap

import (
	"context"
)

type (
	// LogCtx holds contextual information for logging
	LogCtx struct {
		Action      string
		UserID      string
		RequestID   string
		RideID      string
		PassengerID string
		DriverID    string
		RideNumber  string
		OfferID     string
	}

	// logCtxKeyStruct is an unexported type for context keys defined in this package.
	logCtxKeyStruct struct{}
)

// logCtxKey is the key for log context values
var LogCtxKey = &logCtxKeyStruct{}

// WithLogCtx returns a new context with the provided LogCtx
func WithLogCtx(ctx context.Context, newLc LogCtx) context.Context {
	// Check if there's an existing LogCtx and merge values
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		if newLc.Action == "" {
			newLc.Action = lc.Action
		}
		if newLc.UserID == "" {
			newLc.UserID = lc.UserID
		}
		if newLc.RequestID == "" {
			newLc.RequestID = lc.RequestID
		}
		if newLc.RideID == "" {
			newLc.RideID = lc.RideID
		}
		if newLc.DriverID == "" {
			newLc.DriverID = lc.DriverID
		}
		return context.WithValue(ctx, LogCtxKey, newLc)
	}
	return context.WithValue(ctx, LogCtxKey, newLc)
}

// WithUserID adds or updates the UserID in the LogCtx within the context
func WithUserID(ctx context.Context, userID string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.UserID = userID
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{UserID: userID})
}

// WithDriverID adds or updates the DriverID in the LogCtx within the context
func WithDriverID(ctx context.Context, driverID string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.DriverID = driverID
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{DriverID: driverID})
}

// WithRequestID adds or updates the RequestID in the LogCtx within the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.RequestID = requestID
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{RequestID: requestID})
}

// WithRideID adds or updates the RideID in the LogCtx within the context
func WithRideID(ctx context.Context, rideID string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.RideID = rideID
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{RideID: rideID})
}

// WithAction adds or updates the Action in the LogCtx within the context
func WithAction(ctx context.Context, action string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.Action = action
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{Action: action})
}

// WithPassengerID adds or updates the PassengerID in the LogCtx within the context
func WithPassengerID(ctx context.Context, passengerID string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.PassengerID = passengerID
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{PassengerID: passengerID})
}

// WithRideNumber adds or updates the RideNumber in the LogCtx within the context
func WithRideNumber(ctx context.Context, rideNumber string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.RideNumber = rideNumber
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{RideNumber: rideNumber})
}

// WithOfferID adds or updates the OfferID in the LogCtx within the context
func WithOfferID(ctx context.Context, offerID string) context.Context {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		lc.OfferID = offerID
		return context.WithValue(ctx, LogCtxKey, lc)
	}
	return context.WithValue(ctx, LogCtxKey, LogCtx{OfferID: offerID})
}

func GetRequestID(ctx context.Context) string {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		return lc.RequestID
	}
	return ""
}

func GetLogCtx(ctx context.Context) LogCtx {
	if lc, ok := ctx.Value(LogCtxKey).(LogCtx); ok {
		return lc
	}
	return LogCtx{}
}
