package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

var ErrInvalidMode = errors.New("invalid mode")

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

func (app *App) Run(ctx context.Context) error {
	if app.service == nil {
		if err := app.initService(ctx, app.mode); err != nil {
			return err
		}
	}

	if err := app.service.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (app *App) initService(ctx context.Context, mode types.ServiceMode) error {
	var (
		service Service
		err     error
	)
	switch mode {
	case types.RideService:
		app.log.Info(ctx, "", "started service")
	case types.DriverAndLocationService:
		app.log.Info(ctx, "", "started service")
	case types.AdminService:
		app.log.Info(ctx, "", "started service")
	default:
		return ErrInvalidMode
	}

	if err != nil {
		return fmt.Errorf("failed to init service: %w", err)
	}
	if service == nil {
		return fmt.Errorf("failed to initialize: %s", mode)
	}

	app.service = service

	return nil
}
