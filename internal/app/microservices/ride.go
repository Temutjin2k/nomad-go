package microservices

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Temutjin2k/ride-hail-system/config"
	httpserver "github.com/Temutjin2k/ride-hail-system/internal/adapter/http/server"
	repo "github.com/Temutjin2k/ride-hail-system/internal/adapter/postgres"
	"github.com/Temutjin2k/ride-hail-system/internal/service/auth"
	ridego "github.com/Temutjin2k/ride-hail-system/internal/service/ride"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	postgres "github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
)

type RideService struct {
	postgresDB *postgres.PostgreDB
	httpServer *httpserver.API
	cfg        config.Config
	log        logger.Logger
	trm        trm.TxManager
	publisher  ridego.RideMsgBroker
}

func NewRide(ctx context.Context, cfg config.Config, log logger.Logger) (*RideService, error) {
	postgresDB, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		log.Error(ctx, "Failed to setup database", err)
		return nil, err
	}

	trm := trm.New(postgresDB.Pool)
	rideRepo := repo.NewRideRepo(postgresDB.Pool)
	userRepo := repo.NewUserRepo(postgresDB.Pool)
	refreshTokenRepo := repo.NewRefreshTokenRepo(postgresDB.Pool)

	// TODO: fix all of them
	rideService := ridego.NewRideService(rideRepo, log, trm, nil)
	_ = rideService

	tokenSvc := auth.NewTokenService(cfg.Auth.JWTSecret, userRepo, refreshTokenRepo, trm, cfg.Auth.RefreshTokenTTL, cfg.Auth.AccessTokenTTL, log)
	authSvc := auth.NewAuthService(userRepo, tokenSvc, log)

	httpServer, err := httpserver.New(cfg, nil, nil, nil, authSvc, log)
	if err != nil {
		log.Error(ctx, "Failed to setup http server", err)
		return nil, err
	}

	return &RideService{
		httpServer: httpServer,
		postgresDB: postgresDB,
		cfg:        cfg,
		log:        log,
	}, nil
}

func (s *RideService) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	s.httpServer.Run(ctx, errCh)
	defer func() {
		s.close(ctx)
		s.log.Info(ctx, "ride service closed")
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info(ctx, "Ride service has been started")

	select {
	case errRun := <-errCh:
		return errRun
	case sig := <-shutdownCh:
		s.log.Info(ctx, "shutting down application", "signal", sig.String())
		return nil
	}
}

func (s *RideService) close(ctx context.Context) {
	if s.httpServer != nil {
		if err := s.httpServer.Stop(ctx); err != nil {
			s.log.Warn(ctx, "Failed to gracefully close http server", "error", err.Error())
		}
	}

	if s.postgresDB != nil && s.postgresDB.Pool != nil {
		s.postgresDB.Pool.Close()
	}
}
