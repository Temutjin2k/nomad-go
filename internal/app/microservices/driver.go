package microservices

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/server"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/locationIQ"
	repo "github.com/Temutjin2k/ride-hail-system/internal/adapter/postgres"
	publisher "github.com/Temutjin2k/ride-hail-system/internal/adapter/rabbit"
	"github.com/Temutjin2k/ride-hail-system/internal/service/auth"
	drivergo "github.com/Temutjin2k/ride-hail-system/internal/service/driver"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	"github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
)

type DriverService struct {
	postgresDB *postgres.PostgreDB
	httpServer *server.API
	rabbitMQ   *rabbit.RabbitMQ
	cfg        config.Config
	log        logger.Logger
}

func NewDriver(ctx context.Context, cfg config.Config, log logger.Logger) (*DriverService, error) {
	postgresDB, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		log.Error(ctx, "Failed to setup database", err)
		return nil, err
	}

	rabbitMq, err := rabbit.New(ctx, cfg.RabbitMQ.GetDSN(), log)
	if err != nil {
		log.Error(ctx, "Failed to setup rabbitmq", err)
		return nil, err
	}

	// Repo adapters
	trm := trm.New(postgresDB.Pool)
	driverRepo := repo.NewDriverRepo(postgresDB.Pool)
	sessionRepo := repo.NewSessionRepo(postgresDB.Pool)
	coordinateRepo := repo.NewCoordinateRepo(postgresDB.Pool)
	userRepo := repo.NewUserRepo(postgresDB.Pool)
	rideRepo := repo.NewRideRepo(postgresDB.Pool)
	refreshTokenRepo := repo.NewRefreshTokenRepo(postgresDB.Pool)

	// Message Broker publisher
	driverProducer := publisher.NewDriverProducer(rabbitMq)

	// External API client
	locationIQclient := locationIQ.New(cfg.ExternalAPIConfig.LocationIQapiKey)

	// Main Service
	driverService := drivergo.New(driverRepo, sessionRepo, coordinateRepo, userRepo, rideRepo, locationIQclient, driverProducer, trm, log)

	tokenService := auth.NewTokenService(cfg.Auth.JWTSecret, userRepo, refreshTokenRepo, trm, cfg.Auth.RefreshTokenTTL, cfg.Auth.AccessTokenTTL, log)
	authService := auth.NewAuthService(userRepo, tokenService, log)

	httpServer, err := server.New(cfg, driverService, nil, authService, log)
	if err != nil {
		log.Error(ctx, "Failed to setup http server", err)
		return nil, err
	}

	return &DriverService{
		httpServer: httpServer,
		postgresDB: postgresDB,
		rabbitMQ:   rabbitMq,
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

	if s.rabbitMQ != nil && s.rabbitMQ.Conn != nil {
		s.rabbitMQ.Conn.Close()
	}
}