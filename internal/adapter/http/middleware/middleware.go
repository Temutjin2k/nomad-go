package middleware

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type (
	AuthService interface {
		RoleCheck(ctx context.Context, token string) (*models.User, error)
	}

	Middleware struct {
		auth AuthService
		log  logger.Logger
	}
)

func NewMiddleware(auth AuthService, log logger.Logger) *Middleware {
	return &Middleware{
		auth: auth,
		log:  log,
	}
}
