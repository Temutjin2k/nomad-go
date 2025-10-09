package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/golang-jwt/jwt/v5"
)

const (
	RefreshToken = "refresh_token"
	AccessToken  = "access_token"
)

func IsValidTokenType(tokenType string) bool {
	return tokenType == RefreshToken || tokenType == AccessToken
}

type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type CustomClaims struct {
	UserID    uuid.UUID `json:"ID"`
	TokenID   uuid.UUID `json:"jti"`
	TokenType string    `json:"typ"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	jwt.RegisteredClaims
}

type RefreshTokenRecord struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at"`
	Revoked   bool       `json:"revoked"`
	CreatedAt time.Time  `json:"created_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}
