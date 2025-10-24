package models

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
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

/* ======================= service ======================= */

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address,omitempty"`
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
	Priority             int

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

/* ======================= rabbitmq ======================= */

type LocationMessage struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address,omitempty"`
}

type RideRequestedMessage struct {
	RideID              uuid.UUID       `json:"ride_id"`
	RideNumber          string          `json:"ride_number"`
	PickupLocation      LocationMessage `json:"pickup_location"`
	DestinationLocation LocationMessage `json:"destination_location"`
	RideType            string          `json:"ride_type"`
	EstimatedFare       float64         `json:"estimated_fare"`
	MaxDistanceKm       float64         `json:"max_distance_km"`
	TimeoutSeconds      int             `json:"timeout_seconds"`
	CorrelationID       string          `json:"correlation_id"`
}

type RideStatusUpdateMessage struct {
	RideID        uuid.UUID  `json:"ride_id"`
	Status        string     `json:"status"`
	Timestamp     time.Time  `json:"timestamp"`
	DriverID      *uuid.UUID `json:"driver_id,omitempty"`
	CorrelationID string     `json:"correlation_id"`
	FinalFare     float64    `json:"final_fare"`
}

/* ======================= Websocket ======================= */

type RideOffer struct {
	ID                          uuid.UUID       `json:"offer_id"`
	MsgType                     string          `json:"type"` // By default must be: "ride_offer"
	RideID                      uuid.UUID       `json:"ride_id"`
	RideNumber                  string          `json:"ride_number"`
	PickupLocation              LocationMessage `json:"pickup_location"`
	DestinationLocation         LocationMessage `json:"destination_location"`
	EstimatedFare               float64         `json:"estimated_fare"`
	DriverEarnings              float64         `json:"driver_earnings"`
	DistanceToPickupKm          float64         `json:"distance_to_pickup_km"`
	EstimatedRideDurationMinute int             `json:"estimated_ride_duration_minutes"`
	ExpiresAt                   time.Time       `json:"expires_at"`
}

type RideOfferResponse struct {
	ID              uuid.UUID       `json:"offer_id"`
	RideID          uuid.UUID       `json:"ride_id"`
	Accepted        bool            `json:"accepted"`
	CurrentLocation LocationMessage `json:"current_location"`
}
