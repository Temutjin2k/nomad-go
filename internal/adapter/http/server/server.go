package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Temutjin2k/ride-hail-system/config"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/handler"
	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/middleware"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
)

type (
	API struct {
		server *http.Server
		log    logger.Logger
	}

	handlers struct {
		ride   *handler.Ride
		driver *handler.Driver
		admin  *handler.Admin
		auth   *handler.Auth

		health *handler.Health
	}
)

func New(
	ctx context.Context,
	cfg config.Config,
	driverService *handler.DriverServiceOptions,
	rideService handler.RideService,
	adminService handler.AdminService,
	authService handler.AuthService,
	wshub handler.ConnectionHub,
	logger logger.Logger,
) (*API, error) {
	if authService == nil {
		return nil, errors.New("auth service is required")
	}

	handlers := newHandlers(cfg,
		driverService,
		rideService,
		adminService,
		authService,
		wshub,
		logger,
	)

	mux := http.NewServeMux()
	m := middleware.NewMiddleware(authService, logger)

	setupRoutes(mux, handlers, m, cfg.Mode, logger)

	api := &API{
		server: &http.Server{
			Addr:    serverAddress(cfg),
			Handler: withMiddleware(mux, m, cfg.Mode),
		},
		log: logger,
	}

	return api, nil
}

func (a *API) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ctx = wrap.WithAction(ctx, "http_server_stop")

	a.log.Debug(ctx, "shutting down HTTP server...", "address", a.server.Addr)
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}
	a.log.Debug(ctx, "shutting down HTTP server completed")

	return nil
}

func (a *API) Run(ctx context.Context, errCh chan<- error) {
	go func() {
		ctx = wrap.WithAction(ctx, "http_server_start")
		a.log.Info(ctx, "started http server", "address", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to start HTTP server: %w", err)
			return
		}
	}()
}

// withMiddleware applies middlewares to the mux
func withMiddleware(mux *http.ServeMux, m *middleware.Middleware, mode types.ServiceMode) http.Handler {
	serviceName := mode.String()

	var handler http.Handler = mux
	handler = m.Auth(handler)
	handler = m.Metrics(serviceName)(handler)
	handler = m.RequestID(handler)
	handler = m.Recover(handler)

	return handler
}

func serverAddress(cfg config.Config) string {
	serverIPAddress := "%s:%s"
	switch cfg.Mode {
	case types.RideService:
		return fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.RideService)
	case types.DriverAndLocationService:
		return fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.DriverLocationService)
	case types.AdminService:
		return fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.AdminService)
	case types.AuthService:
		return fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.AuthService)
	}

	return ""
}

func newHandlers(
	cfg config.Config,
	driverService *handler.DriverServiceOptions,
	rideService handler.RideService,
	adminService handler.AdminService,
	authService handler.AuthService,
	wshub handler.ConnectionHub,
	logger logger.Logger,
) *handlers {
	return &handlers{
		ride:   handler.NewRide(rideService, authService, wshub, logger),
		driver: handler.NewDriver(driverService, logger),
		admin:  handler.NewAdmin(adminService, logger),
		auth:   handler.NewAuth(authService, logger),
		health: handler.NewHealth(cfg.Mode.String(), logger),
	}
}
