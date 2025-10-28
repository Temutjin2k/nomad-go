package microservices

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Temutjin2k/ride-hail-system/config"
	httpserver "github.com/Temutjin2k/ride-hail-system/internal/adapter/http/server"
	wshandler "github.com/Temutjin2k/ride-hail-system/internal/adapter/http/ws"
	repo "github.com/Temutjin2k/ride-hail-system/internal/adapter/postgres"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/rabbit"
	"github.com/Temutjin2k/ride-hail-system/internal/service/auth"
	ridecalc "github.com/Temutjin2k/ride-hail-system/internal/service/calculator"
	ridego "github.com/Temutjin2k/ride-hail-system/internal/service/ride"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	postgres "github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	rabbitmq "github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
	ws "github.com/Temutjin2k/ride-hail-system/pkg/wsHub"
)

type RideService struct {
	postgresDB *postgres.PostgreDB
	httpServer *httpserver.API
	rabbitMQ   *rabbitmq.RabbitMQ
	consumers  *RideConsumers

	cfg config.Config
	log logger.Logger
}

type RideConsumers struct {
	rideConsumer *rabbit.RideBroker
	rideService  *ridego.RideService
	log          logger.Logger
}

func (c *RideConsumers) Start(ctx context.Context, errCh chan error) {
	go func() {
		c.log.Info(ctx, "ConsumeDriverLocationUpdate has been started")
		if err := c.rideConsumer.ConsumeDriverLocationUpdate(ctx, c.rideService.HandleDriverLocationUpdate); err != nil {
			errCh <- fmt.Errorf("failed to start ConsumeDriverLocationUpdate: %w", err)
			return
		}
		c.log.Info(ctx, "ConsumeDriverLocationUpdate has been finished")
	}()

	go func() {
		c.log.Info(ctx, "ConsumeDriverStatusUpdate has been started")
		if err := c.rideConsumer.ConsumeDriverStatusUpdate(ctx, c.rideService.HandleDriverStatusUpdate); err != nil {
			errCh <- fmt.Errorf("failed to start ConsumeDriverStatusUpdate: %w", err)
			return
		}
		c.log.Info(ctx, "ConsumeDriverStatusUpdate has been finished")
	}()
}

// NewRide creates ride microservice
func NewRide(ctx context.Context, cfg config.Config, log logger.Logger) (*RideService, error) {
	// init Postgres
	postgresDB, err := postgres.New(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to setup database: %w", err)
	}

	// init RabbitMQ
	rabbitClient, err := rabbitmq.New(ctx, cfg.RabbitMQ.GetDSN(), log)
	if err != nil {
		return nil, fmt.Errorf("failed to setup rabbitmq: %w", err)
	}
	rabbitRideBroker := rabbit.NewRideBroker(rabbitClient, log)

	// init repositories
	rideRepo := repo.NewRideRepo(postgresDB.Pool)
	userRepo := repo.NewUserRepo(postgresDB.Pool)
	refreshTokenRepo := repo.NewRefreshTokenRepo(postgresDB.Pool)
	eventRepo := repo.NewRideEvent(postgresDB.Pool)

	// init services
	trm := trm.New(postgresDB.Pool)
	calculator := ridecalc.New()

	hub := ws.NewConnHub(log)
	wsRide := wshandler.NewRideWsHandler(hub)

	rideService := ridego.NewRideService(rideRepo, calculator, trm, rabbitRideBroker, wsRide, eventRepo, log)
	tokenSvc := auth.NewTokenService(cfg.Auth.JWTSecret, userRepo, refreshTokenRepo, trm, cfg.Auth.RefreshTokenTTL, cfg.Auth.AccessTokenTTL, log)
	authSvc := auth.NewAuthService(userRepo, tokenSvc, log)

	// init http server
	httpServer, err := httpserver.New(ctx, cfg, nil, rideService, nil, authSvc, log)
	if err != nil {
		return nil, fmt.Errorf("failed to setup http server: %w", err)
	}

	return &RideService{
		httpServer: httpServer,
		postgresDB: postgresDB,
		rabbitMQ:   rabbitClient,
		consumers: &RideConsumers{
			rideConsumer: rabbitRideBroker,
			rideService:  rideService,
			log:          log,
		},

		cfg: cfg,
		log: log,
	}, nil
}

func (s *RideService) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	s.httpServer.Run(ctx, errCh)
	s.consumers.Start(ctx, errCh)

	defer func() {
		s.close(ctx)
		s.log.Info(ctx, "ride service closed")
	}()

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info(ctx, "ride service has been started")

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
			s.log.Warn(ctx, "failed to gracefully close http server", "error", err.Error())
		}
	}

	if s.postgresDB != nil && s.postgresDB.Pool != nil {
		s.postgresDB.Pool.Close()
	}

	if s.rabbitMQ != nil {
		if err := s.rabbitMQ.Close(ctx); err != nil {
			s.log.Warn(ctx, "failed to close rabbitmq connection", "error", err.Error())
		}
	}
}
