package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
)

const serverIPAddress = "%s:%s"

type API struct {
	mode   types.ServiceMode
	mux    *http.ServeMux
	server *http.Server
	routes *handlers // routes/handlers

	addr string
	cfg  config.Config
	log  logger.Logger
}

type handlers struct {
	ride   *handler.Ride
	driver *handler.Driver
	admin  *handler.Admin
}

func New(cfg config.Config, driverService handler.DriverService, logger logger.Logger) (*API, error) {
	var addr string
	handlers := &handlers{}

	switch cfg.Mode {
	case types.RideService:
		addr = fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.RideService)
		handlers.ride = handler.NewRide(logger)
	case types.DriverAndLocationService:
		addr = fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.DriverLocationService)
		handlers.driver = handler.NewDriver(driverService, logger)
	case types.AdminService:
		addr = fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.AdminService)
		handlers.admin = handler.NewAdmin(logger)
	default:
		return nil, fmt.Errorf("invalid mode: %s", cfg.Mode)
	}

	api := &API{
		mode: cfg.Mode,

		mux:    http.NewServeMux(),
		routes: handlers,

		addr: addr,
		cfg:  cfg,
		log:  logger,
	}

	api.server = &http.Server{
		Addr:    api.addr,
		Handler: api.mux,
	}

	api.setupRoutes()

	return api, nil
}

func (a *API) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ctx = logger.WithAction(ctx, "http_server_stop")

	a.log.Debug(ctx, "shutting down HTTP server...", "address", a.addr)
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}
	a.log.Debug(ctx, "shutting down HTTP server completed")

	return nil
}

func (a *API) Run(ctx context.Context, errCh chan<- error) {
	go func() {
		ctx = logger.WithAction(ctx, "http_server_start")
		a.log.Info(ctx, "started http server", "address", a.addr)
		if err := http.ListenAndServe(a.addr, a.withMiddleware()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to start HTTP server: %w", err)
			return
		}
	}()
}
