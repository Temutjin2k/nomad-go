package microservices

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Temutjin2k/ride-hail-system/config"
	httpserver "github.com/Temutjin2k/ride-hail-system/internal/adapter/http/server"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/postgres"
	"github.com/Temutjin2k/ride-hail-system/internal/service/admin"
	"github.com/Temutjin2k/ride-hail-system/internal/service/auth"
	ridecalc "github.com/Temutjin2k/ride-hail-system/internal/service/calculator"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	postgresclient "github.com/Temutjin2k/ride-hail-system/pkg/postgres"
	"github.com/Temutjin2k/ride-hail-system/pkg/trm"
)

type AdminService struct {
	postgresDB *postgresclient.PostgreDB
	httpServer *httpserver.API

	cfg config.Config
	log logger.Logger
}

func NewAdmin(ctx context.Context, cfg config.Config, log logger.Logger) (*AdminService, error) {
	db, err := postgresclient.New(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	// repositories
	adminRepo := postgres.NewAdminRepo(db.Pool)
	userRepo := postgres.NewUserRepo(db.Pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepo(db.Pool)

	// services
	calculator := ridecalc.New()
	adminSvc := admin.NewAdminService(adminRepo, calculator, log)
	txManager := trm.New(db.Pool)
	tokenSvc := auth.NewTokenService(cfg.Auth.JWTSecret, userRepo, refreshTokenRepo, txManager, cfg.Auth.RefreshTokenTTL, cfg.Auth.AccessTokenTTL, log)
	authSvc := auth.NewAuthService(userRepo, tokenSvc, log)

	server, err := httpserver.New(cfg, nil, nil, adminSvc, authSvc, log)
	if err != nil {
		return nil, err
	}

	return &AdminService{
		postgresDB: db,
		httpServer: server,
		cfg:        cfg,
		log:        log,
	}, nil
}

func (s *AdminService) Start(ctx context.Context) error {
	defer func() {
		s.close(ctx)
		s.log.Info(ctx, "admin service closed")
	}()

	errCh := make(chan error, 1)
	s.httpServer.Run(ctx, errCh)

	// Waiting signal
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info(ctx, "service started")
	select {
	case errRun := <-errCh:
		return errRun
	case sig := <-shutdownCh:
		s.log.Info(ctx, "shuting down application", "signal", sig.String())
		return nil
	}
}

func (s *AdminService) close(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if err := s.httpServer.Stop(ctx); err != nil {
		s.log.Error(ctx, "failed to shutdown HTTP server", err)
	}

	s.postgresDB.Pool.Close()
}
