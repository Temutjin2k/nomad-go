package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
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
	DestinationLocation   Location  `json:"destination_location"`
	DistanceCompletedKm   float64   `json:"distance_completed_km"`
	DistanceRemainingKm   float64   `json:"distance_remaining_km"`
}

type RideEvent struct {
	OldStatus        types.RideStatus `json:"old_status"`
	NewStatus        types.RideStatus `json:"new_status"`
	DriverID         uuid.UUID        `json:"driver_id"`
	Location         Location         `json:"location"`
	EstimatedArrival time.Time        `json:"estimated_arrival"`
}

type RideStatusUpdateMessage struct {
	RideID        uuid.UUID        `json:"ride_id"`
	Status        types.RideStatus `json:"status"`
	Timestamp     time.Time        `json:"timestamp"`
	FinalFare     float64          `json:"final_fare,omitempty"`
	CorrelationID string           `json:"correlation_id,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Ride struct {
	ID          uuid.UUID
	RideNumber  string
	Status      string
	PassengerID uuid.UUID
	RideType    string
	Pickup      Location
	Destination Location
	DriverID    *uuid.UUID

	// Расчетные поля
	EstimatedFare        float64
	EstimatedDurationMin int
	EstimatedDistanceKm  float64

	// Финальная стоимость.
	FinalFare *float64

	// Причина отмены, есть только у отмененных поездок
	CancellationReason *string

	// Временные метки
	CreatedAt   time.Time
	MatchedAt   *time.Time
	ArrivedAt   *time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	CancelledAt *time.Time
}
