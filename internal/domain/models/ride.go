package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

type RideInfo struct {
	RideID                uuid.UUID `json:"ride_id"`
	RideNumber            string    `json:"ride_number"`
	Status                string    `json:"status"`
	PassengerID           uuid.UUID `json:"passenger_id"`
	DriverID              uuid.UUID `json:"driver_id"`
	PickupAddress         string    `json:"pickup_address"`
	DestinationAddress    string    `json:"destination_address"`
	StartedAt             time.Time `json:"started_at"`
	EstimatedCompletion   time.Time `json:"estimated_completion"`
	CurrentDriverLocation Location  `json:"current_driver_location"`
	DistanceCompletedKm   float64   `json:"distance_completed_km"`
	DistanceRemainingKm   float64   `json:"distance_remaining_km"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
