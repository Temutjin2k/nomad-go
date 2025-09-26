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
	}
}

// setupAdminRoutes setups routes for admin service
func (a *API) setupAdminRoutes() {
	a.mux.HandleFunc("GET /admin/overview", nil)     // Get system metrics overview
	a.mux.HandleFunc("GET /admin/rides/active", nil) // Get list of active rides
}

// setupRideRoutes setups routes for ride service
func (a *API) setupRideRoutes() {
	a.mux.HandleFunc("POST /rides", nil)                       // Create a new ride request
	a.mux.HandleFunc("POST /rides/{ride_id}/cancel", nil)      // Cancel a ride
	a.mux.HandleFunc("GET /ws/passengers/{passenger_id}", nil) // WebSocket connection for passengers
}

// setupDriverAndLocationRoutes setups routes for driver and location service
func (a *API) setupDriverAndLocationRoutes() {
	a.mux.HandleFunc("POST /drivers/{driver_id}/online", nil)   // Driver goes online
	a.mux.HandleFunc("POST /drivers/{driver_id}/offline", nil)  // Driver goes offline
	a.mux.HandleFunc("POST /drivers/{driver_id}/location", nil) // Update driver location
	a.mux.HandleFunc("POST /drivers/{driver_id}/start", nil)    // Start a ride
	a.mux.HandleFunc("POST /drivers/{driver_id}/complete", nil) // Complete a ride
	a.mux.HandleFunc("GET /ws/drivers/{driver_id}", nil)        // WebSocket connection for drivers
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
