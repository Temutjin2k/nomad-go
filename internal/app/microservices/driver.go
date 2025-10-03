package microservices

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/server"
	repo "github.com/Temutjin2k/ride-hail-system/internal/adapter/postgres"
	drivergo "github.com/Temutjin2k/ride-hail-system/internal/service/driver.go"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	"github.com/Temutjin2k/ride-hail-system/pkg/postgres"
)

type DriverService struct {
	postgresDB *postgres.PostgreDB
	httpServer *server.API
	cfg        config.Config
	log        logger.Logger
}

func NewDriver(ctx context.Context, cfg config.Config, log logger.Logger) (*DriverService, error) {
	postgresDB, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		log.Error(ctx, "Failed to setup database", err)
		return nil, err
	}

	driverRepo := repo.NewDriverRepo(postgresDB.Pool)
	driverService := drivergo.New(driverRepo, log)

	httpServer, err := server.New(cfg, driverService, nil, log)
	if err != nil {
		log.Error(ctx, "Failed to setup http server", err)
		return nil, err
	}

	return &DriverService{
		httpServer: httpServer,
		postgresDB: postgresDB,
		cfg:        cfg,
		log:        log,
	}, nil
}

func (s *DriverService) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	s.httpServer.Run(ctx, errCh)
	defer func() {
		s.close(ctx)
		s.log.Info(ctx, "driver service closed")
	}()

	// Waiting signal
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info(ctx, "Driver has been service started")

	select {
	case errRun := <-errCh:
		return errRun
	case sig := <-shutdownCh:
		s.log.Info(ctx, "shuting down application", "signal", sig.String())
		return nil
	}
}

func (s *DriverService) close(ctx context.Context) {
	if s.httpServer != nil {
		if err := s.httpServer.Stop(ctx); err != nil {
			s.log.Warn(ctx, "Failed to gracefully close http server", "error", err.Error())
		}
	}

	if s.postgresDB != nil && s.postgresDB.Pool != nil {
		s.postgresDB.Pool.Close()
	}
}
