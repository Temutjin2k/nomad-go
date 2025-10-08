package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/app/microservices"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

var (
	ErrInvalidMode           = errors.New("invalid mode")
	ErrServiceNotInitialized = errors.New("service not initialized")
)

type Service interface {
	Start(ctx context.Context) error
}

type App struct {
	mode    types.ServiceMode
	service Service

	cfg config.Config
	log logger.Logger
}

// NewApplication
func NewApplication(ctx context.Context, cfg config.Config, log logger.Logger) (*App, error) {
	app := &App{
		mode: cfg.Mode,
		cfg:  cfg,
		log:  log,
	}

	if err := app.initService(ctx, app.mode); err != nil {
		return nil, err
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.service == nil {
		return ErrServiceNotInitialized
	}

	if err := a.service.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (a *App) initService(ctx context.Context, mode types.ServiceMode) error {
	var (
		service Service
		err     error
	)
	switch mode {
	case types.RideService:
		service, err = microservices.NewRide(ctx, a.cfg, a.log)
	case types.DriverAndLocationService:
		service, err = microservices.NewDriver(ctx, a.cfg, a.log)
	case types.AdminService:
		service, err = microservices.NewAdmin(ctx, a.cfg, a.log)
	case types.AuthService:
		service, err = microservices.NewAuth(ctx, a.cfg, a.log)
	default:
		return ErrInvalidMode
	}

	if err != nil {
		return fmt.Errorf("failed to init service: %w", err)
	}
	if service == nil {
		return fmt.Errorf("failed to initialize: %s", mode)
	}

	a.service = service

	return nil
}
