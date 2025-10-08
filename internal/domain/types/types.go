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

// Enum для классов
type VehicleClass string

const (
	EconomyClass VehicleClass = "ECONOMY"
	PremiumClass VehicleClass = "PREMIUM"
	XLClass      VehicleClass = "XL"
)

// Enum для статуса водителя
type DriverStatus string

const (
	OfflineStatus   DriverStatus = "OFFLINE"
	AvailableStatus DriverStatus = "AVAILABLE"
	BusyStatus      DriverStatus = "BUSY"
	EnRouteStatus   DriverStatus = "EN_ROUTE"
)

// Enum для статуса пользователя
type UserStatus string

const (
	ActiveStatus   UserStatus = "ACTIVE"
	InActiveStatus UserStatus = "INACTIVE"
	BannedStatus   UserStatus = "BANNED"
)

// Enum для роли пользователя
type UserRole string

func (r UserRole) String() string {
	return string(r)
}

const (
	PassengerRole UserRole = "PASSENGER"
	DriverRole    UserRole = "DRIVER"
	AdminRole     UserRole = "ADMIN"
)

// Enum для типов пользователей
type EntityType string

const (
	Driver    EntityType = "driver"
	Passenger EntityType = "passenger"
)
