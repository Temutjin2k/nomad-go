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
	GenerateTokens(ctx context.Context, user *models.User) (*models.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (*models.TokenPair, error)
	Validate(ctx context.Context, token string) (*models.CustomClaims, error)
}

type RefreshTokenRepo interface {
	Save(ctx context.Context, record *models.RefreshTokenRecord) error
	Get(ctx context.Context, tokenID uuid.UUID) (*models.RefreshTokenRecord, error)
	MarkUsed(ctx context.Context, tokenID uuid.UUID) error
}
