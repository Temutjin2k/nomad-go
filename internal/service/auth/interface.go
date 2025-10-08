package auth

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *models.User) (uuid.UUID, error)
	GetUser(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error)
}

type TokenProvider interface {
	GenerateTokens(user *models.User) (*models.TokenPair, error)
	Refresh(refreshToken string) (*models.TokenPair, error)
	Validate(token string) (*models.CustomClaims, error)
}
