package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type Driver struct {
	ID            uuid.UUID          // unique identifier
	Name          string             // full name of the driver
	CreatedAt     time.Time          // timestamp of creation
	UpdatedAt     time.Time          // timestamp of last update
	LicenseNumber string             // driver's license number
	Vehicle       Vehicle            // embedded struct for vehicle details
	Rating        float64            // average rating from passengers
	TotalRides    int                // number of completed rides
	TotalEarnings int                // in smallest currency unit, e.g., tyin for KZT, cents for USD
	Status        types.DriverStatus // e.g., "available", "on_trip", "offline"
	IsVerified    bool               // Indicates if the driver's documents have been verified
}

type Vehicle struct {
	Type  types.VehicleClass
	Make  string `json:"make"`
	Model string `json:"model"`
	Color string `json:"color"`
	Plate string `json:"plate"`
	Year  int    `json:"year"`
}

// DriverStatusUpdateMessage — структура сообщения для обновления статуса водителя
type DriverStatusUpdateMessage struct {
	DriverID  uuid.UUID          `json:"driver_id"`
	Status    types.DriverStatus `json:"status"`
	RideID    string             `json:"ride_id,omitempty"`
	Timestamp string             `json:"timestamp"`
}
