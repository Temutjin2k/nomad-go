package dto

import (
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/Temutjin2k/ride-hail-system/pkg/validator"
)

type CreateRideRequest struct {
	PassengerID          string  `json:"passenger_id"`
	PickupLatitude       float64 `json:"pickup_latitude"`
	PickupLongitude      float64 `json:"pickup_longitude"`
	PickupAddress        string  `json:"pickup_address"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationAddress   string  `json:"destination_address"`
	RideType             string  `json:"ride_type"`
}

// для создания поездки
func (r *CreateRideRequest) Validate(v *validator.Validator) {
	// PassengerID
	v.Check(r.PassengerID != "", "passenger_id", "must be provided")
	if r.PassengerID != "" {
		_, err := uuid.Parse(r.PassengerID)
		v.Check(err == nil, "passenger_id", "must be a valid UUID")
	}

	// Pickup Location
	v.Check(r.PickupAddress != "", "pickup_address", "must be provided")
	v.Check(len(r.PickupAddress) <= 255, "pickup_address", "must not be more than 255 characters long")
	v.Check(r.PickupLatitude >= -90 && r.PickupLatitude <= 90, "pickup_latitude", "must be between -90 and 90")
	v.Check(r.PickupLongitude >= -180 && r.PickupLongitude <= 180, "pickup_longitude", "must be between -180 and 180")

	// Destination Location
	v.Check(r.DestinationAddress != "", "destination_address", "must be provided")
	v.Check(len(r.DestinationAddress) <= 255, "destination_address", "must not be more than 255 characters long")
	v.Check(r.DestinationLatitude >= -90 && r.DestinationLatitude <= 90, "destination_latitude", "must be between -90 and 90")
	v.Check(r.DestinationLongitude >= -180 && r.DestinationLongitude <= 180, "destination_longitude", "must be between -180 and 180")

	// RideType
	v.Check(r.RideType != "", "ride_type", "must be provided")
	if r.RideType != "" {
		v.Check(validator.PermittedValue(r.RideType, "ECONOMY", "PREMIUM", "XL"), "ride_type", "must be one of ECONOMY, PREMIUM, or XL")
	}
}


type CreateRideResponse struct {
	RideID               uuid.UUID `json:"ride_id"`
	RideNumber           string    `json:"ride_number"`
	Status               string    `json:"status"`
	Estimated_fare       float64   `json:"estimated_fare"`
	EstimatedDurationMin int       `json:"estimated_duration_minutes"`
	EstimatedDistanceKm  float64   `json:"estimated_distance_km"`
}

type CancelRideRequest struct {
	Reason string `json:"reason"`
}

// для отмены поездки
func (r *CancelRideRequest) Validate(v *validator.Validator) {
	v.Check(r.Reason != "", "reason", "must be provided")
	v.Check(len(r.Reason) <= 500, "reason", "must not be more than 500 characters long")
}


type CancelRideResponse struct {
	RideID      uuid.UUID `json:"ride_id"`
	Status      string    `json:"status"`
	CancelledAt time.Time `json:"cancelled_at"`
	Message     string    `json:"message"`
}

func (r *CreateRideRequest) ToModel() (*models.Ride, error) {
	passengerUUID, err := uuid.Parse(r.PassengerID)
	if err != nil {
		return nil, err
	}

	return &models.Ride{
		PassengerID: passengerUUID,
		RideType:    r.RideType,
		Pickup: models.Location{
			Latitude:  r.PickupLatitude,
			Longitude: r.PickupLongitude,
			Address:   r.PickupAddress,
		},
		Destination: models.Location{
			Latitude:  r.DestinationLatitude,
			Longitude: r.DestinationLongitude,
			Address:   r.DestinationAddress,
		},
	}, nil
}
