package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

// PassengerLocationUpdateDTO - это DTO для WebSocket-сообщения,
// отправляемого пассажиру.
type PassengerLocationUpdateDTO struct {
	Type               string          `json:"type"`
	RideID             uuid.UUID       `json:"ride_id"`
	DriverLocation     LocationMessage `json:"driver_location"`
	EstimatedArrival   time.Time       `json:"estimated_arrival"`
	DistanceToPickupKm float64         `json:"distance_to_pickup_km"`
}