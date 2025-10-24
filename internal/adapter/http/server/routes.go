package server

import (
	"encoding/json"
	"net/http"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
)

// setupRoutes - setups http routes
func (a *API) setupRoutes() {
	// System Health
	a.mux.HandleFunc("/health", a.HealthCheck)

	switch a.mode {
	case types.AdminService:
		a.setupAdminRoutes()
	case types.RideService:
		a.setupRideRoutes()
	case types.DriverAndLocationService:
		a.setupDriverAndLocationRoutes()
	case types.AuthService:
		a.SetupAuthRoutes()
	}
}

// setupAdminRoutes setups routes for admin service
func (a *API) setupAdminRoutes() {
	a.mux.Handle("GET /admin/overview", a.m.RequireRoles(a.routes.admin.GetOverview, types.RoleAdmin))        // Get system metrics overview
	a.mux.Handle("GET /admin/rides/active", a.m.RequireRoles(a.routes.admin.GetActiveRides, types.RoleAdmin)) // Get list of active rides
}

// setupRideRoutes setups routes for ride service
func (a *API) setupRideRoutes() {
	a.mux.HandleFunc("POST /rides", nil)                       // Create a new ride request
	a.mux.HandleFunc("POST /rides/{ride_id}/cancel", nil)      // Cancel a ride
	a.mux.HandleFunc("GET /ws/passengers/{passenger_id}", nil) // WebSocket connection for passengers
}

// setupDriverAndLocationRoutes setups routes for driver and location service
func (a *API) setupDriverAndLocationRoutes() {
	a.mux.HandleFunc("POST /drivers", a.routes.driver.Register)
	a.mux.HandleFunc("POST /drivers/{driver_id}/online", a.routes.driver.GoOnline)         // Driver goes online
	a.mux.HandleFunc("POST /drivers/{driver_id}/offline", a.routes.driver.GoOffline)       // Driver goes offline
	a.mux.HandleFunc("POST /drivers/{driver_id}/location", a.routes.driver.UpdateLocation) // Update driver location
	a.mux.HandleFunc("POST /drivers/{driver_id}/start", a.routes.driver.StartRide)         // Start a ride
	a.mux.HandleFunc("POST /drivers/{driver_id}/complete", a.routes.driver.CompleteRide)   // Complete a ride
	a.mux.HandleFunc("GET /ws/drivers/{driver_id}", a.routes.driver.HandleWS)              // WebSocket connection for drivers
}

func (a *API) SetupAuthRoutes() {
	a.mux.HandleFunc("POST /auth/register", a.routes.auth.Register)
	a.mux.HandleFunc("POST /auth/login", a.routes.auth.Login)
	a.mux.HandleFunc("POST /auth/refresh", a.routes.auth.Refresh)
	a.mux.HandleFunc("GET /auth/me", a.routes.auth.Profile)
}

// HealthCheck - returns system information.
func (a *API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status": "available",
		"system_info": map[string]string{
			"address": a.addr,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.log.Error(r.Context(), "healthcheck", err)
		return
	}
}
