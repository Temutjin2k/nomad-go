package microservices

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/server"
	wshandler "github.com/Temutjin2k/ride-hail-system/internal/adapter/http/ws"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/locationIQ"
	repo "github.com/Temutjin2k/ride-hail-system/internal/adapter/postgres"
	rabbitAdapter "github.com/Temutjin2k/ride-hail-system/internal/adapter/rabbit"
	"github.com/Temutjin2k/ride-hail-system/internal/service/auth"
	ridecalc "github.com/Temutjin2k/ride-hail-system/internal/service/calculator"
	drivergo "github.com/Temutjin2k/ride-hail-system/internal/service/driver"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	"github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
	ws "github.com/Temutjin2k/ride-hail-system/pkg/wsHub"
)

type DriverService struct {
	postgresDB *postgres.PostgreDB
	httpServer *server.API
	rabbitMQ   *rabbit.RabbitMQ
	consumers  Consumers
	cfg        config.Config
	log        logger.Logger
}

type Consumers struct {
	rideConsumer *rabbitAdapter.RideConsumer
	uc           *drivergo.Service
	log          logger.Logger
}

func (c *Consumers) Start(ctx context.Context, errCh chan error) {
	go func() {
		c.log.Info(ctx, "Ride request consume has been started")
		if err := c.rideConsumer.ConsumeRideRequest(ctx, c.uc.SearchDriver); err != nil {
			errCh <- fmt.Errorf("failed to start ride consume process: %w", err)
			return
		}
		c.log.Info(ctx, "Ride request consume has been finished")
	}()
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
	driverProducer := rabbitAdapter.NewDriverProducer(rabbitMq)

	// Message Broker consumer
	rideConsumer := rabbitAdapter.NewRideConsumer(rabbitMq, log)

	// External API client
	locationIQclient := locationIQ.New(cfg.ExternalAPIConfig.LocationIQapiKey)

	// Calculator service
	calculator := ridecalc.New()

	// Websocket service
	wsHub := ws.NewConnHub(log)
	sender := wshandler.NewDriverHub(wsHub)

	// Main Service
	driverService := drivergo.New(driverRepo, sessionRepo, coordinateRepo, userRepo, rideRepo, locationIQclient, driverProducer, calculator, sender, trm, log)
	tokenService := auth.NewTokenService(cfg.Auth.JWTSecret, userRepo, refreshTokenRepo, trm, cfg.Auth.RefreshTokenTTL, cfg.Auth.AccessTokenTTL, log)
	authService := auth.NewAuthService(userRepo, tokenService, log)

	options := &handler.DriverServiceOptions{
		WsConnections: wsHub,
		Service:       driverService,
	}

	httpServer, err := server.New(ctx, cfg, options, nil, nil, authService, log)
	if err != nil {
		log.Error(ctx, "Failed to setup http server", err)
		return nil, err
	}

	return &DriverService{
		httpServer: httpServer,
		postgresDB: postgresDB,
		rabbitMQ:   rabbitMq,
		consumers: Consumers{
			rideConsumer: rideConsumer,
			uc:           driverService,
			log:          log,
		},
		cfg: cfg,
		log: log,
	}, nil
}

func (s *DriverService) Start(ctx context.Context) error {
	errCh := make(chan error, 2)

	s.httpServer.Run(ctx, errCh)
	s.consumers.Start(ctx, errCh)
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
