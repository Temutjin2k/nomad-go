package microservices

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

type RideService struct {
	cfg config.Config
	log logger.Logger
}

func NewRide(ctx context.Context, cfg config.Config, log logger.Logger) (*RideService, error) {
	return &RideService{
		cfg: cfg,
		log: log,
	}, nil
}

func (s *RideService) Start(ctx context.Context) error {
	return nil
}
