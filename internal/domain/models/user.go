package models

import (
	"context"
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

// anonymousUser variable for user.
var anonymousUser = &User{}

func AnonymousUser() *User {
	return anonymousUser
}

// --- context helpers ---

type ctxKey int

const userCtxKey ctxKey = iota

// UserFromContext returns authenticated user or nil.
func UserFromContext(ctx context.Context) *User {
	u, ok := ctx.Value(userCtxKey).(*User)
	if !ok {
		return nil
	}
	return u
}

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

type UserCreateRequest struct {
	Name     string         `json:"name"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Attrs    map[string]any `json:"attrs,omitempty"`
}

type User struct {
	ID           uuid.UUID      `json:"id"`
	Email        string         `json:"email"`
	Role         string         `json:"role"`
	Status       string         `json:"status"`
	PasswordHash string         `json:"-"`               // stored in DB as password_hash
	Attrs        map[string]any `json:"attrs,omitempty"` // jsonb
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at,omitzero"`
}

func (u *User) IsAnonymous() bool {
	return u == anonymousUser
}
