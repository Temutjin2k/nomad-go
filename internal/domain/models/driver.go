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
	TotalEarnings float64            // in smallest currency unit, e.g., tyin for KZT, cents for USD
	Status        types.DriverStatus // e.g., "available", "on_trip", "offline"
	IsVerified    bool               // Indicates if the driver's documents have been verified
}

// DriverWithDistance представляет водителя с координатами и расстоянием до точки
type DriverWithDistance struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	Rating     float64         `json:"rating"`
	Location   LocationMessage `json:"location"`
	Vehicle    Vehicle         `json:"vehicle"`
	DistanceKm float64         `json:"distance_km"`
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
	RideID    uuid.UUID          `json:"ride_id,omitempty"`
	Timestamp time.Time          `json:"timestamp"`
}

type DriverInfo struct {
	Name    string  `json:"name"`
	Rating  float64 `json:"rating"`
	Vehicle Vehicle `json:"vehicle"`
}

type DriverMatchResponse struct {
	RideID                  uuid.UUID       `json:"ride_id"`
	DriverID                uuid.UUID       `json:"driver_id"`
	Accepted                bool            `json:"accepted"`
	EstimatedArrivalMinutes int             `json:"estimated_arrival_minutes"`
	DriverLocation          LocationMessage `json:"driver_location"`
	DriverInfo              DriverInfo      `json:"driver_info"`
	CorrelationID           string          `json:"correlation_id"`
}

type DriverLocationUpdate struct {
	Type           string    `json:"type"`            // тип сообщения, например "location_update"
	RideID         uuid.UUID `json:"ride_id"`         // идентификатор поездки
	Latitude       float64   `json:"latitude"`        // широта
	Longitude      float64   `json:"longitude"`       // долгота
	AccuracyMeters float64   `json:"accuracy_meters"` // точность GPS
	SpeedKmh       float64   `json:"speed_kmh"`       // скорость, км/ч
	HeadingDegrees float64   `json:"heading_degrees"` // направление движения
}
