package microservices

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type AdminService struct {
	cfg config.Config
	log logger.Logger
}

func NewAdmin(ctx context.Context, cfg config.Config, log logger.Logger) (*AdminService, error) {
	return &AdminService{
		cfg: cfg,
		log: log,
	}, nil
}

func (s *AdminService) Start(ctx context.Context) error {
	return nil
}
