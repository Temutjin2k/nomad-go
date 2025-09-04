package microservices

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type DriverService struct {
	cfg config.Config
	log logger.Logger
}

func NewDriver(ctx context.Context, cfg config.Config, log logger.Logger) (*DriverService, error) {
	return &DriverService{
		cfg: cfg,
		log: log,
	}, nil
}

func (s *DriverService) Start(ctx context.Context) error {
	return nil
}
