package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/golang-jwt/jwt/v5"
)

const (
	Refresh = "refresh_token"
	Access  = "access_token"
)

type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type CustomClaims struct {
	ID        uuid.UUID `json:"ID"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	Role      string    `json:"role"`
	IsRefresh bool      `json:"is_refresh"`
	jwt.RegisteredClaims
}
