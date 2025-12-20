package server

import (
	"context"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/adapter/http/middleware"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

// setupRoutes - setups http routes
func setupRoutes(mux *http.ServeMux, routes *handlers, m *middleware.Middleware, mode types.ServiceMode, log logger.Logger) {
	// System Health
	mux.HandleFunc("/health", routes.health.HealthCheck)

	setupSwaggerRoutes(mux, mode, log)
	setupMetricsRoute(mux)

	switch mode {
	case types.AdminService:
		setupAdminRoutes(mux, routes, m)
	case types.RideService:
		setupRideRoutes(mux, routes, m)
	case types.DriverAndLocationService:
		setupDriverAndLocationRoutes(mux, routes, m)
	case types.AuthService:
		setupAuthRoutes(mux, routes)
	}
}

// setupAdminRoutes setups routes for admin service
func setupAdminRoutes(mux *http.ServeMux, routes *handlers, m *middleware.Middleware) {
	mux.Handle("GET /admin/overview", m.RequireRoles(routes.admin.GetOverview, types.RoleAdmin))        // Get system metrics overview
	mux.Handle("GET /admin/rides/active", m.RequireRoles(routes.admin.GetActiveRides, types.RoleAdmin)) // Get list of active rides
}

// setupRideRoutes setups routes for ride service
func setupRideRoutes(mux *http.ServeMux, routes *handlers, m *middleware.Middleware) {
	mux.Handle("POST /rides", m.RequireRoles(routes.ride.CreateRide, types.RolePassenger))                  // Create a new ride request
	mux.Handle("POST /rides/{ride_id}/cancel", m.RequireRoles(routes.ride.CancelRide, types.RolePassenger)) // Cancel a ride
	mux.HandleFunc("GET /ws/passengers/{passenger_id}", routes.ride.HandleWebSocket)                        // WebSocket connection for passengers
}

// setupDriverAndLocationRoutes setups routes for driver and location service
func setupDriverAndLocationRoutes(mux *http.ServeMux, routes *handlers, m *middleware.Middleware) {
	mux.HandleFunc("POST /drivers", routes.driver.Register)
	mux.Handle("POST /drivers/{driver_id}/online", m.RequireRoles(routes.driver.GoOnline, types.RoleDriver))         // Driver goes online
	mux.Handle("POST /drivers/{driver_id}/offline", m.RequireRoles(routes.driver.GoOffline, types.RoleDriver))       // Driver goes offline
	mux.Handle("POST /drivers/{driver_id}/location", m.RequireRoles(routes.driver.UpdateLocation, types.RoleDriver)) // Update driver location
	mux.Handle("POST /drivers/{driver_id}/start", m.RequireRoles(routes.driver.StartRide, types.RoleDriver))         // Start a ride
	mux.Handle("POST /drivers/{driver_id}/complete", m.RequireRoles(routes.driver.CompleteRide, types.RoleDriver))   // Complete a ride
	mux.HandleFunc("GET /ws/drivers/{driver_id}", routes.driver.HandleWS)                                            // WebSocket connection for drivers
}

func setupAuthRoutes(mux *http.ServeMux, routes *handlers) {
	mux.HandleFunc("POST /auth/register", routes.auth.Register)
	mux.HandleFunc("POST /auth/login", routes.auth.Login)
	mux.HandleFunc("POST /auth/refresh", routes.auth.Refresh)
	mux.HandleFunc("GET /auth/me", routes.auth.Profile)
}

// setupSwaggerRoutes configures Swagger UI endpoints based on service mode
func setupSwaggerRoutes(mux *http.ServeMux, mode types.ServiceMode, log logger.Logger) {
	var instanceName string

	switch mode {
	case types.RideService:
		instanceName = "ride"
	case types.DriverAndLocationService:
		instanceName = "driver"
	case types.AdminService:
		instanceName = "admin"
	case types.AuthService:
		instanceName = "auth"
	default:
		log.Warn(wrap.WithAction(context.Background(), "setup swagger routes"), "unknown service mode for swagger setup", "mode", mode)
		return
	}

	// Swagger UI endpoint
	swaggerURL := httpSwagger.InstanceName(instanceName)
	mux.HandleFunc("/swagger/", httpSwagger.Handler(swaggerURL))
}

// setupMetricsRoute configures the Prometheus metrics endpoint
func setupMetricsRoute(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())
}
