package types

type ServiceMode string

// Ride Service - Orchestrates the complete ride lifecycle and manages passenger interactions
// Driver & Location Service - Handles driver operations, matching algorithms, and real-time location tracking
// Admin Service - Provides monitoring, analytics, and system oversight capabilities
const (
	RideService              ServiceMode = "ride-service"
	DriverAndLocationService ServiceMode = "driver-service"
	AdminService             ServiceMode = "admin-service"
	AuthService              ServiceMode = "auth-service"
)

// Enum для классов
type VehicleClass string

const (
	ClassEconomy VehicleClass = "ECONOMY"
	ClassPremium VehicleClass = "PREMIUM"
	ClassXL      VehicleClass = "XL"
)

// Enum для статуса водителя
type DriverStatus string

const (
	StatusDriverOffline   DriverStatus = "OFFLINE"
	StatusDriverAvailable DriverStatus = "AVAILABLE"
	StatusDriverBusy      DriverStatus = "BUSY"
	StatusDriverEnRoute   DriverStatus = "EN_ROUTE"
)

// Enum для статуса пользователя
type UserStatus string

func (s UserStatus) String() string {
	return string(s)
}

const (
	StatusUserActive   UserStatus = "ACTIVE"
	StatusUserInactive UserStatus = "INACTIVE"
	StatusUserBanned   UserStatus = "BANNED"
)

// Enum для роли пользователя
type UserRole string

func (r UserRole) String() string {
	return string(r)
}

const (
	RolePassenger UserRole = "PASSENGER"
	RoleDriver    UserRole = "DRIVER"
	RoleAdmin     UserRole = "ADMIN"
)

// Enum для типов пользователей
type EntityType string

const (
	Driver    EntityType = "driver"
	Passenger EntityType = "passenger"
)
