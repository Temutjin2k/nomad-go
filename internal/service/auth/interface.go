package auth

import (
	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type UserRepo interface {
	CreateUser(user *models.User) (uuid.UUID, error)
	GetUser(email string) (*models.User, error)
	GetUserByID(userID int) (*models.User, error)
	DeleteUser(userID int) error
	UpdateUser(name string, role string, userID int) error
}

type TokenProvider interface {
	GenerateTokens(user *models.User) (*models.TokenPair, error)
	Refresh(refreshToken string) (*models.TokenPair, error)
	Validate(token string) (*models.CustomClaims, error)
}
