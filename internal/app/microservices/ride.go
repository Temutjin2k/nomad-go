package microservices

import (
	"context"

	"github.com/Temutjin2k/ride-hail-system/config"
	httpserver "github.com/Temutjin2k/ride-hail-system/internal/adapter/http/server"
	repo "github.com/Temutjin2k/ride-hail-system/internal/adapter/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	postgres "github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
)

type RideService struct {
	postgresDB *postgres.PostgreDB
	httpServer *httpserver.API
	cfg config.Config
	log logger.Logger
}

func NewRide(ctx context.Context, cfg config.Config, log logger.Logger) (*RideService, error) {
	postgresDB, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		log.Error(ctx, "Failed to setup db", err)
		return nil, err
	}

	trm := trm.New(postgresDB.Pool)

	

	

}

func (s *RideService) Start(ctx context.Context) error {
	return nil
}
