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

type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type CustomClaims struct {
	UserID uuid.UUID `json:"ID"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}
