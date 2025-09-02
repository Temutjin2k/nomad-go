package types

import "context"

// Context key for request_id (unexported to avoid collisions)
type requestID struct{}

var requestIDKey = &requestID{}

func GetRequestIDKey() *requestID {
	return requestIDKey
}

// Helper to set request_id in context
func WithRequestIDContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, GetRequestIDKey(), requestID)
}
