package types

type ServiceMode string

// Ride Service - Orchestrates the complete ride lifecycle and manages passenger interactions
// Driver & Location Service - Handles driver operations, matching algorithms, and real-time location tracking
// Admin Service - Provides monitoring, analytics, and system oversight capabilities
const (
	RideService              ServiceMode = "ride-service"
	DriverAndLocationService ServiceMode = "driver-service"
	AdminService             ServiceMode = "admin-service"
)
