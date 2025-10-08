package wrap

import (
	"context"
)

type (
	// LogCtx holds contextual information for logging
	LogCtx struct {
		Action    string
		UserID    string
		RequestID string
		RideID    string
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
