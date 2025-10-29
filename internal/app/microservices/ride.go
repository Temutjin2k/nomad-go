package microservices

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	// sync и cancel для корректного завершения
	wg     sync.WaitGroup
	cancel context.CancelFunc
	mu     sync.Mutex // защищает cancel от параллельных вызовов
}

func (c *RideConsumers) Start(parentCtx context.Context, errCh chan error) {
	// создаём дочерний контекст, который можно будет отменить через Stop
	ctx, cancel := context.WithCancel(parentCtx)
	c.mu.Lock()
	c.cancel = cancel
	c.mu.Unlock()

	// первая горутина
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.log.Info(ctx, "ConsumeDriverLocationUpdate has been started")
		if err := c.rideConsumer.ConsumeDriverLocationUpdate(ctx, c.rideService.HandleDriverLocationUpdate); err != nil {
			// пробрасываем ошибку в канал, если он ещё открыт
			select {
			case errCh <- fmt.Errorf("failed to start ConsumeDriverLocationUpdate: %w", err):
			default:
				// если канал полон/никто не слушает — просто залогируем
				c.log.Error(ctx, "ConsumeDriverLocationUpdate error, errCh blocked", err)
			}
			return
		}
		c.log.Info(ctx, "ConsumeDriverLocationUpdate has been finished")
	}()

	// вторая горутина
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.log.Info(ctx, "ConsumeDriverStatusUpdate has been started")
		if err := c.rideConsumer.ConsumeDriverStatusUpdate(ctx, c.rideService.HandleDriverStatusUpdate); err != nil {
			select {
			case errCh <- fmt.Errorf("failed to start ConsumeDriverStatusUpdate: %w", err):
			default:
				c.log.Error(ctx, "ConsumeDriverStatusUpdate error, errCh blocked", err)
			}
			return
		}
		c.log.Info(ctx, "ConsumeDriverStatusUpdate has been finished")
	}()
}

// Stop отменяет внутренний контекст и ждёт завершения горутин с заданным таймаутом.
// Возвращает ошибку, если ожидание превысило timeout.
func (c *RideConsumers) Stop(timeout time.Duration) error {
	c.mu.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	c.mu.Unlock()

	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.log.Info(context.Background(), "ride consumers stopped gracefully")
		return nil
	case <-time.After(timeout):
		c.log.Warn(context.Background(), "timeout while waiting for ride consumers to stop")
		return fmt.Errorf("timeout waiting for ride consumers to stop")
	}
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

	wsHub := ws.NewConnHub(log)
	wsRide := wshandler.NewRideWsHandler(wsHub)

	rideService := ridego.NewRideService(rideRepo, calculator, trm, rabbitRideBroker, wsRide, eventRepo, log)
	tokenSvc := auth.NewTokenService(cfg.Auth.JWTSecret, userRepo, refreshTokenRepo, trm, cfg.Auth.RefreshTokenTTL, cfg.Auth.AccessTokenTTL, log)
	authSvc := auth.NewAuthService(userRepo, tokenSvc, log)

	// init http server
	httpServer, err := httpserver.New(ctx, cfg, nil, rideService, nil, authSvc, wsHub, log)
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
		// тут не передаём ctx отменяемый — Stop сам отменит дочерний контекст потребителей
		if err := s.consumers.Stop(5 * time.Second); err != nil {
			s.log.Warn(ctx, "consumers stop error", "error", err.Error())
		}
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

	fmt.Println("closing rabbit")
	if s.rabbitMQ != nil {
		if err := s.rabbitMQ.Close(ctx); err != nil {
			s.log.Warn(ctx, "failed to close rabbitmq connection", "error", err.Error())
		}
	}

	fmt.Println("closing postgres")
	if s.postgresDB != nil && s.postgresDB.Pool != nil {
		s.postgresDB.Pool.Close()
	}

	fmt.Println("closed everything")
}
