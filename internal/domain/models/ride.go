package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

/* ======================= for admin service ======================= */

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
	CurrentDriverLocation LocationInfo  `json:"current_driver_location"`
	DistanceCompletedKm   float64   `json:"distance_completed_km"`
	DistanceRemainingKm   float64   `json:"distance_remaining_km"`
}

type LocationInfo struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

/* ======================= service ======================= */

type Location struct {
    Latitude  float64
    Longitude float64
    Address   string
}

type Ride struct {
    ID          uuid.UUID
    RideNumber  string
    Status      string
    PassengerID uuid.UUID
    RideType    string
    Pickup      Location
    Destination Location
    DriverID *uuid.UUID

    // Расчетные поля
    EstimatedFare        float64
    EstimatedDurationMin int
    EstimatedDistanceKm  float64

    // Финальная стоимость. 
    FinalFare *float64

    // Причина отмены, есть только у отмененных поездок
    CancellationReason *string

    // Временные метки
    CreatedAt    time.Time 
    MatchedAt    *time.Time
    ArrivedAt    *time.Time
    StartedAt    *time.Time
    CompletedAt  *time.Time
    CancelledAt  *time.Time
}