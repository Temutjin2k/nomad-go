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

const serverIPAddress = "%s:%s"

type API struct {
	mode   types.ServiceMode
	mux    *http.ServeMux
	server *http.Server
	routes *handlers // routes/handlers
	m      *middleware.Middleware

	addr string
	cfg  config.Config
	log  logger.Logger
}

type handlers struct {
	ride   *handler.Ride
	driver *handler.Driver
	admin  *handler.Admin
	auth   *handler.Auth
}

func New(
	cfg config.Config,
	driverService handler.DriverService,
	rideService handler.RideService,
	adminService handler.AdminService,
	authService handler.AuthService,
	logger logger.Logger,
) (*API, error) {
	var addr string
	handlers := &handlers{}

	if authService == nil {
		return nil, errors.New("auth service is required")
	}

	switch cfg.Mode {
	case types.RideService:
		addr = fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.RideService)
		handlers.ride = handler.NewRide(logger, rideService)
	case types.DriverAndLocationService:
		addr = fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.DriverLocationService)
		handlers.driver = handler.NewDriver(driverService, logger)
	case types.AdminService:
		addr = fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.AdminService)
		handlers.admin = handler.NewAdmin(adminService, logger)
	case types.AuthService:
		addr = fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.Services.AuthService)
		handlers.auth = handler.NewAuth(authService, logger)
	default:
		return nil, fmt.Errorf("invalid mode: %s", cfg.Mode)
	}

	mid := middleware.NewMiddleware(authService, logger)

	api := &API{
		mode: cfg.Mode,

		mux:    http.NewServeMux(),
		routes: handlers,
		m:      mid,
		addr:   addr,
		cfg:    cfg,
		log:    logger,
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
	ctx = wrap.WithAction(ctx, "http_server_stop")

	a.log.Debug(ctx, "shutting down HTTP server...", "address", a.addr)
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}
	a.log.Debug(ctx, "shutting down HTTP server completed")

	return nil
}

func (a *API) Run(ctx context.Context, errCh chan<- error) {
	go func() {
		ctx = wrap.WithAction(ctx, "http_server_start")
		a.log.Info(ctx, "started http server", "address", a.addr)
		if err := http.ListenAndServe(a.addr, a.withMiddleware()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to start HTTP server: %w", err)
			return
		}
	}()
}

// withMiddleware applies middlewares to the mux
func (a *API) withMiddleware() http.Handler {
	return a.m.Recover(a.m.RequestID(a.m.Auth(a.mux)))
}
